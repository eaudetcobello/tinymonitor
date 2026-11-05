package client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/client"
	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/server"
)

// window returns the one-minute range used as from and to in most tests.
func window() (time.Time, time.Time) {
	from := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)

	return from, from.Add(time.Minute)
}

// serve starts an httptest server with handler and returns a client pointed at it.
func serve(t *testing.T, handler http.HandlerFunc, timeout time.Duration) *client.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, timeout)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	return c
}

func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestGetSendsQueryAndDecodesMetrics(t *testing.T) {
	t.Parallel()

	from, to := window()

	var gotMethod, gotPath string

	var gotQuery map[string]string

	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotQuery = map[string]string{}

		for k := range r.URL.Query() {
			gotQuery[k] = r.URL.Query().Get(k)
		}

		_, _ = w.Write([]byte(`[{"agent_id":7,"kind":"memory","time":"2026-11-04T10:00:30.0000005Z",` +
			`"values":{"used_pct":42.5}}]`))
	}, 0)

	est := time.FixedZone("EST", -5*3600)

	got, err := c.Get(t.Context(), 7, model.KindMemory, from.In(est), to.Add(500).In(est))
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if gotMethod != http.MethodGet || gotPath != "/metrics" {
		t.Errorf("request = %s %s, want GET /metrics", gotMethod, gotPath)
	}

	wantQuery := map[string]string{
		"agent_id": "7",
		"kind":     "memory",
		"from":     "2026-11-04T10:00:00Z",
		"to":       "2026-11-04T10:01:00.0000005Z",
	}
	if !reflect.DeepEqual(gotQuery, wantQuery) {
		t.Errorf("query = %v, want %v", gotQuery, wantQuery)
	}

	want := []model.Metric{{
		AgentID: 7,
		Kind:    model.KindMemory,
		Time:    time.Date(2026, 11, 4, 10, 0, 30, 500, time.UTC),
		Values:  map[string]float64{"used_pct": 42.5},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("metrics = %+v, want %+v", got, want)
	}
}

func TestGetEmptyResult(t *testing.T) {
	t.Parallel()

	from, to := window()

	for _, body := range []string{"[]", "null"} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			got, err := serve(t, respond(http.StatusOK, body), 0).Get(t.Context(), 7, model.KindCPU, from, to)
			if err != nil {
				t.Fatalf("get: %v", err)
			}

			if got == nil || len(got) != 0 {
				t.Errorf("metrics = %#v, want a non-nil empty slice", got)
			}
		})
	}
}

func TestGetErrorStatus(t *testing.T) {
	t.Parallel()

	from, to := window()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"bad request", http.StatusBadRequest, "missing kind\n"},
		{"server error", http.StatusInternalServerError, `{"message":"Internal Server Error"}`},
		{"not found", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := serve(t, respond(tt.status, tt.body), 0).Get(t.Context(), 7, model.KindCPU, from, to)

			statusErr, ok := errors.AsType[*client.StatusError](err)
			if !ok {
				t.Fatalf("err = %v, want *client.StatusError", err)
			}

			wantBody := strings.TrimSpace(tt.body)
			if statusErr.StatusCode != tt.status || statusErr.Body != wantBody {
				t.Errorf("StatusError = {%d %q}, want {%d %q}", statusErr.StatusCode, statusErr.Body, tt.status, wantBody)
			}

			if !strings.Contains(err.Error(), strconv.Itoa(tt.status)) || !strings.Contains(err.Error(), wantBody) {
				t.Errorf("error %q does not mention the status code and body", err)
			}
		})
	}
}

func TestGetErrorBodyIsTruncated(t *testing.T) {
	t.Parallel()

	from, to := window()

	_, err := serve(t, respond(http.StatusBadGateway, strings.Repeat("x", 1<<20)), 0).
		Get(t.Context(), 7, model.KindCPU, from, to)

	statusErr, ok := errors.AsType[*client.StatusError](err)
	if !ok {
		t.Fatalf("err = %v, want *client.StatusError", err)
	}

	if len(statusErr.Body) > 4096 {
		t.Errorf("body length = %d, want at most 4096", len(statusErr.Body))
	}
}

