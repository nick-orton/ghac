package ui

import tea "github.com/charmbracelet/bubbletea"

// keyHandler processes a single key event. It returns the (possibly modified)
// model, an optional command, and whether the key was consumed. When handled
// is true the chain stops; when false the next handler is tried.
type keyHandler func(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool)

// keyHandlers is the ordered chain of key handlers. Each handler is tried in
// sequence; the first to return handled=true wins. To add a new global key
// binding, append a new function — no existing handler code needs to change.
var keyHandlers = []keyHandler{
	handleQuit,
	handleRenameModal,
	handleEsc,
	handleHelpToggle,
	handleThemeToggle,
	handleThemeModal,
	handleHelpModal,
	handlePendingF,
	handleMediaKeys,
	handleScreenSwitch,
}

// handleQuit handles q and ctrl+c, which always quit regardless of modal state.
func handleQuit(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, true
	}
	return m, nil, false
}

// handleRenameModal delegates all keys to the active screen while the rename
// modal is open. The volume screen owns rename input handling.
func handleRenameModal(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.volume.showRename {
		return m, nil, false
	}
	newM, cmd := m.delegateToActiveScreen(msg)
	return newM, cmd, true
}

// handleEsc closes the theme or help modal when Escape is pressed.
func handleEsc(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if msg.String() != "esc" {
		return m, nil, false
	}
	if m.showTheme {
		applyTheme(Themes[m.originalThemeIdx])
		m.activeThemeIdx = m.originalThemeIdx
		m.showTheme = false
		return m, nil, true
	}
	if m.showHelp {
		m.showHelp = false
		return m, nil, true
	}
	return m, nil, false
}

// handleHelpToggle toggles the help modal on ?.
func handleHelpToggle(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if msg.String() != "?" || m.showTheme {
		return m, nil, false
	}
	m.showHelp = !m.showHelp
	return m, nil, true
}

// handleThemeToggle opens or closes the theme modal on ctrl+t.
func handleThemeToggle(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if msg.String() != "ctrl+t" || m.showHelp {
		return m, nil, false
	}
	if m.showTheme {
		// Second press: revert and close (same as Esc).
		applyTheme(Themes[m.originalThemeIdx])
		m.activeThemeIdx = m.originalThemeIdx
		m.showTheme = false
	} else {
		m.originalThemeIdx = m.activeThemeIdx
		m.themeModal = newThemeScreen(m.activeThemeIdx)
		m.showTheme = true
	}
	return m, nil, true
}

// handleThemeModal routes keys into the theme modal while it is open.
// Enter confirms the selection; all other keys are forwarded to the modal.
func handleThemeModal(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showTheme {
		return m, nil, false
	}
	if msg.String() == "enter" {
		m.activeThemeIdx = m.themeModal.cursor
		_ = SaveThemeState(Themes[m.activeThemeIdx].Name)
		m.showTheme = false
		return m, nil, true
	}
	m.themeModal, _ = m.themeModal.Update(msg)
	m.activeThemeIdx = m.themeModal.cursor
	return m, nil, true
}

// handleHelpModal swallows all keys while the help modal is open.
func handleHelpModal(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showHelp {
		return m, nil, false
	}
	return m, nil, true
}

// handlePendingF forwards keys to the active screen while it is waiting for
// an f<letter> fast-navigation sequence, preventing global handlers from
// stealing keys like "p".
func handlePendingF(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.activeScreenPendingF() {
		return m, nil, false
	}
	newM, cmd := m.delegateToActiveScreen(msg)
	return newM, cmd, true
}

// handleMediaKeys handles global playback controls: p (play/pause) and z (random).
func handleMediaKeys(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.String() {
	case "p":
		if m.mpdClient != nil {
			if m.playerStatus == "play" {
				_ = m.mpdClient.Pause()
			} else {
				_ = m.mpdClient.Play()
			}
		}
		return m, nil, true
	case "z":
		if m.mpdClient != nil {
			_ = m.mpdClient.Random(!m.randomOn)
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleScreenSwitch routes 1/2/3 to the corresponding screen.
func handleScreenSwitch(m Model, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.String() {
	case "1":
		m.activeScreen = screenVolume
		return m, nil, true
	case "2":
		m.activeScreen = screenPlaylist
		return m, nil, true
	case "3":
		m.activeScreen = screenNavigator
		return m, nil, true
	}
	return m, nil, false
}
