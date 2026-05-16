package mpd

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	gompd "github.com/fhs/gompd/v2/mpd"
)

// Client wraps two gompd connections: one for commands and one for idle
// watching. The command connection must not be shared with any goroutine;
// it is called synchronously from the Bubble Tea Update loop. The idle
// connection is used exclusively by the ListenIdle goroutine.
type Client struct {
	cmd  *gompd.Client
	idle *gompd.Watcher
	addr string
}

// Connect dials MPD at addr (e.g. "192.168.1.10:6600"), establishing both
// the command connection and the idle watcher. Returns a ready-to-use
// Client. The caller must call Close() when done.
func Connect(addr string) (*Client, error) {
	cmd, err := gompd.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("mpd command connection: %w", err)
	}

	idle, err := gompd.NewWatcher("tcp", addr, "", "player")
	if err != nil {
		_ = cmd.Close()
		return nil, fmt.Errorf("mpd idle connection: %w", err)
	}

	return &Client{cmd: cmd, idle: idle, addr: addr}, nil
}

// Close closes both MPD connections.
func (c *Client) Close() {
	_ = c.idle.Close()
	_ = c.cmd.Close()
}

// Ping sends a no-op command to keep the command connection alive.
func (c *Client) Ping() error {
	return c.cmd.Ping()
}

// Status returns the current player state as a MsgPlayerState.
func (c *Client) Status() (MsgPlayerState, error) {
	return c.queryPlayerState()
}

// CurrentSong returns metadata for the currently playing song.
func (c *Client) CurrentSong() (Song, error) {
	attrs, err := c.cmd.CurrentSong()
	if err != nil {
		return Song{}, fmt.Errorf("mpd CurrentSong: %w", err)
	}
	return songFromAttrs(attrs), nil
}

// Play resumes or starts playback. If MPD is paused it unpauses; if stopped
// it begins from the current position (or the first song).
func (c *Client) Play() error {
	// MPD's `play` without a position resumes where it left off.
	return c.cmd.Play(-1)
}

// Pause pauses playback. Has no effect when already paused.
func (c *Client) Pause() error {
	return c.cmd.Pause(true)
}

// ListenIdle returns a tea.Cmd that blocks until the next MPD player event,
// queries the full player state, and returns a MsgPlayerState (or MsgError on
// failure). Call the returned Cmd again from Update to keep listening.
func (c *Client) ListenIdle() tea.Cmd {
	return func() tea.Msg {
		select {
		case subsystem, ok := <-c.idle.Event:
			if !ok || subsystem == "" {
				return MsgError{Err: fmt.Errorf("mpd idle channel closed")}
			}
			state, err := c.queryPlayerState()
			if err != nil {
				return MsgError{Err: err}
			}
			return state
		case err := <-c.idle.Error:
			if err != nil {
				return MsgError{Err: fmt.Errorf("mpd idle error: %w", err)}
			}
			return MsgError{Err: fmt.Errorf("mpd idle: unknown error")}
		}
	}
}

// queryPlayerState fetches status + current song from the command connection
// and assembles a MsgPlayerState.
func (c *Client) queryPlayerState() (MsgPlayerState, error) {
	status, err := c.cmd.Status()
	if err != nil {
		return MsgPlayerState{}, fmt.Errorf("mpd Status: %w", err)
	}

	song, err := c.CurrentSong()
	if err != nil {
		return MsgPlayerState{}, err
	}

	return MsgPlayerState{
		Status:        status["state"],
		Song:          song,
		Elapsed:       parseDuration(status["elapsed"]),
		TotalDuration: parseDuration(status["duration"]),
	}, nil
}

// songFromAttrs converts gompd Attrs to a Song, using File as the fallback
// when title metadata is absent.
func songFromAttrs(attrs gompd.Attrs) Song {
	return Song{
		Title:  attrs["Title"],
		Artist: attrs["Artist"],
		Album:  attrs["Album"],
		File:   attrs["file"],
	}
}

// parseDuration parses an MPD time value (float seconds, e.g. "123.456")
// into a time.Duration. Returns 0 on empty input or parse error.
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	secs, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}
