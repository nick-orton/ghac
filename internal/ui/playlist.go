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
	title := styleTitle.Render("Playlist Control")
	msg := stylePlaceholder.Render("not yet connected")
	return title + "\n\n" + msg
}
