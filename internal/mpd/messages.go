package mpd

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Song holds metadata for a single track.
type Song struct {
	Title  string
	Artist string
	Album  string
	File   string // fallback display when metadata is absent
}

// MsgPlayerState is emitted by the idle listener when the MPD player
// state changes. It carries the complete current player state.
type MsgPlayerState struct {
	Status        string // "play", "pause", "stop"
	Song          Song
	Elapsed       time.Duration
	TotalDuration time.Duration
}

// MsgTick is sent every second to advance the progress bar.
type MsgTick struct {
	Time time.Time
}

// MsgError is sent when the MPD connection is lost or an error occurs.
type MsgError struct {
	Err error
}

// TickCmd returns a tea.Cmd that fires a MsgTick after one second.
// Call it again from Update to keep the ticker running.
func TickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return MsgTick{Time: t}
	})
}
