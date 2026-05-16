package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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
			m := New()
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
		m := New()
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
		name      string
		setupKey  string
		originID  screenID
	}{
		{"from volume", "1", screenVolume},
		{"from playlist", "2", screenPlaylist},
		{"from navigator", "3", screenNavigator},
	}

	for _, tt := range origins {
		t.Run(tt.name, func(t *testing.T) {
			m := New()

			// Navigate to origin screen.
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.setupKey)})
			m = updated.(Model)

			// Open help.
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			m = updated.(Model)
			if m.activeScreen != screenHelp {
				t.Fatalf("expected screenHelp after ?, got %v", m.activeScreen)
			}
			if m.prevScreen != tt.originID {
				t.Fatalf("expected prevScreen = %v, got %v", tt.originID, m.prevScreen)
			}

			// Press Esc to return.
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = updated.(Model)
			if m.activeScreen != tt.originID {
				t.Errorf("after Esc: activeScreen = %v, want %v", m.activeScreen, tt.originID)
			}
		})
	}
}

func TestEscOnNonHelpScreenIsIgnored(t *testing.T) {
	m := New() // starts on screenVolume
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.activeScreen != screenVolume {
		t.Errorf("Esc on volume screen changed screen to %v", m.activeScreen)
	}
	if cmd != nil {
		// cmd may be nil since volumeScreen.Update returns nil; that's fine.
		_ = cmd
	}
}

func TestWindowSizeStored(t *testing.T) {
	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	if m.width != 120 || m.height != 40 {
		t.Errorf("width/height = %d/%d, want 120/40", m.width, m.height)
	}
}
