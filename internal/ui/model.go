package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"ghac/internal/mpd"
	"ghac/internal/snapcast"
)

// screenID identifies which screen is currently active.
type screenID int

const (
	screenVolume screenID = iota
	screenPlaylist
	screenNavigator
)

// Model is the root Bubble Tea model. It owns all screen sub-models and
// handles global key bindings (screen switching, play/pause, quit).
type Model struct {
	activeScreen screenID
	showHelp     bool
	width        int
	height       int

	// mpdClient is a pointer so the TCP connections are never copied.
	mpdClient *mpd.Client

	// snapClient is a pointer so the TCP connection is never copied.
	snapClient *snapcast.Client

	// Player state populated from MsgPlayerState and advanced by MsgTick.
	playerStatus   string
	currentSong    mpd.Song
	elapsed        time.Duration
	totalDuration  time.Duration
	currentSongPos int  // 0-indexed playlist position; -1 if none playing
	randomOn       bool // true when MPD random (shuffle) mode is active

	// errMsg is set on fatal errors; View() shows it and Update() quits.
	errMsg string

	volume    volumeScreen
	playlist  playlistScreen
	navigator navigatorScreen
	help      helpScreen
}

// NewParams holds all dependencies and initial state for New(). Using a struct
// avoids a long positional parameter list and makes call sites self-documenting.
type NewParams struct {
	MPD         *mpd.Client
	MPDState    mpd.MsgPlayerState
	Snapcast    *snapcast.Client
	SnapClients []snapcast.SnapClient
	Playlist    []mpd.PlaylistEntry
	NavEntries  []mpd.DirEntry // initial listing of the music library root
}

// New creates a new root model with the Player Volume screen active.
// Client pointers may be nil during tests; commands will be no-ops in that case.
func New(p NewParams) Model {
	return Model{
		activeScreen:   screenVolume,
		mpdClient:      p.MPD,
		snapClient:     p.Snapcast,
		playerStatus:   p.MPDState.Status,
		currentSong:    p.MPDState.Song,
		elapsed:        p.MPDState.Elapsed,
		totalDuration:  p.MPDState.TotalDuration,
		currentSongPos: p.MPDState.SongPos,
		randomOn:       p.MPDState.Random,
		volume:         newVolumeScreen(p.Snapcast, p.SnapClients),
		playlist:       newPlaylistScreen(p.MPD, p.Playlist, p.MPDState.SongPos),
		navigator:      newNavigatorScreen(p.MPD, p.NavEntries).withPlaylist(p.Playlist),
		help:           newHelpScreen(),
	}
}

// Init starts the idle listener, the SnapCast notification listener, and the
// 1-second progress ticker.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{mpd.TickCmd()}
	if m.mpdClient != nil {
		cmds = append(cmds, m.mpdClient.ListenIdle())
	}
	if m.snapClient != nil {
		cmds = append(cmds, m.snapClient.ListenNotifications())
	}
	return tea.Batch(cmds...)
}

// Update handles messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.navigator = m.navigator.withWidth(msg.Width).withHeight(msg.Height)
		m.playlist = m.playlist.withHeight(msg.Height)
		return m, nil

	case mpd.MsgPlayerState:
		m.playerStatus = msg.Status
		m.currentSong = msg.Song
		m.elapsed = msg.Elapsed
		m.totalDuration = msg.TotalDuration
		m.currentSongPos = msg.SongPos
		m.randomOn = msg.Random
		m.playlist = m.playlist.withCurrentPos(msg.SongPos)
		// Re-subscribe to the next idle event.
		if m.mpdClient != nil {
			return m, m.mpdClient.ListenIdle()
		}
		return m, nil

	case mpd.MsgPlaylistChanged:
		m.playlist = m.playlist.withEntries(msg.Entries, m.currentSongPos)
		m.navigator = m.navigator.withPlaylist(msg.Entries)
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

	case snapcast.MsgClientsUpdated:
		m.volume = m.volume.withClients(msg.Clients)
		if m.snapClient != nil {
			return m, m.snapClient.ListenNotifications()
		}
		return m, nil

	case snapcast.MsgError:
		m.errMsg = msg.Err.Error()
		return m, tea.Quit

	case tea.KeyMsg:
		// Global quit keys are always handled.
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		// While the rename modal is open, delegate all keys to the volume
		// screen (which handles esc, ctrl+s, and text editing).
		if m.volume.showRename {
			return m.delegateToActiveScreen(msg)
		}

		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
		}

		// While the help modal is open, swallow all other key events.
		if m.showHelp {
			return m, nil
		}

		// If the active screen is waiting for a second key (f<letter>
		// fast-navigation), forward ALL keys to it before global handlers
		// can steal them (e.g. "p" for play/pause).
		if m.activeScreenPendingF() {
			return m.delegateToActiveScreen(msg)
		}

		switch msg.String() {
		case "p":
			if m.mpdClient != nil {
				if m.playerStatus == "play" {
					_ = m.mpdClient.Pause()
				} else {
					_ = m.mpdClient.Play()
				}
			}
			return m, nil
		case "z":
			if m.mpdClient != nil {
				_ = m.mpdClient.Random(!m.randomOn)
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
		}
	}

	return m.delegateToActiveScreen(msg)
}

// activeScreenPendingF reports whether the active screen is mid f<letter>
// fast-navigation sequence and needs to consume the next key itself.
func (m Model) activeScreenPendingF() bool {
	switch m.activeScreen {
	case screenPlaylist:
		return m.playlist.pendingF
	case screenNavigator:
		return m.navigator.pendingF
	}
	return false
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
	}
	return m, cmd
}

