package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
	"ghac/internal/snapcast"
)

// newTestModel returns a root model with no backend clients (safe for unit tests).
func newTestModel() Model {
	return New(NewParams{})
}

func TestScreenSwitching(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		wantScreen screenID
	}{
		{"key 1 switches to volume", "1", screenVolume},
		{"key 2 switches to playlist", "2", screenPlaylist},
		{"key 3 switches to navigator", "3", screenNavigator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			got := updated.(Model).activeScreen
			if got != tt.wantScreen {
				t.Errorf("after pressing %q: activeScreen = %v, want %v", tt.key, got, tt.wantScreen)
			}
		})
	}
}

func TestHelpToggle(t *testing.T) {
	m := newTestModel()

	// ? opens the modal.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)
	if !m.showHelp {
		t.Error("showHelp should be true after pressing ?")
	}
	// activeScreen must not change.
	if m.activeScreen != screenVolume {
		t.Errorf("activeScreen = %v, want screenVolume", m.activeScreen)
	}

	// ? again closes the modal.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)
	if m.showHelp {
		t.Error("showHelp should be false after pressing ? again")
	}
}

func TestHelpEscClosesModal(t *testing.T) {
	origins := []struct {
		name     string
		key      string
		originID screenID
	}{
		{"from volume", "1", screenVolume},
		{"from playlist", "2", screenPlaylist},
		{"from navigator", "3", screenNavigator},
	}

	for _, tt := range origins {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()

			// Go to origin screen.
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			m = updated.(Model)

			// Open help modal.
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			m = updated.(Model)
			if !m.showHelp {
				t.Fatalf("showHelp should be true after pressing ?")
			}
			if m.activeScreen != tt.originID {
				t.Fatalf("activeScreen changed to %v, want %v", m.activeScreen, tt.originID)
			}

			// Esc closes the modal; activeScreen unchanged.
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = updated.(Model)
			if m.showHelp {
				t.Error("showHelp should be false after Esc")
			}
			if m.activeScreen != tt.originID {
				t.Errorf("after Esc: activeScreen = %v, want %v", m.activeScreen, tt.originID)
			}
		})
	}
}

func TestHelpModalSwallowsKeys(t *testing.T) {
	m := newTestModel()
	m.activeScreen = screenVolume

	// Open help modal.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)

	// Pressing "2" while modal is open should NOT switch screens.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(Model)
	if m.activeScreen != screenVolume {
		t.Errorf("screen changed to %v while modal was open, want screenVolume", m.activeScreen)
	}
	if !m.showHelp {
		t.Error("showHelp should still be true after swallowed key")
	}
}

func TestEscOnNonHelpScreenIsIgnored(t *testing.T) {
	m := newTestModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.activeScreen != screenVolume {
		t.Errorf("Esc on volume screen changed screen to %v", m.activeScreen)
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestQuitKeys(t *testing.T) {
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	}

	for _, key := range keys {
		m := newTestModel()
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Errorf("expected quit command for key %v, got nil", key)
			continue
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("expected tea.QuitMsg for key %v, got %T", key, msg)
		}
	}
}

