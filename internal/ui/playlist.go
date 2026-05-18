package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
)

// playlistScreen is the Playlist Control screen. It displays the MPD
// playback queue and supports cursor navigation, song selection, removal,
// clear, and playback jump.
type playlistScreen struct {
	entries    []mpd.PlaylistEntry
	cursor     int
	offset     int         // index of the first visible entry (viewport top)
	pendingG   bool        // true after a single 'g' press, waiting for 'gg'
	pendingF   bool        // true after 'f' press, waiting for letter to jump to
	selected   map[int]bool
	currentPos int         // playlist position of currently-playing song; -1 if none
	height     int         // terminal height for viewport sizing
	mc         *mpd.Client // may be nil in tests; commands become no-ops
}

func newPlaylistScreen(mc *mpd.Client, entries []mpd.PlaylistEntry, currentPos int) playlistScreen {
	return playlistScreen{
		entries:    entries,
		cursor:     0,
		selected:   make(map[int]bool),
		currentPos: currentPos,
		mc:         mc,
	}
}

// withEntries returns a copy with the playlist replaced, selection cleared,
// cursor clamped, and current playing position updated atomically.
func (s playlistScreen) withEntries(entries []mpd.PlaylistEntry, currentPos int) playlistScreen {
	s.entries = entries
	s.currentPos = currentPos
	s.selected = make(map[int]bool)
	if len(entries) == 0 {
		s.cursor = 0
	} else if s.cursor >= len(entries) {
		s.cursor = len(entries) - 1
	}
	return s.clampOffset()
}

// withCurrentPos returns a copy with the current playing position updated.
func (s playlistScreen) withCurrentPos(pos int) playlistScreen {
	s.currentPos = pos
	return s
}

// withHeight returns a copy with the terminal height updated and the viewport
// offset re-clamped so the cursor remains visible.
func (s playlistScreen) withHeight(h int) playlistScreen {
	s.height = h
	return s.clampOffset()
}

// viewportHeight returns the number of entry rows that fit on screen.
// The overhead is: nowplaying(1) + sep(1) + tabstrip(1) + sep(1) +
// border_top(1) + border_bottom(1) = 6 lines.
func (s playlistScreen) viewportHeight() int {
	if s.height < 10 {
		return 24 // sensible default before the first WindowSizeMsg
	}
	h := s.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

// clampOffset adjusts the viewport offset so the cursor is always visible.
// Call this after any change to cursor, offset, or height.
func (s playlistScreen) clampOffset() playlistScreen {
	vh := s.viewportHeight()
	if s.cursor < s.offset {
		s.offset = s.cursor
	} else if s.cursor >= s.offset+vh {
		s.offset = s.cursor - vh + 1
	}
	if s.offset < 0 {
		s.offset = 0
	}
	return s
}

func (s playlistScreen) Update(msg tea.Msg) (playlistScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Capture and clear pending states before processing the key.
		wasPendingG := s.pendingG
		wasPendingF := s.pendingF
		s.pendingG = false
		s.pendingF = false

		// If f<key> sequence is in progress, consume this key as the jump
		// target and do not pass it to the normal key handler.
		if wasPendingF {
			key := msg.String()
			if len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' ||
				len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
				s = s.jumpToLetter(rune(strings.ToLower(key)[0]))
			}
			break
		}

		switch msg.String() {
		case "j":
			if s.cursor < len(s.entries)-1 {
				s.cursor++
				s = s.clampOffset()
			}
		case "k":
			if s.cursor > 0 {
				s.cursor--
				s = s.clampOffset()
			}
		case "G":
			if len(s.entries) > 0 {
				s.cursor = len(s.entries) - 1
				s = s.clampOffset()
			}
		case "f":
			s.pendingF = true
		case "g":
			if wasPendingG {
				s.cursor = 0 // gg → top
				s = s.clampOffset()
			} else {
				s.pendingG = true
			}
		case "ctrl+d":
			s.cursor += s.viewportHeight() / 2
			if s.cursor >= len(s.entries) {
				s.cursor = len(s.entries) - 1
			}
			if s.cursor < 0 {
				s.cursor = 0
			}
			s = s.clampOffset()
		case "ctrl+u":
			s.cursor -= s.viewportHeight() / 2
			if s.cursor < 0 {
				s.cursor = 0
			}
			s = s.clampOffset()
		case " ":
			if s.cursor < len(s.entries) {
				// Copy the map so mutation doesn't alias the caller's copy.
				sel := make(map[int]bool, len(s.selected))
				for k, v := range s.selected {
					sel[k] = v
				}
				s.selected = sel
				if s.selected[s.cursor] {
					delete(s.selected, s.cursor)
				} else {
					s.selected[s.cursor] = true
				}
			}
		case "x":
			s = s.removeSongs()
		case "X":
			if s.mc != nil {
				_ = s.mc.Clear()
			}
			s.entries = nil
			s.selected = make(map[int]bool)
			s.cursor = 0
		case "enter":
			if s.cursor < len(s.entries) && s.mc != nil {
				_ = s.mc.PlayAt(s.cursor)
			}
		}
	}
	return s, nil
}

