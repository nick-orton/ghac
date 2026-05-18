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

// screenID identifies a screen by its index in Model.screens.
type screenID int

const (
	screenVolume screenID = iota
	screenPlaylist
	screenNavigator
)

// screen is implemented by every sub-screen. It lets the root model dispatch
// messages, render tabs, and query screen state without a type switch.
// Adding a new screen only requires implementing this interface and appending
// the screen to the slice in New() — no other model.go changes needed.
type screen interface {
	update(tea.Msg) (screen, tea.Cmd)
	View() string
	hasPendingF() bool
	capturesAllInput() bool
	activeModal() (title, content string, minWidth, maxWidth int, ok bool)
	tabTitle() string
	screenTitle() string
}

// Model is the root Bubble Tea model. It owns all screen sub-models and
// handles global key bindings (screen switching, play/pause, quit).
type Model struct {
	activeScreen screenID
	showHelp     bool
	showTheme    bool
	width        int
	height       int

	// Theme state.
	activeThemeIdx   int         // index into Themes
	originalThemeIdx int         // theme at the time the modal opened (for revert)
	themeModal       themeScreen // theme selector modal

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

	// screens holds the ordered sub-screens; activeScreen indexes into it.
	screens []screen
	help    helpScreen
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
	ThemeIdx    int            // index into Themes; 0 = default
}

// New creates a new root model with the Player Volume screen active.
// Client pointers may be nil during tests; commands will be no-ops in that case.
func New(p NewParams) Model {
	idx := p.ThemeIdx
	if idx < 0 || idx >= len(Themes) {
		idx = 0
	}
	applyTheme(Themes[idx])
	return Model{
		activeScreen:   screenVolume,
		activeThemeIdx: idx,
		mpdClient:      p.MPD,
		snapClient:     p.Snapcast,
		playerStatus:   p.MPDState.Status,
		currentSong:    p.MPDState.Song,
		elapsed:        p.MPDState.Elapsed,
		totalDuration:  p.MPDState.TotalDuration,
		currentSongPos: p.MPDState.SongPos,
		randomOn:       p.MPDState.Random,
		screens: []screen{
			newVolumeScreen(p.Snapcast, p.SnapClients),
			newPlaylistScreen(p.MPD, p.Playlist, p.MPDState.SongPos),
			newNavigatorScreen(p.MPD, p.NavEntries).withPlaylist(p.Playlist),
		},
		help: newHelpScreen(),
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
		m = m.broadcastToScreens(msg)
		return m, nil

	case mpd.MsgPlayerState:
		m.playerStatus = msg.Status
		m.currentSong = msg.Song
		m.elapsed = msg.Elapsed
		m.totalDuration = msg.TotalDuration
		m.currentSongPos = msg.SongPos
		m.randomOn = msg.Random
		m = m.broadcastToScreens(msg)
		// Re-subscribe to the next idle event.
		if m.mpdClient != nil {
			return m, m.mpdClient.ListenIdle()
		}
		return m, nil

	case mpd.MsgPlaylistChanged:
		m = m.broadcastToScreens(msg)
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
		m = m.broadcastToScreens(msg)
		if m.snapClient != nil {
			return m, m.snapClient.ListenNotifications()
		}
		return m, nil

	case snapcast.MsgError:
		m.errMsg = msg.Err.Error()
		return m, tea.Quit

	case tea.KeyMsg:
		for _, handler := range keyHandlers {
			if newM, cmd, handled := handler(m, msg); handled {
				return newM, cmd
			}
		}
	}

	return m.delegateToActiveScreen(msg)
}

// activeScreenPendingF reports whether the active screen is mid f<letter>
// fast-navigation sequence and needs to consume the next key itself.
func (m Model) activeScreenPendingF() bool {
	return m.screens[m.activeScreen].hasPendingF()
}

// delegateToActiveScreen forwards a message to the currently active screen.
func (m Model) delegateToActiveScreen(msg tea.Msg) (Model, tea.Cmd) {
	s, cmd := m.screens[m.activeScreen].update(msg)
	m.screens[m.activeScreen] = s
	return m, cmd
}

// broadcastToScreens delivers msg to every screen's update method. Commands
// returned by screens are discarded — backend-subscription commands are owned
// by the root model. Use this for backend messages that screens need to
// observe in order to keep their own state current.
func (m Model) broadcastToScreens(msg tea.Msg) Model {
	for i, s := range m.screens {
		newS, _ := s.update(msg)
		m.screens[i] = newS
	}
	return m
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

	title := m.screens[m.activeScreen].screenTitle()
	content := m.screens[m.activeScreen].View()

	background := np + "\n" + m.tabStripView() + "\n" + screenBorder(title, content, m.width)

	// Pad the background to exactly m.height lines so the frame height is
	// always consistent regardless of whether a modal overlay is open. Without
	// this, closing a modal that extended beyond the background height leaves
	// orphaned lines (including the bottom border) from the previous frame.
	if m.height > 0 {
		bgLines := strings.Count(background, "\n") + 1
		if bgLines < m.height {
			background += strings.Repeat("\n", m.height-bgLines)
		}
	}

	if mTitle, mContent, minW, maxW, ok := m.screens[m.activeScreen].activeModal(); ok {
		modalWidth := m.width - 4
		if modalWidth > maxW {
			modalWidth = maxW
		}
		if modalWidth < minW {
			modalWidth = minW
		}
		modal := modalBorder(mTitle, mContent, modalWidth)
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

	if m.showTheme {
		// Modal width must fit the longest theme name (with cursor prefix)
		// and the hint line at the bottom.
		const hintLine = "  [enter] confirm  [esc] cancel"
		modalWidth := len(hintLine)
		for _, t := range Themes {
			if w := 2 + len(t.Name); w > modalWidth { // 2 for "▶ " prefix
				modalWidth = w
			}
		}
		modalWidth += 4 // border sides (2) + inner padding (2)
		if m.width-4 < modalWidth {
			modalWidth = m.width - 4
		}
		if modalWidth < 20 {
			modalWidth = 20
		}
		modal := modalBorder("Theme", m.themeModal.View(), modalWidth)
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
	parts := make([]string, len(m.screens)+1)
	for i, s := range m.screens {
		if screenID(i) == m.activeScreen {
			parts[i] = styleTabActive.Render(s.tabTitle())
		} else {
			parts[i] = styleTabInactive.Render(s.tabTitle())
		}
	}
	// ?:Help is always inactive — help is a modal overlay, not a peer screen.
	parts[len(m.screens)] = styleTabInactive.Render("?:Help")
	return strings.Join(parts, "  ")
}
