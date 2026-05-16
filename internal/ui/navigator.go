package ui

import tea "github.com/charmbracelet/bubbletea"

// navigatorScreen is the Song Navigator screen (Phase 1: placeholder).
type navigatorScreen struct{}

func newNavigatorScreen() navigatorScreen {
	return navigatorScreen{}
}

func (s navigatorScreen) Update(msg tea.Msg) (navigatorScreen, tea.Cmd) {
	return s, nil
}

func (s navigatorScreen) View() string {
	return stylePlaceholder.Render("not yet connected")
}