func TestGetMalformedJSON(t *testing.T) {
	t.Parallel()

	from, to := window()

	_, err := serve(t, respond(http.StatusOK, `[{"agent_id":`), 0).Get(t.Context(), 7, model.KindCPU, from, to)
	if err == nil || !strings.Contains(err.Error(), "decode metrics") {
		t.Errorf("err = %v, want a decode error", err)
	}
}

// blockUntilDone holds the request open until the client goes away.
func blockUntilDone(_ http.ResponseWriter, r *http.Request) {
	<-r.Context().Done()
}

func TestGetHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	from, to := window()

	c := serve(t, blockUntilDone, time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(20*time.Millisecond, cancel)

	_, err := c.Get(ctx, 7, model.KindCPU, from, to)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestGetTimesOut(t *testing.T) {
	t.Parallel()

	from, to := window()

	c := serve(t, blockUntilDone, 50*time.Millisecond)

	start := time.Now()

	_, err := c.Get(t.Context(), 7, model.KindCPU, from, to)
	if err == nil {
		t.Fatal("err = nil, want a timeout")
	}

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Get took %v, want it bounded by the 50ms timeout", elapsed)
	}
}

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	for _, baseURL := range []string{"", "localhost:1323", "ftp://example.com", "http://", "http://[::1"} {
		if _, err := client.New(baseURL, 0); !errors.Is(err, client.ErrInvalidBaseURL) {
			t.Errorf("New(%q) err = %v, want ErrInvalidBaseURL", baseURL, err)
		}
	}
}

func TestNewJoinsMetricsPath(t *testing.T) {
	t.Parallel()

	from, to := window()

	tests := []struct {
		suffix   string
		wantPath string
	}{
		{"/", "/metrics"},
		{"/api", "/api/metrics"},
		{"/api/", "/api/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			t.Parallel()

			var gotPath string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte("[]"))
			}))
			t.Cleanup(srv.Close)

			c, err := client.New(srv.URL+tt.suffix, 0)
			if err != nil {
				t.Fatalf("new client: %v", err)
			}

			if _, err := c.Get(t.Context(), 7, model.KindCPU, from, to); err != nil {
				t.Fatalf("get: %v", err)
			}

			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

// fakeReader records the arguments of its last call and returns a fixed result.
type fakeReader struct {
	metrics        []model.Metric
	agentID        uint32
	kind           model.Kind
	gotFrom, gotTo time.Time
}

func (f *fakeReader) Get(
	_ context.Context, agentID uint32, kind model.Kind, from, to time.Time,
) ([]model.Metric, error) {
	f.agentID, f.kind, f.gotFrom, f.gotTo = agentID, kind, from, to

	return f.metrics, nil
}

// serveReal starts the real HTTP handler backed by reader and returns a client pointed at it.
func serveReal(t *testing.T, reader server.MetricsReader) *client.Client {
	t.Helper()

	srv := httptest.NewServer(server.New(reader))
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, 0)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	return c
}

// TestGetAgainstServer checks that the client and the real handler agree on parameters and JSON fields.
func TestGetAgainstServer(t *testing.T) {
	t.Parallel()

	from, to := window()

	want := []model.Metric{{
		AgentID: 7,
		Kind:    model.KindMemory,
		Time:    time.Date(2026, 11, 4, 10, 0, 30, 500, time.UTC),
		Values:  map[string]float64{"used_pct": 42.5, "avail_gib": 9.25},
	}}
	reader := &fakeReader{metrics: want}

	got, err := serveReal(t, reader).Get(t.Context(), 7, model.KindMemory, from, to.Add(500))
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("metrics = %+v, want %+v", got, want)
	}

	if reader.agentID != 7 || reader.kind != model.KindMemory ||
		!reader.gotFrom.Equal(from) || !reader.gotTo.Equal(to.Add(500)) {
		t.Errorf("server got agent %d kind %q range [%v, %v]", reader.agentID, reader.kind, reader.gotFrom, reader.gotTo)
	}
}

func TestGetAgainstServerRejectsUnknownKind(t *testing.T) {
	t.Parallel()

	from, to := window()

	_, err := serveReal(t, &fakeReader{}).Get(t.Context(), 7, "disk", from, to)

	statusErr, ok := errors.AsType[*client.StatusError](err)
	if !ok || statusErr.StatusCode != http.StatusBadRequest || !strings.Contains(statusErr.Body, `unknown kind "disk"`) {
		t.Errorf("err = %v, want a 400 StatusError about the unknown kind", err)
	}
}
