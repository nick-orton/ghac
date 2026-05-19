package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
)

// bulkEditThreshold is the minimum number of songs that must be affected for a
// playlist edit operation to require confirmation. Operations on more songs
// than this threshold prompt the user with y/n before executing.
const bulkEditThreshold = 50

// playlistConfirmKind identifies the bulk-edit operation awaiting y/n confirmation.
type playlistConfirmKind int

const (
	playlistConfirmNone   playlistConfirmKind = iota
	playlistConfirmRemove                     // x pressed with > bulkEditThreshold songs
	playlistConfirmClear                      // X always requires confirmation
)

// playlistScreen is the Playlist Control screen. It displays the MPD
// playback queue and supports cursor navigation, song selection, removal,
// clear, and playback jump.
type playlistScreen struct {
	listCursor
	entries    []mpd.PlaylistEntry
	currentPos int         // playlist position of currently-playing song; -1 if none
	mc         *mpd.Client // may be nil in tests; commands become no-ops
	// confirmation prompt state (zero value = no pending confirmation)
	confirmMsg     string              // non-empty = awaiting y/n; shown at bottom of View()
	confirmPending playlistConfirmKind // action to execute on 'y'
}

func newPlaylistScreen(mc *mpd.Client, entries []mpd.PlaylistEntry, currentPos int) playlistScreen {
	return playlistScreen{
		listCursor: newListCursor(6),
		entries:    entries,
		currentPos: currentPos,
		mc:         mc,
	}
}

// withEntries returns a copy with the playlist replaced, selection cleared,
// cursor clamped, and current playing position updated atomically.
// Any pending confirmation is dismissed because the song count is now stale.
func (s playlistScreen) withEntries(entries []mpd.PlaylistEntry, currentPos int) playlistScreen {
	s.entries = entries
	s.currentPos = currentPos
	s.selected = make(map[int]bool)
	s.confirmMsg = ""
	s.confirmPending = playlistConfirmNone
	if len(entries) == 0 {
		s.cursor = 0
	} else if s.cursor >= len(entries) {
		s.cursor = len(entries) - 1
	}
	s.listCursor = s.clampOffset()
	return s
}

// withCurrentPos returns a copy with the current playing position updated.
func (s playlistScreen) withCurrentPos(pos int) playlistScreen {
	s.currentPos = pos
	return s
}

// withHeight returns a copy with the terminal height updated and the viewport
// offset re-clamped so the cursor remains visible.
func (s playlistScreen) withHeight(h int) playlistScreen {
	s.listCursor = s.listCursor.withHeight(h)
	return s
}

func (s playlistScreen) Update(msg tea.Msg) (playlistScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return s.withHeight(msg.Height), nil
	case mpd.MsgPlayerState:
		return s.withCurrentPos(msg.SongPos), nil
	case mpd.MsgPlaylistChanged:
		return s.withEntries(msg.Entries, s.currentPos), nil
	case tea.KeyMsg:
		// While a confirmation is pending capturesAllInput() returns true, so
		// global handlers (screen switch, play/pause, etc.) have already been
		// bypassed. Only y/n/esc are meaningful here; all other keys are dropped.
		if s.confirmPending != playlistConfirmNone {
			switch msg.String() {
			case "y":
				pending := s.confirmPending
				s.confirmMsg = ""
				s.confirmPending = playlistConfirmNone
				switch pending {
				case playlistConfirmRemove:
					s = s.removeSongs()
				case playlistConfirmClear:
					if s.mc != nil {
						_ = s.mc.Clear()
					}
					s.entries = nil
					s.selected = make(map[int]bool)
					s.cursor = 0
				}
			case "n", "esc":
				s.confirmMsg = ""
				s.confirmPending = playlistConfirmNone
			}
			break
		}

		wasPendingG, wasPendingF, lc := s.capturePending()
		s.listCursor = lc

		// If f<key> is in progress consume this key as the jump target.
		if wasPendingF {
			key := msg.String()
			if len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' ||
				len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
				s.listCursor = s.jumpToLetter(
					rune(strings.ToLower(key)[0]),
					func(i int) string { return entryDisplayName(s.entries[i]) },
					len(s.entries),
				)
			}
			break
		}

		switch msg.String() {
		case "j":
			s.listCursor = s.moveDown(len(s.entries))
		case "k":
			s.listCursor = s.moveUp()
		case "G":
			s.listCursor = s.moveToEnd(len(s.entries))
		case "f":
			s.pendingF = true
		case "g":
			if wasPendingG {
				s.listCursor = s.moveToTop()
			} else {
				s.pendingG = true
			}
		case "ctrl+d":
			s.listCursor = s.halfPageDown(len(s.entries))
		case "ctrl+u":
			s.listCursor = s.halfPageUp()
		case " ":
			s.listCursor = s.toggleSelected(s.cursor, len(s.entries))
		case "ctrl+j":
			s = s.moveSong(1)
		case "ctrl+k":
			s = s.moveSong(-1)
		case "x":
			toRemove := s.collectRemovePositions()
			if len(toRemove) > bulkEditThreshold {
				s.confirmMsg = fmt.Sprintf("Remove %d songs? [y/n]", len(toRemove))
				s.confirmPending = playlistConfirmRemove
			} else {
				s = s.removeSongs()
			}
		case "X":
			if len(s.entries) == 0 {
				break
			}
			s.confirmMsg = fmt.Sprintf("Clear all %d songs? [y/n]", len(s.entries))
			s.confirmPending = playlistConfirmClear
		case "enter":
			if s.cursor < len(s.entries) && s.mc != nil {
				_ = s.mc.PlayAt(s.cursor)
			}
		}
	}
	return s, nil
}

