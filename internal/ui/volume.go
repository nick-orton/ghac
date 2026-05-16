package ui

import tea "github.com/charmbracelet/bubbletea"

// volumeScreen is the Player Volume screen (Phase 1: placeholder).
type volumeScreen struct{}

func newVolumeScreen() volumeScreen {
	return volumeScreen{}
}

func (s volumeScreen) Update(msg tea.Msg) (volumeScreen, tea.Cmd) {
	return s, nil
}

func (s volumeScreen) View() string {
	title := styleTitle.Render("Player Volume")
	msg := stylePlaceholder.Render("not yet connected")
	return title + "\n\n" + msg
}
