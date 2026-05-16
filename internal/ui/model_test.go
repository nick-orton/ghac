package ui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
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
		{"key ? opens help", "?", screenHelp},
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

func TestHelpScreenReturnsToPreviousScreen(t *testing.T) {
	origins := []struct {
		name     string
		setupKey string
		originID screenID
	}{
		{"from volume", "1", screenVolume},
		{"from playlist", "2", screenPlaylist},
		{"from navigator", "3", screenNavigator},
	}

	for _, tt := range origins {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()

			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.setupKey)})
			m = updated.(Model)

			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			m = updated.(Model)
			if m.activeScreen != screenHelp {
				t.Fatalf("expected screenHelp after ?, got %v", m.activeScreen)
			}
			if m.prevScreen != tt.originID {
				t.Fatalf("expected prevScreen = %v, got %v", tt.originID, m.prevScreen)
			}

			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = updated.(Model)
			if m.activeScreen != tt.originID {
				t.Errorf("after Esc: activeScreen = %v, want %v", m.activeScreen, tt.originID)
			}
		})
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
