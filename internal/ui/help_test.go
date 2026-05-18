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

func TestHelpUpdatePassesThroughMessages(t *testing.T) {
	h := newHelpScreen()
	got, cmd := h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		t.Errorf("helpScreen.Update returned non-nil cmd for unhandled key")
	}
	_ = got
}
