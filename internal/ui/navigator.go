package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ghac/internal/mpd"
)

// navigatorScreen is the Song Navigator screen. It browses the MPD music
// library by directory structure and allows enqueuing songs and directories.
type navigatorScreen struct {
	entries     []mpd.DirEntry
	cursor      int
	offset      int        // index of the first visible entry (viewport top)
	selected    map[int]bool
	currentPath string     // MPD URI of the current directory; "" for root
	width       int        // terminal width for right-aligned metadata
	height      int        // terminal height for viewport sizing
	mc          *mpd.Client // may be nil in tests; commands become no-ops
}

func newNavigatorScreen(mc *mpd.Client, entries []mpd.DirEntry) navigatorScreen {
	return navigatorScreen{
		entries:  entries,
		cursor:   0,
		offset:   0,
		selected: make(map[int]bool),
		mc:       mc,
	}
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
	s.height = h
	return s.clampOffset()
}

// viewportHeight returns the number of entry rows that fit on screen.
// The overhead is: nowplaying(1) + sep(1) + tabstrip(1) + sep(1) +
// border_top(1) + breadcrumb(1) + border_bottom(1) = 7 lines.
func (s navigatorScreen) viewportHeight() int {
	if s.height < 10 {
		return 24 // sensible default before the first WindowSizeMsg
	}
	h := s.height - 7
	if h < 1 {
		h = 1
	}
	return h
}

// clampOffset adjusts the viewport offset so the cursor is always visible.
// Call this after any change to cursor, offset, or height.
func (s navigatorScreen) clampOffset() navigatorScreen {
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

func (s navigatorScreen) Update(msg tea.Msg) (navigatorScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case "l":
			if s.cursor < len(s.entries) && s.entries[s.cursor].IsDir {
				s = s.enterDir(s.entries[s.cursor].Path)
			}
		case "h":
			s = s.goUp()
		case " ":
			if s.cursor < len(s.entries) {
				// Copy the map so mutation does not alias the caller's copy.
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
		case "enter":
			s = s.enqueue()
		}
	}
	return s, nil
}

// enterDir navigates into the directory at path, resetting cursor and selection.
func (s navigatorScreen) enterDir(path string) navigatorScreen {
	s.entries = s.fetchEntries(path)
	s.currentPath = path
	s.cursor = 0
	s.offset = 0
	s.selected = make(map[int]bool)
	return s
}

// goUp navigates to the parent directory. No-op at root ("").
func (s navigatorScreen) goUp() navigatorScreen {
	if s.currentPath == "" {
		return s
	}
	parent := navParent(s.currentPath)
	s.entries = s.fetchEntries(parent)
	s.currentPath = parent
	s.cursor = 0
	s.offset = 0
	s.selected = make(map[int]bool)
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

	s.selected = make(map[int]bool)
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
// Layout for directories:
//
//	▶ * DirectoryName/
//
// Layout for files (metadata right-aligned when terminal width is known):
//
//	▶ * filename.flac          Title – Artist
func (s navigatorScreen) renderRow(i int, entry mpd.DirEntry) string {
	cursor := "  "
	if i == s.cursor {
		cursor = "▶ "
	}

	sel := " "
	if s.selected[i] {
		sel = "*"
	}

	prefix := cursor + sel + " " // always 4 visible characters

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
			// innerWidth = terminal width minus border (4) minus prefix (4).
			available := w - 8
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
