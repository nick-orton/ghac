package mpd

import (
	"fmt"
	"strconv"
	"strings"
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

	idle, err := gompd.NewWatcher("tcp", addr, "", "player", "playlist")
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

// PlaylistInfo returns all songs currently in the MPD playback queue.
func (c *Client) PlaylistInfo() ([]PlaylistEntry, error) {
	attrs, err := c.cmd.PlaylistInfo(-1, -1)
	if err != nil {
		return nil, fmt.Errorf("mpd PlaylistInfo: %w", err)
	}
	entries := make([]PlaylistEntry, len(attrs))
	for i, a := range attrs {
		pos, _ := strconv.Atoi(a["Pos"])
		entries[i] = PlaylistEntry{
			Song: songFromAttrs(a),
			Pos:  pos,
		}
	}
	return entries, nil
}

// PlayAt starts playing the song at the given 0-indexed playlist position.
func (c *Client) PlayAt(pos int) error {
	return c.cmd.Play(pos)
}

// Delete removes the song at the given 0-indexed playlist position.
func (c *Client) Delete(pos int) error {
	return c.cmd.Delete(pos, -1)
}

// Clear removes all songs from the playlist and stops playback.
func (c *Client) Clear() error {
	return c.cmd.Clear()
}

// Move moves the song at position from to position to in the playlist.
func (c *Client) Move(from, to int) error {
	return c.cmd.Move(from, -1, to)
}

// ListInfo lists the contents of the directory at path. Use an empty string
// for the root of the music library. Playlist entries in the response are
// skipped. Note: gompd's ListInfo lowercases all attribute keys.
func (c *Client) ListInfo(path string) ([]DirEntry, error) {
	attrs, err := c.cmd.ListInfo(path)
	if err != nil {
		return nil, fmt.Errorf("mpd ListInfo: %w", err)
	}
	entries := make([]DirEntry, 0, len(attrs))
	for _, a := range attrs {
		if uri, ok := a["directory"]; ok {
			entries = append(entries, DirEntry{
				Name:  mpdBase(uri),
				Path:  uri,
				IsDir: true,
			})
		} else if uri, ok := a["file"]; ok {
			entries = append(entries, DirEntry{
				Name:  mpdBase(uri),
				Path:  uri,
				IsDir: false,
				// gompd's ListInfo lowercases all attribute keys (unlike
				// PlaylistInfo/CurrentSong which preserve MPD's capitalization).
				// Do NOT replace this with songFromAttrs — it uses "Title" etc.
				Song: Song{
					Title:  a["title"],
					Artist: a["artist"],
					Album:  a["album"],
					File:   uri,
				},
			})
		}
		// playlist entries are skipped
	}
	return entries, nil
}

// Add appends the song or directory at uri to the MPD playback queue.
// MPD enqueues directories recursively.
func (c *Client) Add(uri string) error {
	return c.cmd.Add(uri)
}

// mpdBase returns the final path segment of an MPD URI, which always uses
// forward slashes regardless of the operating system.
func mpdBase(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// ListenIdle returns a tea.Cmd that blocks until the next MPD player or
// playlist event, queries the relevant state, and returns the appropriate
// message (MsgPlayerState or MsgPlaylistChanged). Call the returned Cmd
// again from Update to keep listening.
func (c *Client) ListenIdle() tea.Cmd {
	return func() tea.Msg {
		select {
		case subsystem, ok := <-c.idle.Event:
			if !ok || subsystem == "" {
				return MsgError{Err: fmt.Errorf("mpd idle channel closed")}
			}
			switch subsystem {
			case "playlist":
				entries, err := c.PlaylistInfo()
				if err != nil {
					return MsgError{Err: err}
				}
				return MsgPlaylistChanged{Entries: entries}
			default: // "player" and other subsystems map to player state
				state, err := c.queryPlayerState()
				if err != nil {
					return MsgError{Err: err}
				}
				return state
			}
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

	// status["song"] is absent when the playlist is empty or MPD is stopped
	// with no current position; use -1 as the sentinel for "none playing".
	songPos := -1
	if s := status["song"]; s != "" {
		if p, err := strconv.Atoi(s); err == nil {
			songPos = p
		}
	}

	return MsgPlayerState{
		Status:        status["state"],
		Song:          song,
		Elapsed:       parseDuration(status["elapsed"]),
		TotalDuration: parseDuration(status["duration"]),
		SongPos:       songPos,
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
