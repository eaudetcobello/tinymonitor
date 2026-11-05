// Package tui implements the tinymonitor-tui terminal app.
package tui

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/client"
	"github.com/eaudetcobello/tinymonitor/internal/model"
)

const (
	// DefaultServerURL is the address the tinymonitor daemon listens on.
	DefaultServerURL = "http://localhost:1323"
	// DefaultAgentID is the agent ID the collector currently writes.
	DefaultAgentID uint32 = 123123123
	// DefaultWindow is the rolling time span shown on the graph.
	DefaultWindow = 5 * time.Minute
	// DefaultRefresh is how often the API is polled.
	DefaultRefresh = time.Second
)

var (
	ErrInvalidFlag = errors.New("invalid flag")
	errAgentID     = errors.New("expected an integer between 0 and 4294967295")
)

// Config holds the validated command-line settings of tinymonitor-tui.
type Config struct {
	// ServerURL is the base URL of the tinymonitor HTTP API.
	ServerURL string
	AgentID   uint32
	Kind      model.Kind
	// Value is the series to graph, such as used_pct. Empty selects the kind's default series.
	Value string
	// Window is the rolling time span shown on the graph.
	Window time.Duration
	// Refresh is how often the API is polled.
	Refresh time.Duration
}

// ParseConfig parses and validates args (without the program name).
// Errors are returned, not printed. --help writes the usage to output and returns flag.ErrHelp.
func ParseConfig(args []string, output io.Writer) (Config, error) {
	cfg := Config{AgentID: DefaultAgentID}

	var kind string

	// Errors are returned rather than printed, so the caller reports them once.
	fs := flag.NewFlagSet("tinymonitor-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.StringVar(&cfg.ServerURL, "server", DefaultServerURL, "base URL of the tinymonitor HTTP API")
	fs.Func("agent", fmt.Sprintf("agent ID to graph (default %d)", DefaultAgentID), func(s string) error {
		id, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return errAgentID
		}

		cfg.AgentID = uint32(id)

		return nil
	})
	fs.StringVar(&kind, "kind", string(model.KindMemory), fmt.Sprintf("metric kind, one of %v", model.Kinds()))
	fs.StringVar(&cfg.Value, "value", "",
		"value series to graph, such as used_pct (default: used_pct)")
	fs.DurationVar(&cfg.Window, "window", DefaultWindow, "rolling time window shown on the graph")
	fs.DurationVar(&cfg.Refresh, "refresh", DefaultRefresh, "how often to poll the API, at most -window")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(fs, output)

			return Config{}, err //nolint:wrapcheck // Callers match flag.ErrHelp.
		}

		return Config{}, fmt.Errorf("%w: %w", ErrInvalidFlag, err)
	}

	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("%w: unexpected argument %q", ErrInvalidFlag, fs.Arg(0))
	}

	if err := cfg.validate(kind); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func printUsage(fs *flag.FlagSet, output io.Writer) {
	_, _ = fmt.Fprint(output,
		"tinymonitor-tui graphs tinymonitor metrics live in the terminal.\n\n",
		"Usage: tinymonitor-tui [flags]\n\nFlags:\n")

	fs.SetOutput(output)
	fs.PrintDefaults()
}

func (c *Config) validate(kind string) error {
	if _, err := client.New(c.ServerURL, 0); err != nil {
		return fmt.Errorf("%w -server: %w", ErrInvalidFlag, err)
	}

	k, err := model.ParseKind(kind)
	if err != nil {
		return fmt.Errorf("%w -kind: %w", ErrInvalidFlag, err)
	}

	c.Kind = k
	c.Value = strings.TrimSpace(c.Value)

	if c.Window <= 0 {
		return fmt.Errorf("%w -window %s: must be greater than 0", ErrInvalidFlag, c.Window)
	}

	if c.Refresh <= 0 {
		return fmt.Errorf("%w -refresh %s: must be greater than 0", ErrInvalidFlag, c.Refresh)
	}

	if c.Refresh > c.Window {
		return fmt.Errorf("%w -refresh %s: must not exceed -window %s", ErrInvalidFlag, c.Refresh, c.Window)
	}

	return nil
}
