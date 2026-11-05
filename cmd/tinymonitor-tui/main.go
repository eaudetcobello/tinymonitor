// Command tinymonitor-tui graphs tinymonitor metrics live in the terminal.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/eaudetcobello/tinymonitor/internal/tui"
)

func main() {
	os.Exit(run())
}

// run returns the process exit code: 0 on a clean exit or --help, 1 when the app fails, 2 on invalid flags.
func run() int {
	cfg, err := tui.ParseConfig(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "tinymonitor-tui: %v\nRun tinymonitor-tui -help for usage.\n", err)

		return 2
	}

	// Bubble Tea quits cleanly on SIGTERM and returns ErrInterrupted on a SIGINT that bypasses raw mode
	// (input that is not a TTY). Both are a requested stop, not a failure.
	_, err = tea.NewProgram(tui.New(cfg)).Run()
	if err != nil && !errors.Is(err, tea.ErrInterrupted) {
		_, _ = fmt.Fprintf(os.Stderr, "tinymonitor-tui: %v\n", err)

		return 1
	}

	return 0
}