// View renders the current screen with the now-playing bar at the top.
// When showHelp is true, a modal overlay is composited over the screen.
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
		Random:        m.randomOn,
	}
	np := NowPlayingView(ps, m.width)

	var title, content string
	switch m.activeScreen {
	case screenVolume:
		title, content = "Player Volume", m.volume.View()
	case screenPlaylist:
		title, content = "Playlist Control", m.playlist.View()
	case screenNavigator:
		title, content = "Library Navigator", m.navigator.View()
	}

	background := np + "\n" + m.tabStripView() + "\n" + screenBorder(title, content, m.width)

	if m.volume.showRename {
		// Render the rename modal and overlay it centered on the background.
		modalWidth := m.width - 4
		if modalWidth > 50 {
			modalWidth = 50
		}
		if modalWidth < 30 {
			modalWidth = 30
		}
		modal := modalBorder("Rename Client", m.volume.renameModalContent(), modalWidth)
		modalLines := strings.Count(modal, "\n") + 1
		x := (m.width - modalWidth) / 2
		y := (m.height - modalLines) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		return placeOverlay(x, y, modal, background)
	}

	if !m.showHelp {
		return background
	}

	// Render the help modal and overlay it centered on the background.
	modalWidth := m.width - 4
	if modalWidth > 82 {
		modalWidth = 82
	}
	if modalWidth < 20 {
		modalWidth = 20
	}
	modal := modalBorder("Help", m.help.View(), modalWidth)

	modalLines := strings.Count(modal, "\n") + 1
	x := (m.width - modalWidth) / 2
	y := (m.height - modalLines) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return placeOverlay(x, y, modal, background)
}

// placeOverlay composites fg over bg, positioning the top-left corner of fg
// at column x, row y (0-indexed). Both strings may contain ANSI escape codes.
// Background content is visible to the left and right of the overlay.
// If bg has fewer lines than y+len(fgLines), the remaining fg lines are
// emitted on blank lines so the full modal is always visible.
func placeOverlay(x, y int, fg, bg string) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	// Total output rows: at least as many as bg, but enough to show all of fg.
	totalRows := len(bgLines)
	if needed := y + len(fgLines); needed > totalRows {
		totalRows = needed
	}

	var b strings.Builder
	for i := 0; i < totalRows; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}

		bgLine := ""
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}

		fgIdx := i - y
		if fgIdx < 0 || fgIdx >= len(fgLines) {
			b.WriteString(bgLine)
			continue
		}

		fgLine := fgLines[fgIdx]
		fgWidth := ansi.StringWidth(fgLine)

		// Left portion of background up to column x.
		left := ansi.Truncate(bgLine, x, "")
		leftWidth := ansi.StringWidth(left)
		b.WriteString(left)
		if leftWidth < x {
			b.WriteString(strings.Repeat(" ", x-leftWidth))
		}

		// Modal content, followed by a reset so modal styles don't bleed right.
		b.WriteString(fgLine)
		b.WriteString("\x1b[0m")

		// Right portion of background after the modal.
		b.WriteString(ansi.TruncateLeft(bgLine, x+fgWidth, ""))
	}
	return b.String()
}

// screenBorder wraps content in a single-line box with the screen title
// embedded in the top edge:
//
//	┌─ Title ──────────────────────────────────────┐
//	│ content line                                 │
//	└──────────────────────────────────────────────┘
//
// width is the full terminal width; a minimum of 80 is enforced.
func screenBorder(title, content string, width int) string {
	if width < 4 {
		width = 80
	}

	// Top edge: ┌─ Title ─────...─┐
	styledTitle := styleTitle.Render(title)
	titleSeg := "─ " + styledTitle + " "
	fillLen := width - 2 - lipgloss.Width(titleSeg)
	if fillLen < 1 {
		fillLen = 1
	}
	top := "┌" + titleSeg + strings.Repeat("─", fillLen) + "┐"

	// Bottom edge: └──────...──┘
	bottom := "└" + strings.Repeat("─", width-2) + "┘"

	// Inner content area: width minus two border chars and one space pad each side.
	innerWidth := width - 4

	lines := strings.Split(content, "\n")
	// Drop trailing blank line that screens often emit.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	b.WriteString(top)
	b.WriteByte('\n')
	for _, line := range lines {
		pad := innerWidth - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		b.WriteString("│ ")
		b.WriteString(line)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(" │\n")
	}
	b.WriteString(bottom)
	return b.String()
}

// modalBorder wraps content in a box at the given width, with the title
// embedded in the top edge. Uses the same box-drawing characters as
// screenBorder but operates at modal (not terminal) width.
func modalBorder(title, content string, width int) string {
	if width < 4 {
		width = 20
	}

	styledTitle := styleTitle.Render(title)
	titleSeg := "─ " + styledTitle + " "
	fillLen := width - 2 - lipgloss.Width(titleSeg)
	if fillLen < 1 {
		fillLen = 1
	}
	top := "┌" + titleSeg + strings.Repeat("─", fillLen) + "┐"
	bottom := "└" + strings.Repeat("─", width-2) + "┘"

	innerWidth := width - 4

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	b.WriteString(top)
	b.WriteByte('\n')
	for _, line := range lines {
		pad := innerWidth - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		b.WriteString("│ ")
		b.WriteString(line)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(" │\n")
	}
	b.WriteString(bottom)
	return b.String()
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
		{screenNavigator, "3:Library"},
	}
	parts := make([]string, len(tabs)+1)
	for i, t := range tabs {
		if t.id == m.activeScreen {
			parts[i] = styleTabActive.Render(t.label)
		} else {
			parts[i] = styleTabInactive.Render(t.label)
		}
	}
	// ?:Help is always inactive — help is a modal overlay, not a peer screen.
	parts[len(tabs)] = styleTabInactive.Render("?:Help")
	return strings.Join(parts, "  ")
}
