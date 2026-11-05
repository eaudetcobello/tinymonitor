// Package client reads metrics from the tinymonitor HTTP API.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
)

const (
	// DefaultTimeout bounds a request when New is given a timeout <= 0.
	DefaultTimeout = 10 * time.Second
	// maxErrorBody caps how much of a non-2xx response body ends up in a StatusError.
	maxErrorBody = 4 << 10
)

var ErrInvalidBaseURL = errors.New("invalid base URL")

// Client calls the tinymonitor HTTP API.
type Client struct {
	metricsURL *url.URL
	httpClient *http.Client
}

// New returns a Client for the server at baseURL, which must be an http or https URL with a host.
// Each request is bounded by timeout, or DefaultTimeout when timeout <= 0.
func New(baseURL string, timeout time.Duration) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrInvalidBaseURL, baseURL, err)
	}

	if (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("%w %q: expected http(s)://host[:port]", ErrInvalidBaseURL, baseURL)
	}

	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		metricsURL: base.JoinPath("metrics"),
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// StatusError is returned when the server answers with a non-2xx status.
type StatusError struct {
	StatusCode int
	// Body is the start of the response body, trimmed of surrounding whitespace.
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status %d: %s", e.StatusCode, e.Body)
}

// Get returns the metrics of one kind for an agent with a time in [from, to], ordered by time ascending.
func (c *Client) Get(
	ctx context.Context, agentID uint32, kind model.Kind, from, to time.Time,
) ([]model.Metric, error) {
	u := *c.metricsURL
	u.RawQuery = url.Values{
		"agent_id": {strconv.FormatUint(uint64(agentID), 10)},
		"kind":     {string(kind)},
		"from":     {from.UTC().Format(time.RFC3339Nano)},
		"to":       {to.UTC().Format(time.RFC3339Nano)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get metrics: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))

		return nil, &StatusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	var metrics []model.Metric
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("decode metrics: %w", err)
	}

	if metrics == nil {
		metrics = make([]model.Metric, 0)
	}

	return metrics, nil
}
