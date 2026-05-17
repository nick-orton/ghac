package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
)

// testDirEntries returns a repeatable directory listing for use in tests.
// The listing contains two directories and two files (one with metadata,
// one without).
func testDirEntries() []mpd.DirEntry {
	return []mpd.DirEntry{
		{Name: "rock", Path: "rock", IsDir: true},
		{Name: "jazz", Path: "jazz", IsDir: true},
		{
			Name:  "song.flac",
			Path:  "song.flac",
			IsDir: false,
			Song:  mpd.Song{Title: "My Song", Artist: "My Artist", File: "song.flac"},
		},
		{
			Name:  "notagged.mp3",
			Path:  "notagged.mp3",
			IsDir: false,
			Song:  mpd.Song{File: "notagged.mp3"},
		},
	}
}

func newTestNavigatorScreen() navigatorScreen {
	return newNavigatorScreen(nil, testDirEntries())
}

func pressNavKey(s navigatorScreen, key string) navigatorScreen {
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

// --- Cursor movement ---

func TestNavCursorMoveDown(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "j")
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after j", s.cursor)
	}
}

func TestNavCursorMoveUp(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "j")
	s = pressNavKey(s, "k")
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after j then k", s.cursor)
	}
}

func TestNavCursorDoesNotGoAboveZero(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "k")
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after k at top", s.cursor)
	}
}

func TestNavCursorDoesNotGoBelowBottom(t *testing.T) {
	s := newTestNavigatorScreen() // 4 entries
	for i := 0; i < 10; i++ {
		s = pressNavKey(s, "j")
	}
	if s.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (last entry) after pressing j past bottom", s.cursor)
	}
}

// --- Directory enter/exit ---

func TestNavLEntersDirectory(t *testing.T) {
	// Supply a custom screen where the cursor is on a directory and the
	// client is nil (enterDir will call fetchEntries → nil with nil client).
	s := newTestNavigatorScreen() // cursor=0, which is "rock" dir
	s = pressNavKey(s, "l")
	if s.currentPath != "rock" {
		t.Errorf("currentPath = %q, want %q after l on directory", s.currentPath, "rock")
	}
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after entering directory", s.cursor)
	}
}

func TestNavLOnFileIsNoop(t *testing.T) {
	s := newTestNavigatorScreen()
	// Move cursor to a file entry (index 2).
	s = pressNavKey(s, "j")
	s = pressNavKey(s, "j") // cursor=2, song.flac
	s = pressNavKey(s, "l")
	if s.currentPath != "" {
		t.Errorf("currentPath = %q after l on file, want empty (no change)", s.currentPath)
	}
	if s.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (unchanged) after l on file", s.cursor)
	}
}

func TestNavHGoesUp(t *testing.T) {
	// Start in a subdirectory.
	s := navigatorScreen{
		entries:     testDirEntries(),
		cursor:      1,
		selected:    make(map[int]bool),
		currentPath: "rock/classic",
	}
	s = pressNavKey(s, "h")
	if s.currentPath != "rock" {
		t.Errorf("currentPath = %q, want %q after h", s.currentPath, "rock")
	}
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after h", s.cursor)
	}
}

func TestNavHAtRootIsNoop(t *testing.T) {
	s := newTestNavigatorScreen() // currentPath=""
	s = pressNavKey(s, "h")
	if s.currentPath != "" {
		t.Errorf("currentPath = %q after h at root, want empty", s.currentPath)
	}
}

func TestNavLEnteringDirResetsCursor(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "j") // cursor=1 before entering
	s = pressNavKey(s, "l") // enter "jazz"
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after entering directory", s.cursor)
	}
}

func TestNavLEnteringDirClearsSelection(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, " ") // select entry 0 (rock/)
	s = pressNavKey(s, "l") // enter "rock"
	if len(s.selected) != 0 {
		t.Errorf("selection should be cleared after entering directory, len=%d", len(s.selected))
	}
}

// --- Root boundary ---

func TestNavParentOfTopLevel(t *testing.T) {
	got := navParent("rock")
	if got != "" {
		t.Errorf("navParent(%q) = %q, want %q", "rock", got, "")
	}
}

func TestNavParentOfNestedPath(t *testing.T) {
	got := navParent("rock/classic")
	if got != "rock" {
		t.Errorf("navParent(%q) = %q, want %q", "rock/classic", got, "rock")
	}
}

func TestNavParentOfRoot(t *testing.T) {
	got := navParent("")
	if got != "" {
		t.Errorf("navParent(%q) = %q, want %q", "", got, "")
	}
}

// --- Selection toggle ---

func TestNavToggleSelection(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, " ")
	if !s.selected[0] {
		t.Error("entry 0 should be selected after space")
	}
	s = pressNavKey(s, " ")
	if s.selected[0] {
		t.Error("entry 0 should be deselected after second space")
	}
}

func TestNavSelectionPersistsOnCursorMove(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, " ") // select entry 0
	s = pressNavKey(s, "j") // move cursor to 1
	if !s.selected[0] {
		t.Error("selection on entry 0 should persist after cursor moves")
	}
	if s.selected[1] {
		t.Error("entry 1 should not be selected after cursor moves there")
	}
}

