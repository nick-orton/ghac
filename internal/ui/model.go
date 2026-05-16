package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
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
// handles global key bindings (screen switching, play/pause, quit).
type Model struct {
	activeScreen screenID
	prevScreen   screenID
	width        int
	height       int

	// mpdClient is a pointer so the TCP connections are never copied.
	mpdClient *mpd.Client

	// Player state populated from MsgPlayerState and advanced by MsgTick.
	playerStatus  string
	currentSong   mpd.Song
	elapsed       time.Duration
	totalDuration time.Duration

	// errMsg is set on fatal errors; View() shows it and Update() quits.
	errMsg string

	volume    volumeScreen
	playlist  playlistScreen
	navigator navigatorScreen
	help      helpScreen
}

// New creates a new root model with the Player Volume screen active.
// mpdClient may be nil during tests; play/pause will be no-ops in that case.
// initialState is the player state fetched before the TUI starts.
func New(mpdClient *mpd.Client, initialState mpd.MsgPlayerState) Model {
	return Model{
		activeScreen:  screenVolume,
		prevScreen:    screenVolume,
		mpdClient:     mpdClient,
		playerStatus:  initialState.Status,
		currentSong:   initialState.Song,
		elapsed:       initialState.Elapsed,
		totalDuration: initialState.TotalDuration,
		volume:        newVolumeScreen(),
		playlist:      newPlaylistScreen(),
		navigator:     newNavigatorScreen(),
		help:          newHelpScreen(),
	}
}

// Init starts the idle listener and the 1-second progress ticker.
func (m Model) Init() tea.Cmd {
	if m.mpdClient == nil {
		return mpd.TickCmd()
	}
	return tea.Batch(
		m.mpdClient.ListenIdle(),
		mpd.TickCmd(),
	)
}

// Update handles messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case mpd.MsgPlayerState:
		m.playerStatus = msg.Status
		m.currentSong = msg.Song
		m.elapsed = msg.Elapsed
		m.totalDuration = msg.TotalDuration
		// Re-subscribe to the next idle event.
		if m.mpdClient != nil {
			return m, m.mpdClient.ListenIdle()
		}
		return m, nil

	case mpd.MsgTick:
		if m.playerStatus == "play" {
			m.elapsed += time.Second
		}
		return m, mpd.TickCmd()

	case mpd.MsgError:
		m.errMsg = msg.Err.Error()
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "p":
			if m.mpdClient != nil {
				if m.playerStatus == "play" {
					_ = m.mpdClient.Pause()
				} else {
					_ = m.mpdClient.Play()
				}
			}
			return m, nil
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
	if m.errMsg != "" {
		return "Error: " + m.errMsg + "\n"
	}

	ps := PlayerState{
		Status:        m.playerStatus,
		Title:         m.currentSong.Title,
		Artist:        m.currentSong.Artist,
		Album:         m.currentSong.Album,
		File:          m.currentSong.File,
		Elapsed:       m.elapsed,
		TotalDuration: m.totalDuration,
	}
	np := NowPlayingView(ps, m.width)

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

	return np + "\n" + m.tabStripView() + "\n" + screen
}

// tabStripView renders the tab bar showing all screens with the active one
// highlighted via bold+underline and inactive ones dimmed.
func (m Model) tabStripView() string {
	type tab struct {
		id    screenID
		label string
	}
	tabs := []tab{
		{screenVolume, "1:Volume"},
		{screenPlaylist, "2:Playlist"},
		{screenNavigator, "3:Navigator"},
		{screenHelp, "?:Help"},
	}
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		if t.id == m.activeScreen {
			parts[i] = styleTabActive.Render(t.label)
		} else {
			parts[i] = styleTabInactive.Render(t.label)
		}
	}
	return strings.Join(parts, "  ")
}
