package ui

import (
	"strings"
	"testing"
	"time"
)

func TestNowPlayingViewStopState(t *testing.T) {
	state := PlayerState{Status: "stop"}
	view := NowPlayingView(state, 80)
	if !strings.Contains(view, "No song playing") {
		t.Errorf("expected placeholder text, got: %q", view)
	}
}

func TestNowPlayingViewEmptyStatus(t *testing.T) {
	state := PlayerState{}
	view := NowPlayingView(state, 80)
	if !strings.Contains(view, "No song playing") {
		t.Errorf("expected placeholder for empty status, got: %q", view)
	}
}

func TestNowPlayingViewPlayingShowsTitleAndArtist(t *testing.T) {
	state := PlayerState{
		Status:        "play",
		Title:         "Echoes",
		Artist:        "Pink Floyd",
		Album:         "Meddle",
		File:          "pf/echoes.flac",
		Elapsed:       2*time.Minute + 3*time.Second,
		TotalDuration: 23*time.Minute + 31*time.Second,
	}
	view := NowPlayingView(state, 120)
	if !strings.Contains(view, "Echoes") {
		t.Errorf("expected title in view, got: %q", view)
	}
	if !strings.Contains(view, "Pink Floyd") {
		t.Errorf("expected artist in view, got: %q", view)
	}
	if !strings.Contains(view, "2:03") {
		t.Errorf("expected elapsed time in view, got: %q", view)
	}
}

func TestNowPlayingViewShowsAlbum(t *testing.T) {
	state := PlayerState{
		Status:        "play",
		Title:         "Echoes",
		Artist:        "Pink Floyd",
		Album:         "Meddle",
		Elapsed:       2*time.Minute + 3*time.Second,
		TotalDuration: 23*time.Minute + 31*time.Second,
	}
	view := NowPlayingView(state, 120)
	if !strings.Contains(view, "Meddle") {
		t.Errorf("expected album in view, got: %q", view)
	}
}

func TestNowPlayingViewPausedShowsIcon(t *testing.T) {
	state := PlayerState{
		Status:        "pause",
		Title:         "Comfortably Numb",
		Artist:        "Pink Floyd",
		Elapsed:       30 * time.Second,
		TotalDuration: 6*time.Minute + 23*time.Second,
	}
	view := NowPlayingView(state, 120)
	// Paused icon should be present.
	if !strings.Contains(view, "⏸") {
		t.Errorf("expected pause icon in view, got: %q", view)
	}
}

func TestNowPlayingViewFileFallback(t *testing.T) {
	// No title or artist — should fall back to filename.
	state := PlayerState{
		Status: "play",
		File:   "misc/untitled.mp3",
	}
	view := NowPlayingView(state, 120)
	if !strings.Contains(view, "misc/untitled.mp3") {
		t.Errorf("expected filename fallback in view, got: %q", view)
	}
}

func TestNowPlayingViewTitleOnlyFallback(t *testing.T) {
	// Title present but no artist — should show title without separator.
	state := PlayerState{
		Status: "play",
		Title:  "Unknown Artist Track",
	}
	view := NowPlayingView(state, 120)
	if !strings.Contains(view, "Unknown Artist Track") {
		t.Errorf("expected title in view, got: %q", view)
	}
	if strings.Contains(view, "–") {
		t.Errorf("unexpected separator when artist is absent, got: %q", view)
	}
}

func TestNowPlayingViewZeroWidthSafe(t *testing.T) {
	// Width 0 should not panic.
	state := PlayerState{Status: "play", Title: "Song", Artist: "Artist"}
	_ = NowPlayingView(state, 0)
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
		total   time.Duration
		width   int
		wantMin int // minimum filled characters
		wantMax int // maximum filled characters
	}{
		{"0% progress", 0, time.Minute, 10, 0, 0},
		{"50% progress", 30 * time.Second, time.Minute, 10, 4, 6},
		{"100% progress", time.Minute, time.Minute, 10, 10, 10},
		{"zero total", 0, 0, 10, 0, 10}, // all unfilled
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := progressBar(tt.elapsed, tt.total, tt.width)
			// Strip ANSI codes for rune counting by counting █ characters.
			filled := strings.Count(bar, "█")
			if filled < tt.wantMin || filled > tt.wantMax {
				t.Errorf("filled = %d, want between %d and %d; bar = %q",
					filled, tt.wantMin, tt.wantMax, bar)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00"},
		{30 * time.Second, "0:30"},
		{time.Minute + 7*time.Second, "1:07"},
		{3*time.Minute + 45*time.Second, "3:45"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
	}
	for _, tt := range tests {
		got := formatTime(tt.d)
		if got != tt.want {
			t.Errorf("formatTime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
