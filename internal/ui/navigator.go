package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ghac/internal/mpd"
)

// navClearStatusMsg is sent after a delay to clear the transient status line.
type navClearStatusMsg struct{}

// navConfirmKind identifies the bulk-edit operation awaiting y/n confirmation.
type navConfirmKind int

const (
	navConfirmNone    navConfirmKind = iota
	navConfirmRemove                 // x pressed with > bulkEditThreshold playlist positions
	navConfirmEnqueue                // enter pressed when a directory is in the selection
)

// navigatorScreen is the Song Navigator screen. It browses the MPD music
// library by directory structure and allows enqueuing songs and directories.
type navigatorScreen struct {
	listCursor
	entries     []mpd.DirEntry
	inPlaylist  map[string][]int // MPD URI → playlist positions (supports duplicates)
	currentPath string           // MPD URI of the current directory; "" for root
	width       int              // terminal width for right-aligned metadata
	mc          *mpd.Client      // may be nil in tests; commands become no-ops
	// confirmation prompt state (zero value = no pending confirmation)
	confirmMsg     string         // non-empty = awaiting y/n; shown at bottom of View()
	confirmPending navConfirmKind // action to execute on 'y'
	// transient status message (e.g. "Updating library..."); cleared after a delay
	statusMsg string
}

func newNavigatorScreen(mc *mpd.Client, entries []mpd.DirEntry) navigatorScreen {
	return navigatorScreen{
		listCursor: newListCursor(7),
		entries:    entries,
		inPlaylist: make(map[string][]int),
		mc:         mc,
	}
}

// withPlaylist rebuilds the inPlaylist map from the current queue entries.
// Each URI maps to all playlist positions it occupies (a song may appear
// multiple times). Call this whenever the MPD playlist changes.
func (s navigatorScreen) withPlaylist(entries []mpd.PlaylistEntry) navigatorScreen {
	m := make(map[string][]int, len(entries))
	for _, e := range entries {
		m[e.File] = append(m[e.File], e.Pos)
	}
	s.inPlaylist = m
	return s
}

// withWidth returns a copy with the terminal width updated (used for metadata
// right-alignment in renderRow).
func (s navigatorScreen) withWidth(w int) navigatorScreen {
	s.width = w
	return s
}

// withHeight returns a copy with the terminal height updated and the viewport
// offset re-clamped so the cursor remains visible.
func (s navigatorScreen) withHeight(h int) navigatorScreen {
	s.listCursor = s.listCursor.withHeight(h)
	return s
}

func (s navigatorScreen) Update(msg tea.Msg) (navigatorScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return s.withWidth(msg.Width).withHeight(msg.Height), nil
	case mpd.MsgPlaylistChanged:
		return s.withPlaylist(msg.Entries), nil
	case navClearStatusMsg:
		s.statusMsg = ""
		return s, nil
	case tea.KeyMsg:
		// While a confirmation is pending capturesAllInput() returns true, so
		// global handlers have already been bypassed. Only y/n/esc are
		// meaningful; all other keys are dropped.
		if s.confirmPending != navConfirmNone {
			switch msg.String() {
			case "y":
				pending := s.confirmPending
				s.confirmMsg = ""
				s.confirmPending = navConfirmNone
				switch pending {
				case navConfirmRemove:
					s = s.removeFromPlaylist()
				case navConfirmEnqueue:
					s = s.enqueue()
				}
			case "n", "esc":
				s.confirmMsg = ""
				s.confirmPending = navConfirmNone
			}
			break
		}

		wasPendingG, wasPendingF, lc := s.capturePending()
		s.listCursor = lc

		// If f<key> sequence is in progress, consume this key as the jump
		// target and do not pass it to the normal key handler.
		if wasPendingF {
			key := msg.String()
			if len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' ||
				len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
				s.listCursor = s.jumpToLetter(
					rune(strings.ToLower(key)[0]),
					func(i int) string { return s.entries[i].Name },
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
		case "l":
			if s.cursor < len(s.entries) && s.entries[s.cursor].IsDir {
				s = s.enterDir(s.entries[s.cursor].Path)
			}
		case "h":
			s = s.goUp()
		case " ":
			s.listCursor = s.toggleSelected(s.cursor, len(s.entries))
		case "x":
			positions := s.collectRemovePositions()
			if len(positions) > bulkEditThreshold {
				s.confirmMsg = fmt.Sprintf("Remove %d songs from playlist? [y/n]", len(positions))
				s.confirmPending = navConfirmRemove
			} else {
				s = s.removeFromPlaylist()
			}
		case "enter":
			if msg, needs := s.enqueueConfirmMsg(); needs {
				s.confirmMsg = msg
				s.confirmPending = navConfirmEnqueue
			} else {
				s = s.enqueue()
			}
		case "U":
			if s.mc != nil {
				_ = s.mc.UpdateLibrary(s.currentPath)
			}
			s.statusMsg = "Updating library..."
			return s, func() tea.Msg {
				time.Sleep(3 * time.Second)
				return navClearStatusMsg{}
			}
		}
	}
	return s, nil
}