// --- Enqueue ---

func TestNavEnqueueCursorWithNoClient(t *testing.T) {
	// nil client: enqueue should clear selection without panicking.
	s := newTestNavigatorScreen()
	s = pressNavKey(s, " ")           // select entry 0
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Selection should be cleared after enqueue.
	if len(s.selected) != 0 {
		t.Errorf("selection len = %d, want 0 after enter with nil client", len(s.selected))
	}
}

func TestNavEnqueueSelectedClearsSelection(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, " ") // select entry 0
	s = pressNavKey(s, "j")
	s = pressNavKey(s, " ") // select entry 1
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(s.selected) != 0 {
		t.Error("selection should be cleared after enter")
	}
}

func TestNavEnqueueCursorWhenNothingSelected(t *testing.T) {
	// Verify enqueue uses cursor when selected is empty (no panic, no client).
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "j") // cursor=1
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(s.selected) != 0 {
		t.Error("selection should be empty after enter")
	}
}

// --- withPlaylist / in-queue marker ---

func TestNavWithPlaylistMarksQueuedFile(t *testing.T) {
	s := newTestNavigatorScreen()
	playlist := []mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 0},
	}
	s = s.withPlaylist(playlist)
	if !s.inPlaylist["song.flac"] {
		t.Error("inPlaylist should contain 'song.flac' after withPlaylist")
	}
	if s.inPlaylist["notagged.mp3"] {
		t.Error("inPlaylist should not contain 'notagged.mp3' when not in playlist")
	}
}

func TestNavViewShowsQueueMarkerForQueuedFile(t *testing.T) {
	s := newTestNavigatorScreen()
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 0},
	})
	view := s.View()
	if !strings.Contains(view, "+") {
		t.Error("view should contain '+' marker for a file in the playlist")
	}
}

func TestNavViewNoQueueMarkerWhenNotInPlaylist(t *testing.T) {
	s := newTestNavigatorScreen() // inPlaylist is empty
	view := s.View()
	if strings.Contains(view, "+") {
		t.Error("view should not contain '+' when no files are in the playlist")
	}
}

func TestNavWithPlaylistDoesNotMarkDirectories(t *testing.T) {
	s := newTestNavigatorScreen()
	// Even if a directory path matches a playlist entry File, it should not show +.
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "rock"}, Pos: 0}, // "rock" is a dir in testDirEntries
	})
	// Render the row for the "rock" directory (index 0).
	row := s.renderRow(0, s.entries[0])
	if strings.Contains(row, "+") {
		t.Error("directory rows should never show the '+' queue marker")
	}
}

func TestNavWithPlaylistReplacesOnUpdate(t *testing.T) {
	s := newTestNavigatorScreen()
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 0},
	})
	// Now update with an empty playlist.
	s = s.withPlaylist(nil)
	if len(s.inPlaylist) != 0 {
		t.Errorf("inPlaylist len = %d after empty update, want 0", len(s.inPlaylist))
	}
}

// --- View rendering ---

func TestNavViewShowsBreadcrumbAtRoot(t *testing.T) {
	s := newTestNavigatorScreen()
	view := s.View()
	if !strings.Contains(view, "/ (root)") {
		t.Errorf("view should contain '/ (root)' breadcrumb at root, got: %q", view)
	}
}

func TestNavViewShowsCurrentPath(t *testing.T) {
	s := navigatorScreen{
		entries:     testDirEntries(),
		cursor:      0,
		selected:    make(map[int]bool),
		currentPath: "rock/classic",
	}
	view := s.View()
	if !strings.Contains(view, "rock/classic") {
		t.Errorf("view should contain current path 'rock/classic', got: %q", view)
	}
}

func TestNavViewShowsDirectoriesWithSlash(t *testing.T) {
	s := newTestNavigatorScreen()
	view := s.View()
	if !strings.Contains(view, "rock/") {
		t.Errorf("view should contain 'rock/' for directory entry, got: %q", view)
	}
}

func TestNavViewShowsFileMetadata(t *testing.T) {
	s := newTestNavigatorScreen()
	view := s.View()
	if !strings.Contains(view, "My Song") {
		t.Error("view should contain song title 'My Song'")
	}
	if !strings.Contains(view, "My Artist") {
		t.Error("view should contain artist 'My Artist'")
	}
}

func TestNavViewEmptyDirectory(t *testing.T) {
	s := newNavigatorScreen(nil, nil)
	view := s.View()
	if !strings.Contains(view, "Directory is empty") {
		t.Errorf("empty directory view should contain 'Directory is empty', got: %q", view)
	}
}

func TestNavViewShowsCursorMarker(t *testing.T) {
	s := newTestNavigatorScreen()
	view := s.View()
	if !strings.Contains(view, "▶") {
		t.Error("view should contain '▶' cursor marker")
	}
}

func TestNavViewShowsSelectionMarker(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, " ") // select entry 0
	view := s.View()
	if !strings.Contains(view, "*") {
		t.Error("view should contain '*' selection marker")
	}
}

