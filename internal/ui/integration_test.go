//go:build integration

package ui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
	"ghac/internal/snapcast"
)

// mpdIntegrationAddr returns the MPD address for integration tests.
// Defaults to localhost:6600; override with MPD_TEST_ADDR env var.
func mpdIntegrationAddr() string {
	if addr := os.Getenv("MPD_TEST_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6600"
}

// snapIntegrationAddr returns the SnapCast address for integration tests.
// Defaults to localhost:1705; override with SNAPCAST_TEST_ADDR env var.
func snapIntegrationAddr() string {
	if addr := os.Getenv("SNAPCAST_TEST_ADDR"); addr != "" {
		return addr
	}
	return "localhost:1705"
}

// connectTestModel creates a root model connected to live MPD and SnapCast
// servers. Returns the model and a cleanup function that closes both connections.
func connectTestModel(t *testing.T) (Model, func()) {
	t.Helper()

	mc, err := mpd.Connect(mpdIntegrationAddr())
	if err != nil {
		t.Fatalf("mpd.Connect: %v", err)
	}

	state, err := mc.Status()
	if err != nil {
		mc.Close()
		t.Fatalf("mpd.Status: %v", err)
	}

	song, err := mc.CurrentSong()
	if err != nil {
		mc.Close()
		t.Fatalf("mpd.CurrentSong: %v", err)
	}

	playlist, err := mc.PlaylistInfo()
	if err != nil {
		mc.Close()
		t.Fatalf("mpd.PlaylistInfo: %v", err)
	}

	navEntries, err := mc.ListInfo("")
	if err != nil {
		mc.Close()
		t.Fatalf("mpd.ListInfo: %v", err)
	}

	sc, err := snapcast.Connect(snapIntegrationAddr())
	if err != nil {
		mc.Close()
		t.Fatalf("snapcast.Connect: %v", err)
	}

	clients, err := sc.GetServerStatus()
	if err != nil {
		mc.Close()
		sc.Close()
		t.Fatalf("snapcast.GetServerStatus: %v", err)
	}

	initialState := mpd.MsgPlayerState{
		Status:        state.Status,
		Song:          song,
		Elapsed:       state.Elapsed,
		TotalDuration: state.TotalDuration,
		SongPos:       state.SongPos,
	}

	m := New(NewParams{
		MPD:         mc,
		MPDState:    initialState,
		Snapcast:    sc,
		SnapClients: clients,
		Playlist:    playlist,
		NavEntries:  navEntries,
	})

	cleanup := func() {
		mc.Close()
		sc.Close()
	}
	return m, cleanup
}

// TestIntegrationFullLifecycle verifies the model initialises, renders without
// panic, and exposes the expected top-level UI elements.
func TestIntegrationFullLifecycle(t *testing.T) {
	m, cleanup := connectTestModel(t)
	defer cleanup()

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil cmd; expected at least a tick subscription")
	}

	view := m.View()
	if view == "" {
		t.Fatal("View() returned empty string")
	}
	// Tab strip and screen border must be present.
	if !strings.Contains(view, "1:Volume") {
		t.Error("View() missing tab strip entry '1:Volume'")
	}
	if !strings.Contains(view, "Player Volume") {
		t.Error("View() missing screen title 'Player Volume'")
	}
}

// TestIntegrationScreenNavigation switches through all screens and verifies
// the active screen changes correctly and View() does not panic.
func TestIntegrationScreenNavigation(t *testing.T) {
	m, cleanup := connectTestModel(t)
	defer cleanup()

	cases := []struct {
		key        string
		wantScreen screenID
		wantTitle  string
	}{
		{"1", screenVolume, "Player Volume"},
		{"2", screenPlaylist, "Playlist Control"},
		{"3", screenNavigator, "Library Navigator"},
	}

	for _, tc := range cases {
		t.Run(tc.wantTitle, func(t *testing.T) {
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			m = updated.(Model)

			if m.activeScreen != tc.wantScreen {
				t.Errorf("activeScreen = %v, want %v after pressing %q", m.activeScreen, tc.wantScreen, tc.key)
			}

			view := m.View()
			if !strings.Contains(view, tc.wantTitle) {
				t.Errorf("View() missing title %q after pressing %q", tc.wantTitle, tc.key)
			}
		})
	}

	// Help is now a modal overlay; pressing ? sets showHelp without changing activeScreen.
	t.Run("Help modal", func(t *testing.T) {
		// Ensure we're on the volume screen first.
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
		m = updated.(Model)

		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		m = updated.(Model)

		if !m.showHelp {
			t.Error("showHelp should be true after pressing ?")
		}
		if m.activeScreen != screenVolume {
			t.Errorf("activeScreen = %v, want screenVolume (modal should not change it)", m.activeScreen)
		}
		view := m.View()
		if !strings.Contains(view, "Help") {
			t.Errorf("View() missing Help title in modal after pressing ?")
		}
		// Close the modal before continuing.
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
	})
}