// collectRemovePositions returns all playlist positions that would be deleted
// when x is pressed: positions for selected queued files, or for the cursor
// file if nothing is selected. Directories and unqueued files yield no
// positions.
func (s navigatorScreen) collectRemovePositions() []int {
	if len(s.entries) == 0 {
		return nil
	}
	var uris []string
	if len(s.selected) > 0 {
		for i := range s.selected {
			if i < len(s.entries) && !s.entries[i].IsDir {
				uris = append(uris, s.entries[i].Path)
			}
		}
	} else if s.cursor < len(s.entries) && !s.entries[s.cursor].IsDir {
		uris = append(uris, s.entries[s.cursor].Path)
	}
	var positions []int
	for _, uri := range uris {
		positions = append(positions, s.inPlaylist[uri]...)
	}
	return positions
}

// enqueueConfirmMsg returns (message, true) when the pending enqueue operation
// requires confirmation: any directory in the selection always requires
// confirmation; file-only selections require it when the count exceeds
// bulkEditThreshold. Returns ("", false) when no confirmation is needed.
func (s navigatorScreen) enqueueConfirmMsg() (string, bool) {
	if len(s.entries) == 0 {
		return "", false
	}
	var targets []int
	if len(s.selected) > 0 {
		for i := range s.selected {
			if i < len(s.entries) {
				targets = append(targets, i)
			}
		}
	} else if s.cursor < len(s.entries) {
		targets = []int{s.cursor}
	}
	if len(targets) == 0 {
		return "", false
	}

	var dirs, files int
	for _, i := range targets {
		if s.entries[i].IsDir {
			dirs++
		} else {
			files++
		}
	}

	if dirs > 0 {
		if files == 0 {
			noun := "directories"
			if dirs == 1 {
				noun = "directory"
			}
			return fmt.Sprintf("Add %d %s to playlist? [y/n]", dirs, noun), true
		}
		total := dirs + files
		return fmt.Sprintf("Add %d entries to playlist? [y/n]", total), true
	}

	if files > bulkEditThreshold {
		return fmt.Sprintf("Add %d songs to playlist? [y/n]", files), true
	}

	return "", false
}

// removeFromPlaylist deletes from the MPD playlist every queued file that is
// currently selected (or the cursor entry if nothing is selected). Directories
// and files not in the playlist are silently skipped. Positions are deleted in
// descending order so earlier positions are not shifted by later removals.
// Clears the selection after deleting.
func (s navigatorScreen) removeFromPlaylist() navigatorScreen {
	positions := s.collectRemovePositions()
	if len(positions) == 0 {
		return s
	}

	// Delete highest positions first so lower positions stay valid.
	sort.Sort(sort.Reverse(sort.IntSlice(positions)))
	if s.mc != nil {
		for _, pos := range positions {
			_ = s.mc.Delete(pos)
		}
	}

	s.listCursor = s.clearSelection()
	return s
}

// enterDir navigates into the directory at path, resetting cursor and selection.
func (s navigatorScreen) enterDir(path string) navigatorScreen {
	s.entries = s.fetchEntries(path)
	s.currentPath = path
	s.cursor = 0
	s.offset = 0
	s.listCursor = s.clearSelection()
	return s
}

// goUp navigates to the parent directory. No-op at root ("").
// The cursor is placed on the subdirectory we just left so the user can
// see where they came from.
func (s navigatorScreen) goUp() navigatorScreen {
	if s.currentPath == "" {
		return s
	}
	leaving := navBase(s.currentPath)
	parent := navParent(s.currentPath)
	s.entries = s.fetchEntries(parent)
	s.currentPath = parent
	s.listCursor = s.clearSelection()

	// Find the directory we just came from and restore the cursor to it.
	s.cursor = 0
	for i, e := range s.entries {
		if e.IsDir && e.Name == leaving {
			s.cursor = i
			break
		}
	}

	s.offset = 0
	s.listCursor = s.clampOffset()
	return s
}