// removeSongs deletes selected songs (or the cursor song if none selected)
// from MPD and updates local state optimistically. The server confirms via
// MsgPlaylistChanged.
func (s playlistScreen) removeSongs() playlistScreen {
	if len(s.entries) == 0 {
		return s
	}

	var toRemove []int
	if len(s.selected) > 0 {
		for pos := range s.selected {
			if pos < len(s.entries) {
				toRemove = append(toRemove, pos)
			}
		}
	} else {
		toRemove = []int{s.cursor}
	}

	// Sort descending so each deletion does not shift the positions of
	// remaining targets: MPD positions shift down by one after each delete,
	// so deleting from back to front keeps earlier indices stable.
	sort.Sort(sort.Reverse(sort.IntSlice(toRemove)))

	if s.mc != nil {
		for _, pos := range toRemove {
			_ = s.mc.Delete(pos)
		}
	}

	// Copy the slice before modifying to avoid aliasing the caller's copy
	// (important for test correctness across Bubble Tea model snapshots).
	entries := make([]mpd.PlaylistEntry, len(s.entries))
	copy(entries, s.entries)
	s.entries = entries

	for _, pos := range toRemove {
		s.entries = append(s.entries[:pos], s.entries[pos+1:]...)
	}
	s.selected = make(map[int]bool)

	// Cursor stays at its current index; clamp if it now exceeds the end.
	if len(s.entries) == 0 {
		s.cursor = 0
	} else if s.cursor >= len(s.entries) {
		s.cursor = len(s.entries) - 1
	}

	return s.clampOffset()
}

// jumpToLetter moves the cursor to the next entry (wrapping around) whose
// display name begins with r (already lower-cased). Searches forward from
// cursor+1, wrapping to the top, skipping the cursor itself. No-op if no
// match exists.
func (s playlistScreen) jumpToLetter(r rune) playlistScreen {
	n := len(s.entries)
	for i := 1; i < n; i++ {
		idx := (s.cursor + i) % n
		name := strings.ToLower(entryDisplayName(s.entries[idx]))
		if len(name) > 0 && rune(name[0]) == r {
			s.cursor = idx
			return s.clampOffset()
		}
	}
	return s
}

func (s playlistScreen) View() string {
	if len(s.entries) == 0 {
		return stylePlaceholder.Render("Playlist is empty")
	}

	vh := s.viewportHeight()
	end := s.offset + vh
	if end > len(s.entries) {
		end = len(s.entries)
	}

	var b strings.Builder
	for i := s.offset; i < end; i++ {
		b.WriteString(s.renderRow(i, s.entries[i]))
		b.WriteString("\n")
	}
	return b.String()
}

// renderRow produces one line for a playlist entry.
//
// Prefix layout (5 characters before song text):
//
//	▶ >*  →  cursor + playing + selected
//	  >   →  playing only
//	   *  →  selected only
//	      →  none
func (s playlistScreen) renderRow(i int, entry mpd.PlaylistEntry) string {
	cursor := "  "
	if i == s.cursor {
		cursor = "▶ "
	}

	playing := " "
	if i == s.currentPos {
		playing = ">"
	}

	selected := " "
	if s.selected[i] {
		selected = "*"
	}

	row := cursor + playing + selected + " " + entryDisplayName(entry)

	switch {
	case i == s.cursor:
		return styleRowActive.Render(row)
	case i == s.currentPos:
		return stylePlaylistCurrent.Render(row)
	default:
		return row
	}
}

// entryDisplayName returns a display string for a playlist entry,
// preferring "Title – Artist" metadata over the raw filename.
func entryDisplayName(e mpd.PlaylistEntry) string {
	if e.Title != "" && e.Artist != "" {
		return e.Title + " \u2013 " + e.Artist
	}
	if e.Title != "" {
		return e.Title
	}
	return e.File
}
