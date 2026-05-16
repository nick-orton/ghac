package ui

import tea "github.com/charmbracelet/bubbletea"

// playlistScreen is the Playlist Control screen (Phase 1: placeholder).
type playlistScreen struct{}

func newPlaylistScreen() playlistScreen {
	return playlistScreen{}
}

func (s playlistScreen) Update(msg tea.Msg) (playlistScreen, tea.Cmd) {
	return s, nil
}

func (s playlistScreen) View() string {
	return stylePlaceholder.Render("not yet connected")
}
