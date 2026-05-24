package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/snapcast"
)

const volumeBarWidth = 20

// volumeScreen is the Player Volume screen. It displays one row per SnapCast
// client with real-time volume and mute controls.
type volumeScreen struct {
	clients      []snapcast.SnapClient
	cursor       int
	sc           *snapcast.Client // may be nil in tests; commands become no-ops
	showRename   bool
	renameInput  []rune
	renameCursor int
}

func newVolumeScreen(sc *snapcast.Client, clients []snapcast.SnapClient) volumeScreen {
	return volumeScreen{
		clients: clients,
		cursor:  0,
		sc:      sc,
	}
}

// withClients returns a copy of the screen with the client list replaced and
// the cursor clamped to the new list length.
func (s volumeScreen) withClients(clients []snapcast.SnapClient) volumeScreen {
	s.clients = clients
	if len(clients) == 0 {
		s.cursor = 0
	} else if s.cursor >= len(clients) {
		s.cursor = len(clients) - 1
	}
	return s
}

func (s volumeScreen) Update(msg tea.Msg) (volumeScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case snapcast.MsgClientsUpdated:
		return s.withClients(msg.Clients), nil
	case tea.KeyMsg:
		if s.showRename {
			return s.updateRename(msg)
		}
		switch msg.String() {
		case "j":
			if s.cursor < len(s.clients)-1 {
				s.cursor++
			}
		case "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "l":
			s = s.adjustVolume(s.cursor, +5)
		case "h":
			s = s.adjustVolume(s.cursor, -5)
		case "m":
			s = s.toggleMute(s.cursor)
		case "L":
			for i := range s.clients {
				s = s.adjustVolume(i, +5)
			}
		case "H":
			for i := range s.clients {
				s = s.adjustVolume(i, -5)
			}
		case "M":
			for i := range s.clients {
				s = s.toggleMute(i)
			}
		case "ctrl+r":
			if len(s.clients) > 0 {
				s.showRename = true
				s.renameInput = []rune(s.clients[s.cursor].Name)
				s.renameCursor = len(s.renameInput)
			}
		}
	}
	return s, nil
}

// updateRename handles key events while the rename modal is open.
func (s volumeScreen) updateRename(msg tea.KeyMsg) (volumeScreen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.showRename = false
		s.renameInput = nil
		s.renameCursor = 0
	case "ctrl+s":
		if len(s.renameInput) > 0 {
			newName := string(s.renameInput)
			if s.sc != nil {
				_ = s.sc.SetName(s.clients[s.cursor].ID, newName)
			}
			// Update local state immediately so the display reflects the
			// change without waiting for the server notification.
			clients := make([]snapcast.SnapClient, len(s.clients))
			copy(clients, s.clients)
			clients[s.cursor].Name = newName
			s.clients = clients
			s.showRename = false
			s.renameInput = nil
			s.renameCursor = 0
		}
	case "left":
		if s.renameCursor > 0 {
			s.renameCursor--
		}
	case "right":
		if s.renameCursor < len(s.renameInput) {
			s.renameCursor++
		}
	case "home", "ctrl+a":
		s.renameCursor = 0
	case "end", "ctrl+e":
		s.renameCursor = len(s.renameInput)
	case "backspace":
		if s.renameCursor > 0 {
			s.renameInput = append(s.renameInput[:s.renameCursor-1], s.renameInput[s.renameCursor:]...)
			s.renameCursor--
		}
	case "delete":
		if s.renameCursor < len(s.renameInput) {
			s.renameInput = append(s.renameInput[:s.renameCursor], s.renameInput[s.renameCursor+1:]...)
		}
	default:
		var runes []rune
		switch msg.Type {
		case tea.KeyRunes:
			runes = msg.Runes
		case tea.KeySpace:
			runes = []rune{' '}
		}
		if len(runes) > 0 {
			s.renameInput = append(s.renameInput[:s.renameCursor], append(runes, s.renameInput[s.renameCursor:]...)...)
			s.renameCursor += len(runes)
		}
	}
	return s, nil
}