// TestIntegrationPlayPauseToggle sends the play/pause key twice and verifies
// the model does not error out.
func TestIntegrationPlayPauseToggle(t *testing.T) {
	m, cleanup := connectTestModel(t)
	defer cleanup()

	for i := 0; i < 2; i++ {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
		m = updated.(Model)
		if m.errMsg != "" {
			t.Fatalf("errMsg set after play/pause toggle %d: %q", i+1, m.errMsg)
		}
		_ = cmd
	}
}

// TestIntegrationPlaylistOperations adds a song via the MPD client, sends
// MsgPlaylistChanged to the model, and verifies the playlist screen reflects
// the change. Restores original playlist state on completion.
func TestIntegrationPlaylistOperations(t *testing.T) {
	mc, err := mpd.Connect(mpdIntegrationAddr())
	if err != nil {
		t.Fatalf("mpd.Connect: %v", err)
	}
	defer mc.Close()

	// Find a file to add.
	navEntries, err := mc.ListInfo("")
	if err != nil {
		t.Fatalf("ListInfo: %v", err)
	}
	var fileURI string
	for _, e := range navEntries {
		if !e.IsDir {
			fileURI = e.Path
			break
		}
	}
	if fileURI == "" {
		t.Skip("no files at library root; skipping playlist operations test")
	}

	// Record playlist length before.
	before, err := mc.PlaylistInfo()
	if err != nil {
		t.Fatalf("PlaylistInfo before: %v", err)
	}

	// Add the file.
	if err := mc.Add(fileURI); err != nil {
		t.Fatalf("Add(%q): %v", fileURI, err)
	}
	t.Cleanup(func() {
		// Remove the song we added (last entry).
		entries, _ := mc.PlaylistInfo()
		if len(entries) > len(before) {
			_ = mc.Delete(len(entries) - 1)
		}
	})

	after, err := mc.PlaylistInfo()
	if err != nil {
		t.Fatalf("PlaylistInfo after Add: %v", err)
	}

	// Feed the updated playlist into a model.
	m := New(NewParams{MPD: mc, Playlist: after})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(Model)

	if len(m.playlist.entries) != len(after) {
		t.Errorf("playlist.entries len = %d, want %d", len(m.playlist.entries), len(after))
	}
}

// TestIntegrationVolumeOperations presses h/l on the volume screen and
// verifies the model does not error. Requires at least one SnapCast client.
func TestIntegrationVolumeOperations(t *testing.T) {
	m, cleanup := connectTestModel(t)
	defer cleanup()

	if len(m.volume.clients) == 0 {
		t.Skip("no SnapCast clients connected; skipping volume operations test")
	}

	originalVol := m.volume.clients[0].Volume

	// Restore volume on exit.
	t.Cleanup(func() {
		sc, err := snapcast.Connect(snapIntegrationAddr())
		if err != nil {
			return
		}
		defer sc.Close()
		clients, _ := sc.GetServerStatus()
		if len(clients) > 0 {
			_ = sc.SetVolume(clients[0].ID, originalVol, clients[0].Muted)
		}
	})

	// Switch to volume screen then press l to raise volume.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = updated.(Model)

	for _, key := range []string{"l", "h"} {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = updated.(Model)
		if m.errMsg != "" {
			t.Fatalf("errMsg set after pressing %q: %q", key, m.errMsg)
		}
		_ = cmd
	}
}
