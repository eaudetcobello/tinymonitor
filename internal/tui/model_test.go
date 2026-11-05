package tui_test

import (
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/eaudetcobello/tinymonitor/internal/tui"
)

func newModel(t *testing.T) tui.Model {
	t.Helper()

	cfg, err := tui.ParseConfig(nil, io.Discard)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return tui.New(cfg)
}

// update sends msg to m and returns the updated tui.Model.
func update(t *testing.T, m tui.Model, msg tea.Msg) (tui.Model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)

	got, ok := next.(tui.Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}

	return got, cmd
}

func TestViewFillsTerminalOnResize(t *testing.T) {
	t.Parallel()

	sizes := []struct{ width, height int }{{120, 40}, {80, 24}, {20, 5}, {5, 1}}

	m := newModel(t)

	for _, size := range sizes {
		m, _ = update(t, m, tea.WindowSizeMsg{Width: size.width, Height: size.height})

		v := m.View()
		if !v.AltScreen {
			t.Errorf("%dx%d: AltScreen is off", size.width, size.height)
		}

		if got := lipgloss.Height(v.Content); got != size.height {
			t.Errorf("%dx%d: view is %d lines high", size.width, size.height, got)
		}

		for i, line := range strings.Split(v.Content, "\n") {
			if got := lipgloss.Width(line); got != size.width {
				t.Errorf("%dx%d: line %d is %d cells wide", size.width, size.height, i, got)
			}
		}
	}
}

func TestViewBeforeFirstResizeUsesAltScreen(t *testing.T) {
	t.Parallel()

	if v := newModel(t).View(); !v.AltScreen {
		t.Error("AltScreen is off before the first WindowSizeMsg")
	}
}

func TestQuitKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      tea.KeyPressMsg
		wantQuit bool
	}{
		{"q", tea.KeyPressMsg{Code: 'q', Text: "q"}, true},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, true},
		{"other key", tea.KeyPressMsg{Code: 'x', Text: "x"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, cmd := update(t, newModel(t), tt.key)

			quit := false
			if cmd != nil {
				_, quit = cmd().(tea.QuitMsg)
			}

			if quit != tt.wantQuit {
				t.Errorf("quit = %v, want %v", quit, tt.wantQuit)
			}
		})
	}
}
