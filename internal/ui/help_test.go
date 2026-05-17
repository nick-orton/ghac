package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpViewContainsSections(t *testing.T) {
	h := newHelpScreen()
	view := h.View()

	sections := []string{"Global", "Player Volume", "Playlist Control", "Library Navigator"}
	for _, s := range sections {
		if !strings.Contains(view, s) {
			t.Errorf("help view missing section %q", s)
		}
	}
}

func TestHelpViewContainsAllGlobalKeys(t *testing.T) {
	h := newHelpScreen()
	view := h.View()

	keys := []string{"1", "2", "3", "?", "p", "q", "Esc"}
	for _, k := range keys {
		if !strings.Contains(view, k) {
			t.Errorf("help view missing key %q", k)
		}
	}
}

func TestHelpEscReturnsToOrigin(t *testing.T) {
	// Esc on help is handled in the root model. Verify root model behavior
	// for all three origin screens.
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
			m := New(NewParams{})

			// Go to origin.
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			m = updated.(Model)

			// Open help.
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
			m = updated.(Model)

			// Esc should return to origin.
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m = updated.(Model)

			if m.activeScreen != tt.originID {
				t.Errorf("Esc from help: activeScreen = %v, want %v", m.activeScreen, tt.originID)
			}
		})
	}
}

func TestHelpUpdatePassesThroughMessages(t *testing.T) {
	h := newHelpScreen()
	got, cmd := h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		t.Errorf("helpScreen.Update returned non-nil cmd for unhandled key")
	}
	_ = got
}
