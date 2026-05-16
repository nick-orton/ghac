package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// screenID identifies which screen is currently active.
type screenID int

const (
	screenVolume screenID = iota
	screenPlaylist
	screenNavigator
	screenHelp
)

// Model is the root Bubble Tea model. It owns all screen sub-models and
// handles global key bindings (screen switching, quit).
type Model struct {
	activeScreen screenID
	prevScreen   screenID
	width        int
	height       int

	volume    volumeScreen
	playlist  playlistScreen
	navigator navigatorScreen
	help      helpScreen
}

// New creates a new root model with the Player Volume screen active.
func New() Model {
	return Model{
		activeScreen: screenVolume,
		prevScreen:   screenVolume,
		volume:       newVolumeScreen(),
		playlist:     newPlaylistScreen(),
		navigator:    newNavigatorScreen(),
		help:         newHelpScreen(),
	}
}

// Init satisfies tea.Model. No startup commands are needed in Phase 1.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.activeScreen = screenVolume
			return m, nil
		case "2":
			m.activeScreen = screenPlaylist
			return m, nil
		case "3":
			m.activeScreen = screenNavigator
			return m, nil
		case "?":
			m.prevScreen = m.activeScreen
			m.activeScreen = screenHelp
			return m, nil
		case "esc":
			if m.activeScreen == screenHelp {
				m.activeScreen = m.prevScreen
				return m, nil
			}
		}
	}

	return m.delegateToActiveScreen(msg)
}

// delegateToActiveScreen forwards a message to the currently active screen.
func (m Model) delegateToActiveScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeScreen {
	case screenVolume:
		m.volume, cmd = m.volume.Update(msg)
	case screenPlaylist:
		m.playlist, cmd = m.playlist.Update(msg)
	case screenNavigator:
		m.navigator, cmd = m.navigator.Update(msg)
	case screenHelp:
		m.help, cmd = m.help.Update(msg)
	}
	return m, cmd
}

// View renders the current screen with the now-playing bar at the top.
func (m Model) View() string {
	np := NowPlayingView(m.width)

	var screen string
	switch m.activeScreen {
	case screenVolume:
		screen = m.volume.View()
	case screenPlaylist:
		screen = m.playlist.View()
	case screenNavigator:
		screen = m.navigator.View()
	case screenHelp:
		screen = m.help.View()
	}

	return np + "\n" + screen
}
