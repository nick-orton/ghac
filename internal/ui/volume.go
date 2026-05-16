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
	clients []snapcast.SnapClient
	cursor  int
	sc      *snapcast.Client // may be nil in tests; commands become no-ops
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
	case tea.KeyMsg:
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
		}
	}
	return s, nil
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
	title := styleTitle.Render("Player Volume")
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

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
		cursor = "▶ "
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
			b.WriteString(fillStyle.Render("█"))
		} else {
			b.WriteString(styleProgressEmpty.Render("░"))
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
	return string(runes[:maxLen-1]) + "…"
}
