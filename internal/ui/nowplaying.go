package ui

import "fmt"

// NowPlayingView renders the now-playing bar at the given terminal width.
// In Phase 1 this is a placeholder; Phase 2 will populate it with real MPD data.
func NowPlayingView(width int) string {
	bar := "[ No song playing ]"
	line := fmt.Sprintf("%-*s", width, bar)
	return styleNowPlaying.Render(line)
}