// --- navMeta ---

func TestNavMetaTitleAndArtist(t *testing.T) {
	e := mpd.DirEntry{Song: mpd.Song{Title: "T", Artist: "A"}}
	got := navMeta(e)
	want := "T \u2013 A"
	if got != want {
		t.Errorf("navMeta = %q, want %q", got, want)
	}
}

func TestNavMetaTitleOnly(t *testing.T) {
	e := mpd.DirEntry{Song: mpd.Song{Title: "Only Title"}}
	got := navMeta(e)
	if got != "Only Title" {
		t.Errorf("navMeta = %q, want %q", got, "Only Title")
	}
}

func TestNavMetaNoMetadata(t *testing.T) {
	e := mpd.DirEntry{Song: mpd.Song{File: "raw.flac"}}
	got := navMeta(e)
	if got != "" {
		t.Errorf("navMeta = %q, want empty string when no title", got)
	}
}

func TestNavMetaDirectory(t *testing.T) {
	e := mpd.DirEntry{IsDir: true}
	got := navMeta(e)
	if got != "" {
		t.Errorf("navMeta = %q, want empty for directory", got)
	}
}

// --- Viewport / scrolling ---

// makeEntries returns n file entries for viewport tests.
func makeEntries(n int) []mpd.DirEntry {
	entries := make([]mpd.DirEntry, n)
	for i := range entries {
		entries[i] = mpd.DirEntry{
			Name: fmt.Sprintf("track%02d.flac", i),
			Path: fmt.Sprintf("track%02d.flac", i),
		}
	}
	return entries
}

func TestNavViewportHeightDefault(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50))
	// height == 0 → default viewport
	vh := s.viewportHeight()
	if vh < 1 {
		t.Errorf("viewportHeight() = %d, want >= 1 when height unset", vh)
	}
}

func TestNavViewportHeightUsesTerminalHeight(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(30)
	// overhead is 7: viewportHeight should be 30 - 7 = 23
	got := s.viewportHeight()
	if got != 23 {
		t.Errorf("viewportHeight() = %d, want 23 for terminal height 30", got)
	}
}

func TestNavClampOffsetScrollsDownWithCursor(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh = 10
	// Move cursor past the viewport bottom.
	for i := 0; i < 15; i++ {
		s = pressNavKey(s, "j")
	}
	// cursor = 15; viewport = 10 → offset must be at least cursor - vh + 1 = 6
	if s.cursor != 15 {
		t.Fatalf("cursor = %d, want 15", s.cursor)
	}
	if s.offset > s.cursor {
		t.Errorf("offset %d > cursor %d: cursor not visible", s.offset, s.cursor)
	}
	vh := s.viewportHeight()
	if s.cursor >= s.offset+vh {
		t.Errorf("cursor %d >= offset %d + vh %d: cursor below viewport", s.cursor, s.offset, vh)
	}
}

func TestNavClampOffsetScrollsUpWithCursor(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh = 10
	// Scroll down.
	for i := 0; i < 20; i++ {
		s = pressNavKey(s, "j")
	}
	// Now scroll back up.
	for i := 0; i < 20; i++ {
		s = pressNavKey(s, "k")
	}
	if s.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 after scrolling back to top", s.cursor)
	}
	if s.offset != 0 {
		t.Errorf("offset = %d, want 0 after scrolling cursor back to top", s.offset)
	}
}

func TestNavViewRendersOnlyViewportRows(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh = 10
	view := s.View()
	// Count entry lines: breadcrumb is always first, then entries.
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	// lines[0] = breadcrumb; lines[1..] = entries (up to viewportHeight)
	entryLines := len(lines) - 1
	if entryLines > 10 {
		t.Errorf("view has %d entry lines, want at most 10 for viewportHeight=10", entryLines)
	}
}

func TestNavViewShowsCursorEntryWhenScrolled(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh = 10
	// Move cursor to entry 20.
	for i := 0; i < 20; i++ {
		s = pressNavKey(s, "j")
	}
	view := s.View()
	if !strings.Contains(view, "track20.flac") {
		t.Error("view should contain cursor entry 'track20.flac' after scrolling")
	}
	// Entry 0 should no longer be visible.
	if strings.Contains(view, "track00.flac") {
		t.Error("view should not contain 'track00.flac' after scrolling past it")
	}
}

func TestNavEnterDirResetsOffset(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17)
	for i := 0; i < 20; i++ {
		s = pressNavKey(s, "j")
	}
	// Simulate entering a directory (with no client entries become nil).
	s = s.enterDir("subdir")
	if s.offset != 0 {
		t.Errorf("offset = %d after enterDir, want 0", s.offset)
	}
}

func TestNavGoUpResetsOffset(t *testing.T) {
	s := navigatorScreen{
		entries:     makeEntries(50),
		cursor:      20,
		offset:      15,
		selected:    make(map[int]bool),
		currentPath: "someDir",
		height:      17,
	}
	s = s.goUp()
	if s.offset != 0 {
		t.Errorf("offset = %d after goUp, want 0", s.offset)
	}
}