func TestQuitWhileHelpModalOpen(t *testing.T) {
	m := newTestModel()

	// Open help modal.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)

	// q should still quit even when modal is open.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit cmd while help modal open, got nil")
		return
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestWindowSizeStored(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	if m.width != 120 || m.height != 40 {
		t.Errorf("width/height = %d/%d, want 120/40", m.width, m.height)
	}
}

func TestMsgPlayerStateUpdatesModel(t *testing.T) {
	m := newTestModel()

	msg := mpd.MsgPlayerState{
		Status: "play",
		Song: mpd.Song{
			Title:  "Test Song",
			Artist: "Test Artist",
			Album:  "Test Album",
			File:   "test/song.flac",
		},
		Elapsed:       30 * time.Second,
		TotalDuration: 3*time.Minute + 30*time.Second,
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.playerStatus != "play" {
		t.Errorf("playerStatus = %q, want %q", m.playerStatus, "play")
	}
	if m.currentSong.Title != "Test Song" {
		t.Errorf("currentSong.Title = %q, want %q", m.currentSong.Title, "Test Song")
	}
	if m.elapsed != 30*time.Second {
		t.Errorf("elapsed = %v, want %v", m.elapsed, 30*time.Second)
	}
	if m.totalDuration != 3*time.Minute+30*time.Second {
		t.Errorf("totalDuration = %v, want %v", m.totalDuration, 3*time.Minute+30*time.Second)
	}
}

func TestMsgTickAdvancesElapsedWhenPlaying(t *testing.T) {
	m := newTestModel()
	m.playerStatus = "play"
	m.elapsed = 10 * time.Second

	updated, cmd := m.Update(mpd.MsgTick{Time: time.Now()})
	m = updated.(Model)

	if m.elapsed != 11*time.Second {
		t.Errorf("elapsed = %v, want %v", m.elapsed, 11*time.Second)
	}
	// Should re-subscribe to tick.
	if cmd == nil {
		t.Error("expected tick re-subscription cmd, got nil")
	}
}

func TestMsgTickDoesNotAdvanceWhenPaused(t *testing.T) {
	m := newTestModel()
	m.playerStatus = "pause"
	m.elapsed = 10 * time.Second

	updated, cmd := m.Update(mpd.MsgTick{Time: time.Now()})
	m = updated.(Model)

	if m.elapsed != 10*time.Second {
		t.Errorf("elapsed = %v, want %v (should not advance when paused)", m.elapsed, 10*time.Second)
	}
	// Ticker must still re-subscribe even when paused.
	if cmd == nil {
		t.Error("expected tick re-subscription cmd even when paused, got nil")
	}
}

func TestMsgErrorSetsErrMsgAndQuits(t *testing.T) {
	m := newTestModel()

	updated, cmd := m.Update(mpd.MsgError{Err: errors.New("connection refused")})
	m = updated.(Model)

	if m.errMsg == "" {
		t.Error("errMsg should be set after MsgError")
	}
	if cmd == nil {
		t.Error("expected quit cmd after MsgError, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg after MsgError, got %T", msg)
	}
}

func TestPlayKeyIgnoredWithNoClient(t *testing.T) {
	// With a nil client, pressing p should not panic and should not emit a cmd.
	m := newTestModel()
	m.playerStatus = "pause"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	_ = cmd // cmd is nil when no client; that's fine
	_ = m
}

func TestSnapcastMsgErrorSetsErrMsgAndQuits(t *testing.T) {
	m := newTestModel()

	updated, cmd := m.Update(snapcast.MsgError{Err: errors.New("snapcast disconnected")})
	m = updated.(Model)

	if m.errMsg == "" {
		t.Error("errMsg should be set after snapcast.MsgError")
	}
	if cmd == nil {
		t.Error("expected quit cmd after snapcast.MsgError, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg after snapcast.MsgError, got %T", msg)
	}
}

func TestErrorViewRendersMessage(t *testing.T) {
	m := newTestModel()
	m.errMsg = "mpd connection lost"

	view := m.View()

	if !strings.Contains(view, "Error: mpd connection lost") {
		t.Errorf("View() should contain the error message, got: %q", view)
	}
}

func TestErrorViewDoesNotRenderScreenContent(t *testing.T) {
	m := newTestModel()
	m.errMsg = "mpd connection lost"

	view := m.View()

	// Screen content and navigation should not appear when an error is set.
	if strings.Contains(view, "1:Volume") {
		t.Error("View() should not contain tab strip when errMsg is set")
	}
	if strings.Contains(view, "Player Volume") {
		t.Error("View() should not contain screen title when errMsg is set")
	}
}

func TestBackendMessagesProcessedWhileHelpModalOpen(t *testing.T) {
	m := newTestModel()
	m.playerStatus = "play"
	m.elapsed = 5 * time.Second

	// Open help modal.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(Model)
	if !m.showHelp {
		t.Fatal("showHelp should be true")
	}

	// MsgTick should still advance elapsed even while the modal is open.
	updated, cmd := m.Update(mpd.MsgTick{Time: time.Now()})
	m = updated.(Model)
	if m.elapsed != 6*time.Second {
		t.Errorf("elapsed = %v while modal open, want 6s", m.elapsed)
	}
	if cmd == nil {
		t.Error("expected tick re-subscription cmd, got nil")
	}
	if !m.showHelp {
		t.Error("showHelp should remain true after backend message")
	}
}

func TestPlaceOverlay(t *testing.T) {
	strip := func(s string) string { return strings.ReplaceAll(s, "\x1b[0m", "") }

	t.Run("fg fits within bg height", func(t *testing.T) {
		bg := strings.Repeat("##########\n", 4) + "##########"
		fg := "AAAA\nBBBB\nCCCC"

		lines := strings.Split(placeOverlay(2, 1, fg, bg), "\n")
		if len(lines) != 5 {
			t.Fatalf("expected 5 lines, got %d", len(lines))
		}
		if lines[0] != "##########" {
			t.Errorf("row 0 = %q, want ##########", lines[0])
		}
		for i, want := range []string{"##AAAA####", "##BBBB####", "##CCCC####"} {
			if got := strip(lines[i+1]); got != want {
				t.Errorf("row %d = %q, want %q", i+1, got, want)
			}
		}
		if lines[4] != "##########" {
			t.Errorf("row 4 = %q, want ##########", lines[4])
		}
	})

	t.Run("fg extends beyond bg height", func(t *testing.T) {
		// Background is only 2 lines; modal starts at y=1 and is 3 lines tall.
		bg := "##########\n##########"
		fg := "AAAA\nBBBB\nCCCC"

		lines := strings.Split(placeOverlay(2, 1, fg, bg), "\n")
		// Output must have 4 rows (y=1 + 3 fg lines).
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines, got %d: %q", len(lines), lines)
		}
		if lines[0] != "##########" {
			t.Errorf("row 0 = %q, want ##########", lines[0])
		}
		// Row 1: bg still present, right portion shows.
		// Rows 2-3: bg exhausted, left is blank-padded.
		for i, want := range []string{"##AAAA####", "  BBBB", "  CCCC"} {
			got := strip(lines[i+1])
			if !strings.HasPrefix(got, strings.TrimRight(want, " ")) {
				t.Errorf("row %d = %q, want prefix %q", i+1, got, want)
			}
		}
	})
}

func TestRootModelPlayingSongRemovedThenStateAdvances(t *testing.T) {
	// Start with a 3-song playlist; song at pos 1 is playing.
	entries := []mpd.PlaylistEntry{
		{Song: mpd.Song{Title: "Alpha", File: "a.flac"}, Pos: 0},
		{Song: mpd.Song{Title: "Beta", File: "b.flac"}, Pos: 1},
		{Song: mpd.Song{Title: "Gamma", File: "c.flac"}, Pos: 2},
	}
	m := newTestModel()
	updated, _ := m.Update(mpd.MsgPlayerState{
		Status:  "play",
		Song:    mpd.Song{Title: "Beta", File: "b.flac"},
		SongPos: 1,
	})
	m = updated.(Model)
	updated, _ = m.Update(mpd.MsgPlaylistChanged{Entries: entries})
	m = updated.(Model)

	// Simulate: Beta is removed; playlist now has Alpha and Gamma.
	// MPD advances to what was pos 2 (Gamma), now at pos 1.
	reducedEntries := []mpd.PlaylistEntry{
		{Song: mpd.Song{Title: "Alpha", File: "a.flac"}, Pos: 0},
		{Song: mpd.Song{Title: "Gamma", File: "c.flac"}, Pos: 1},
	}
	updated, _ = m.Update(mpd.MsgPlaylistChanged{Entries: reducedEntries})
	m = updated.(Model)
	updated, _ = m.Update(mpd.MsgPlayerState{
		Status:  "play",
		Song:    mpd.Song{Title: "Gamma", File: "c.flac"},
		SongPos: 1,
	})
	m = updated.(Model)

	pl := m.screens[screenPlaylist].(playlistScreen)
	if m.currentSongPos != 1 {
		t.Errorf("currentSongPos = %d, want 1 after MPD advanced past removed song", m.currentSongPos)
	}
	if pl.currentPos != 1 {
		t.Errorf("playlist.currentPos = %d, want 1", pl.currentPos)
	}
	if len(pl.entries) != 2 {
		t.Errorf("playlist.entries len = %d, want 2", len(pl.entries))
	}
	if pl.entries[1].Title != "Gamma" {
		t.Errorf("entries[1].Title = %q, want Gamma", pl.entries[1].Title)
	}
}
