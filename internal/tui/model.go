package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model is the Bubble Tea model of tinymonitor-tui: a header with the settings,
// a body that fills the terminal and a footer with the keys.
type Model struct {
	cfg           Config
	width, height int
}

// New returns a Model that displays cfg.
func New(cfg Config) Model {
	return Model{cfg: cfg}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model: it tracks the terminal size and quits on q or Ctrl+C.
//
//nolint:ireturn // tea.Model requires returning the interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model. It renders full screen, exactly the size of the terminal.
func (m Model) View() tea.View {
	v := tea.NewView("loading…")
	v.AltScreen = true

	if m.width <= 0 || m.height <= 0 {
		return v
	}

	line := lipgloss.NewStyle().Inline(true).MaxWidth(m.width)
	header := line.Render(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf(
		"tinymonitor  %s  agent %d  %s  window %s  refresh %s",
		m.cfg.ServerURL, m.cfg.AgentID, m.cfg.Kind, m.cfg.Window, m.cfg.Refresh,
	)))
	footer := line.Render(lipgloss.NewStyle().Faint(true).Render("q quit"))

	// The body is what TASK-1.5 replaces with the graph.
	bodyHeight := max(m.height-2, 0)
	body := lipgloss.Place(m.width, bodyHeight, lipgloss.Center, lipgloss.Center, line.Render("no graph yet"))

	v.SetContent(lipgloss.NewStyle().
		Width(m.width).MaxWidth(m.width).
		Height(m.height).MaxHeight(m.height).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, body, footer)))

	return v
}