// collectRemovePositions returns the playlist positions that would be deleted
// when x is pressed: all selected positions (if any selection exists), or just
// the cursor position. Returns nil when the playlist is empty.
func (s playlistScreen) collectRemovePositions() []int {
	if len(s.entries) == 0 {
		return nil
	}
	if len(s.selected) > 0 {
		toRemove := make([]int, 0, len(s.selected))
		for pos := range s.selected {
			if pos < len(s.entries) {
				toRemove = append(toRemove, pos)
			}
		}
		return toRemove
	}
	return []int{s.cursor}
}

// removeSongs deletes selected songs (or the cursor song if none selected)
// from MPD and updates local state optimistically. The server confirms via
// MsgPlaylistChanged.
func (s playlistScreen) removeSongs() playlistScreen {
	toRemove := s.collectRemovePositions()
	if len(toRemove) == 0 {
		return s
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
	s.listCursor = s.clampOffset()
	return s
}

// moveSong moves the song under the cursor by delta positions (+1 = down,
// -1 = up). The cursor follows the moved song. No-op at list boundaries.
func (s playlistScreen) moveSong(delta int) playlistScreen {
	if len(s.entries) == 0 {
		return s
	}
	target := s.cursor + delta
	if target < 0 || target >= len(s.entries) {
		return s
	}

	// Copy slice to avoid aliasing.
	entries := make([]mpd.PlaylistEntry, len(s.entries))
	copy(entries, s.entries)
	entries[s.cursor], entries[target] = entries[target], entries[s.cursor]
	s.entries = entries

	// Keep currentPos consistent with the swap.
	if s.currentPos == s.cursor {
		s.currentPos = target
	} else if s.currentPos == target {
		s.currentPos = s.cursor
	}

	if s.mc != nil {
		_ = s.mc.Move(s.cursor, target)
	}
	s.cursor = target
	return s
}

func (s playlistScreen) View() string {
	var b strings.Builder

	if len(s.entries) == 0 {
		b.WriteString(stylePlaceholder.Render("Playlist is empty"))
	} else {
		vh := s.viewportHeight()
		end := s.offset + vh
		if end > len(s.entries) {
			end = len(s.entries)
		}
		for i := s.offset; i < end; i++ {
			b.WriteString(s.renderRow(i, s.entries[i]))
			b.WriteString("\n")
		}
	}

	if s.confirmMsg != "" {
		b.WriteString(styleRowActive.Render(s.confirmMsg))
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

// screen interface implementation.

func (s playlistScreen) update(msg tea.Msg) (screen, tea.Cmd) { return s.Update(msg) }
func (s playlistScreen) hasPendingF() bool                    { return s.pendingF }
func (s playlistScreen) capturesAllInput() bool               { return s.confirmPending != playlistConfirmNone }
func (s playlistScreen) activeModal() (title, content string, minWidth, maxWidth int, ok bool) {
	return
}
func (s playlistScreen) tabTitle() string    { return "2:Playlist" }
func (s playlistScreen) screenTitle() string { return "Playlist Control" }
