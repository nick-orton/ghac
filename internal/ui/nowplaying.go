package ui

import (
	"fmt"
	"strings"
	"time"
)

// PlayerState holds the data needed to render the now-playing bar.
// The root model populates this from its MPD message fields.
type PlayerState struct {
	Status        string // "play", "pause", "stop"
	Title         string
	Artist        string
	Album         string
	File          string // fallback when metadata absent
	Elapsed       time.Duration
	TotalDuration time.Duration
}

// displayName returns the best available label for the current song.
// Prefers "Title – Artist – Album", degrades gracefully when fields are absent,
// and falls back to filename when no metadata is available.
func (p PlayerState) displayName() string {
	if p.Title != "" && p.Artist != "" && p.Album != "" {
		return p.Title + " – " + p.Artist + " – " + p.Album
	}
	if p.Title != "" && p.Artist != "" {
		return p.Title + " – " + p.Artist
	}
	if p.Title != "" {
		return p.Title
	}
	if p.File != "" {
		return p.File
	}
	return ""
}

// NowPlayingView renders the now-playing bar at the given terminal width.
// When nothing is playing it shows a placeholder. When a song is active it
// shows: [state] title – artist  ━━━━━━░░░░  0:23 / 3:45
func NowPlayingView(state PlayerState, width int) string {
	if width <= 0 {
		width = 80
	}

	if state.Status == "" || state.Status == "stop" || state.displayName() == "" {
		placeholder := "[ No song playing ]"
		line := fmt.Sprintf("%-*s", width, placeholder)
		return styleNowPlaying.Render(line)
	}

	stateIcon := "▶"
	if state.Status == "pause" {
		stateIcon = "⏸"
	}

	name := state.displayName()
	// timeRaw is used for layout width calculation (no ANSI codes).
	// The styled version is used in the final rendered line.
	timeRaw := formatTime(state.Elapsed) + " / " + formatTime(state.TotalDuration)
	progressBarWidth := 20
	// overhead: icon(1) + space(1) + bar(progressBarWidth) + space(1) + time
	overhead := 1 + 1 + progressBarWidth + 1 + len(timeRaw)
	if overhead >= width {
		// Terminal too narrow: show just name truncated.
		line := fmt.Sprintf("%-*s", width, stateIcon+" "+name)
		return styleNowPlaying.Render(truncate(line, width))
	}

	nameWidth := width - overhead
	name = truncate(name, nameWidth)
	name = fmt.Sprintf("%-*s", nameWidth, name)

	bar := progressBar(state.Elapsed, state.TotalDuration, progressBarWidth)
	styledTime := styleTime.Render(timeRaw)

	line := stateIcon + " " + name + bar + " " + styledTime
	return styleNowPlaying.Render(line)
}

// progressBar renders a fixed-width progress bar using block characters.
// filled portion uses "━", unfilled uses "─".
func progressBar(elapsed, total time.Duration, width int) string {
	if width <= 0 {
		return ""
	}
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	ratio := float64(elapsed) / float64(total)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	empty := width - filled
	return styleProgressFill.Render(strings.Repeat("█", filled)) +
		styleProgressEmpty.Render(strings.Repeat("░", empty))
}

// formatTime formats a duration as m:ss (e.g. "3:07"). Handles durations
// longer than an hour by showing h:mm:ss.
func formatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSec := int(d.Seconds())
	sec := totalSec % 60
	min := (totalSec / 60) % 60
	hrs := totalSec / 3600
	if hrs > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hrs, min, sec)
	}
	return fmt.Sprintf("%d:%02d", min, sec)
}

// truncate cuts s to at most maxRunes runes, appending "…" if truncated.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "…"
}
