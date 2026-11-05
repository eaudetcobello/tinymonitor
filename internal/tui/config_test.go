package tui_test

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/tui"
)

const (
	flagServer  = "server"
	flagAgent   = "agent"
	flagKind    = "kind"
	flagWindow  = "window"
	flagRefresh = "refresh"
	flagValue   = "value"
)

// flags turns name/value pairs into "--name value" arguments.
func flags(pairs ...string) []string {
	args := make([]string, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		args = append(args, "--"+pairs[i], pairs[i+1])
	}

	return args
}

func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()

	got, err := tui.ParseConfig(nil, io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	want := tui.Config{
		ServerURL: tui.DefaultServerURL,
		AgentID:   tui.DefaultAgentID,
		Kind:      model.KindMemory,
		Window:    tui.DefaultWindow,
		Refresh:   tui.DefaultRefresh,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Parallel()

	got, err := tui.ParseConfig(flags(
		flagServer, "https://example.com:8080/prefix",
		flagAgent, "7",
		flagKind, "cpu",
		flagWindow, "1h",
		flagRefresh, "5s",
		flagValue, " "+availGiB+" ",
	), io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	want := tui.Config{
		ServerURL: "https://example.com:8080/prefix",
		AgentID:   7,
		Kind:      model.KindCPU,
		Window:    time.Hour,
		Refresh:   5 * time.Second,
		Value:     availGiB,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseConfigRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		mention string
	}{
		{"server without scheme", flags(flagServer, "localhost:1323"), flagServer},
		{"server with other scheme", flags(flagServer, "ftp://x"), flagServer},
		{"empty server", flags(flagServer, ""), flagServer},
		{"negative agent", flags(flagAgent, "-1"), flagAgent},
		{"agent over uint32", flags(flagAgent, "4294967296"), flagAgent},
		{"non-numeric agent", flags(flagAgent, "abc"), flagAgent},
		{"unknown kind", flags(flagKind, "disk"), flagKind},
		{"zero window", flags(flagWindow, "0"), flagWindow},
		{"malformed window", flags(flagWindow, "5 minutes"), flagWindow},
		{"negative refresh", flags(flagRefresh, "-1s"), flagRefresh},
		{"refresh longer than window", flags(flagWindow, "1m", flagRefresh, "2m"), flagRefresh},
		{"unknown flag", []string{"--nope"}, "nope"},
		{"positional argument", []string{"extra"}, "extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder

			_, err := tui.ParseConfig(tt.args, &out)
			if !errors.Is(err, tui.ErrInvalidFlag) {
				t.Fatalf("got error %v, want ErrInvalidFlag", err)
			}

			if out.Len() > 0 {
				t.Errorf("wrote %q to output, want errors returned only", out.String())
			}

			if !strings.Contains(err.Error(), tt.mention) {
				t.Errorf("error %q does not mention %q", err, tt.mention)
			}
		})
	}
}

func TestParseConfigHelpDocumentsEveryFlag(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	_, err := tui.ParseConfig([]string{"--help"}, &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("got error %v, want flag.ErrHelp", err)
	}

	for _, name := range []string{flagServer, flagAgent, flagKind, flagWindow, flagRefresh, flagValue} {
		if !strings.Contains(out.String(), "-"+name+" ") {
			t.Errorf("usage does not document %s:\n%s", name, out.String())
		}
	}
}