// renameModalContent returns the inner content of the rename modal for
// compositing by Model.View().
func (s volumeScreen) renameModalContent() string {
	var b strings.Builder
	left := string(s.renameInput[:s.renameCursor])
	right := string(s.renameInput[s.renameCursor:])
	b.WriteString("\n")
	b.WriteString("  Name: " + left + "_" + right + "\n")
	b.WriteString("\n")
	b.WriteString("  " + styleHelpDesc.Render("Ctrl-S: save   Esc: cancel") + "\n")
	return b.String()
}

// adjustVolume changes the volume of client at index i by delta, clamping to
// 0–100, and sends the change to SnapCast. Returns an updated screen.
func (s volumeScreen) adjustVolume(i, delta int) volumeScreen {
	if i < 0 || i >= len(s.clients) {
		return s
	}
	// Copy the slice so mutations in the returned screen do not alias the
	// caller's copy (important for test correctness).
	clients := make([]snapcast.SnapClient, len(s.clients))
	copy(clients, s.clients)
	s.clients = clients

	newVol := s.clients[i].Volume + delta
	if newVol < 0 {
		newVol = 0
	} else if newVol > 100 {
		newVol = 100
	}
	if s.sc != nil {
		_ = s.sc.SetVolume(s.clients[i].ID, newVol, s.clients[i].Muted)
	}
	s.clients[i].Volume = newVol
	return s
}

// toggleMute flips the muted flag of client at index i and sends the change
// to SnapCast. Returns an updated screen.
func (s volumeScreen) toggleMute(i int) volumeScreen {
	if i < 0 || i >= len(s.clients) {
		return s
	}
	clients := make([]snapcast.SnapClient, len(s.clients))
	copy(clients, s.clients)
	s.clients = clients

	newMuted := !s.clients[i].Muted
	if s.sc != nil {
		_ = s.sc.SetMute(s.clients[i].ID, newMuted, s.clients[i].Volume)
	}
	s.clients[i].Muted = newMuted
	return s
}

func (s volumeScreen) View() string {
	var b strings.Builder

	if len(s.clients) == 0 {
		b.WriteString(stylePlaceholder.Render("No clients connected"))
		return b.String()
	}

	for i, client := range s.clients {
		b.WriteString(s.renderRow(i, client))
		b.WriteString("\n")
	}
	return b.String()
}

// renderRow produces one line of output for a client entry.
//
// Layout:
//
//	▶ Name                 ████████████████░░░░  74%  [M]
//	  Name                 ████████████████░░░░  74%
func (s volumeScreen) renderRow(i int, client snapcast.SnapClient) string {
	cursor := "  "
	if i == s.cursor {
		cursor = symCursor
	}

	name := fmt.Sprintf("%-20s", truncateName(client.Name, 20))
	bar := renderVolumeBar(client.Volume, client.Muted)
	pct := fmt.Sprintf("%3d%%", client.Volume)

	mute := "   "
	if client.Muted {
		mute = "[M]"
	}

	row := cursor + name + "  " + bar + "  " + pct + "  " + mute
	if i == s.cursor {
		return styleRowActive.Render(row)
	}
	return row
}

// renderVolumeBar renders a fixed-width bar. The filled portion is green when
// unmuted and red when muted; the unfilled portion is always dim gray.
func renderVolumeBar(pct int, muted bool) string {
	fillStyle := styleVolumeBarFillUnmuted
	if muted {
		fillStyle = styleVolumeBarFillMuted
	}
	filled := (pct * volumeBarWidth) / 100
	var b strings.Builder
	for i := 0; i < volumeBarWidth; i++ {
		if i < filled {
			b.WriteString(fillStyle.Render(symFilled))
		} else {
			b.WriteString(styleProgressEmpty.Render(symEmpty))
		}
	}
	return b.String()
}

// truncateName shortens s to at most maxLen runes, appending "…" if truncated.
func truncateName(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + symEllipsis
}

// screen interface implementation.

func (s volumeScreen) update(msg tea.Msg) (screen, tea.Cmd) { return s.Update(msg) }
func (s volumeScreen) hasPendingF() bool                    { return false }
func (s volumeScreen) capturesAllInput() bool               { return s.showRename }
func (s volumeScreen) tabTitle() string                     { return "1:Volume" }
func (s volumeScreen) screenTitle() string                  { return "Player Volume" }

func (s volumeScreen) activeModal() (title, content string, minWidth, maxWidth int, ok bool) {
	if !s.showRename {
		return
	}
	return "Rename Client", s.renameModalContent(), 30, 50, true
}