// enqueue adds selected entries (or the cursor entry when nothing is selected)
// to the MPD playlist, then clears the selection. Directories are enqueued
// recursively by MPD.
func (s navigatorScreen) enqueue() navigatorScreen {
	if len(s.entries) == 0 {
		return s
	}

	var toAdd []string
	if len(s.selected) > 0 {
		for i := range s.selected {
			if i < len(s.entries) {
				toAdd = append(toAdd, s.entries[i].Path)
			}
		}
	} else if s.cursor < len(s.entries) {
		toAdd = []string{s.entries[s.cursor].Path}
	}

	if s.mc != nil {
		for _, uri := range toAdd {
			_ = s.mc.Add(uri)
		}
	}

	s.listCursor = s.clearSelection()
	return s
}

// fetchEntries calls ListInfo on the client and returns the result.
// Returns nil when the client is nil or an error occurs.
func (s navigatorScreen) fetchEntries(path string) []mpd.DirEntry {
	if s.mc == nil {
		return nil
	}
	entries, err := s.mc.ListInfo(path)
	if err != nil {
		return nil
	}
	return entries
}

func (s navigatorScreen) View() string {
	var b strings.Builder

	// Breadcrumb showing the current directory path.
	crumb := "/ (root)"
	if s.currentPath != "" {
		crumb = s.currentPath
	}
	b.WriteString(stylePlaceholder.Render("Path: " + crumb))
	b.WriteString("\n")

	if len(s.entries) == 0 {
		b.WriteString(stylePlaceholder.Render("Directory is empty"))
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
	} else if s.statusMsg != "" {
		b.WriteString(stylePlaceholder.Render(s.statusMsg))
		b.WriteString("\n")
	}
	return b.String()
}

// renderRow produces one line for a directory entry.
//
// Prefix layout (5 characters before the name):
//
//	▶ *+  →  cursor + selected + in-playlist
//	  *   →  selected only
//	   +  →  in-playlist only (file already queued)
//	      →  none
//
// Layout for directories:
//
//	▶ *  DirectoryName/
//
// Layout for files (metadata right-aligned when terminal width is known):
//
//	▶ *+ filename.flac          Title – Artist
func (s navigatorScreen) renderRow(i int, entry mpd.DirEntry) string {
	cursor := "  "
	if i == s.cursor {
		cursor = symCursor
	}

	sel := " "
	if s.selected[i] {
		sel = "*"
	}

	queued := " "
	if !entry.IsDir && len(s.inPlaylist[entry.Path]) > 0 {
		queued = "+"
	}

	prefix := cursor + sel + queued + " " // always 5 visible characters

	var row string
	if entry.IsDir {
		row = prefix + styleNavDir.Render(entry.Name+"/")
	} else {
		meta := navMeta(entry)
		if meta == "" {
			row = prefix + entry.Name
		} else {
			w := s.width
			if w < 40 {
				w = 80 // default before WindowSizeMsg arrives
			}
			// innerWidth = terminal width minus border (4) minus prefix (5).
			available := w - 9
			nameW := lipgloss.Width(entry.Name)
			metaW := lipgloss.Width(meta)
			gap := available - nameW - metaW
			if gap >= 2 {
				row = prefix + entry.Name + strings.Repeat(" ", gap) + styleNavMeta.Render(meta)
			} else {
				// Not enough room to right-align; show name only.
				row = prefix + entry.Name
			}
		}
	}

	if i == s.cursor {
		return styleRowActive.Render(row)
	}
	return row
}

// navMeta returns the display metadata string for a file entry.
// Returns "" when no useful metadata is present.
func navMeta(entry mpd.DirEntry) string {
	if entry.Song.Title != "" && entry.Song.Artist != "" {
		return entry.Song.Title + symSeparator + entry.Song.Artist
	}
	if entry.Song.Title != "" {
		return entry.Song.Title
	}
	return ""
}

// navParent returns the parent directory of an MPD URI.
// Returns "" (root) for top-level paths that contain no slash.
func navParent(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
}

// navBase returns the final path segment of an MPD URI.
func navBase(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// screen interface implementation.

func (s navigatorScreen) update(msg tea.Msg) (screen, tea.Cmd) { return s.Update(msg) }
func (s navigatorScreen) hasPendingF() bool                    { return s.pendingF }
func (s navigatorScreen) capturesAllInput() bool               { return s.confirmPending != navConfirmNone }
func (s navigatorScreen) activeModal() (title, content string, minWidth, maxWidth int, ok bool) {
	return
}
func (s navigatorScreen) tabTitle() string    { return "3:Library" }
func (s navigatorScreen) screenTitle() string { return "Library Navigator" }
