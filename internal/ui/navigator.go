package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ghac/internal/mpd"
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
	case tea.KeyMsg:
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
			s = s.removeFromPlaylist()
		case "enter":
			s = s.enqueue()
		}
	}
	return s, nil
}

// removeFromPlaylist deletes from the MPD playlist every queued file that is
// currently selected (or the cursor entry if nothing is selected). Directories
// and files not in the playlist are silently skipped. Positions are deleted in
// descending order so earlier positions are not shifted by later removals.
// Clears the selection after deleting.
func (s navigatorScreen) removeFromPlaylist() navigatorScreen {
	if len(s.entries) == 0 {
		return s
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

	// Collect all playlist positions for the target URIs.
	var positions []int
	for _, uri := range uris {
		positions = append(positions, s.inPlaylist[uri]...)
	}
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
		return b.String()
	}

	vh := s.viewportHeight()
	end := s.offset + vh
	if end > len(s.entries) {
		end = len(s.entries)
	}

	for i := s.offset; i < end; i++ {
		b.WriteString(s.renderRow(i, s.entries[i]))
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
		cursor = "▶ "
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
		return entry.Song.Title + " \u2013 " + entry.Song.Artist
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

func (s navigatorScreen) update(msg tea.Msg) (screen, tea.Cmd)                              { return s.Update(msg) }
func (s navigatorScreen) hasPendingF() bool                                                  { return s.pendingF }
func (s navigatorScreen) capturesAllInput() bool                                             { return false }
func (s navigatorScreen) activeModal() (title, content string, minWidth, maxWidth int, ok bool) { return }
func (s navigatorScreen) tabTitle() string                                                   { return "3:Library" }
func (s navigatorScreen) screenTitle() string                                                { return "Library Navigator" }
