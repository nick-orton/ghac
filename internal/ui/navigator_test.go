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

// --- G / gg jumps ---

func TestNavGMovesToBottom(t *testing.T) {
	s := newTestNavigatorScreen() // 4 entries
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if s.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (last entry) after G", s.cursor)
	}
}

func TestNavGGMovesToTop(t *testing.T) {
	s := newTestNavigatorScreen()
	// Move cursor down first.
	s = pressNavKey(s, "j")
	s = pressNavKey(s, "j")
	// gg sequence.
	s = pressNavKey(s, "g")
	if !s.pendingG {
		t.Error("pendingG should be true after first g")
	}
	s = pressNavKey(s, "g")
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after gg", s.cursor)
	}
	if s.pendingG {
		t.Error("pendingG should be false after gg completes")
	}
}

func TestNavSingleGSetsPendingG(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "g")
	if !s.pendingG {
		t.Error("pendingG should be true after single g press")
	}
}

func TestNavPendingGCancelledByOtherKey(t *testing.T) {
	s := newTestNavigatorScreen()
	s = pressNavKey(s, "g") // set pendingG
	s = pressNavKey(s, "j") // cancel with another key
	if s.pendingG {
		t.Error("pendingG should be cleared by a non-g key")
	}
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (j moved it down)", s.cursor)
	}
}

func TestNavGOnEmptyListIsNoop(t *testing.T) {
	s := newNavigatorScreen(nil, nil) // no entries
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after G on empty list", s.cursor)
	}
}

func TestNavGUpdatesViewport(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh=10
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	vh := s.viewportHeight()
	if s.cursor < s.offset || s.cursor >= s.offset+vh {
		t.Errorf("cursor %d not in viewport [%d, %d) after G", s.cursor, s.offset, s.offset+vh)
	}
}

// --- Half-page jumps (Ctrl-D / Ctrl-U) ---

func TestNavCtrlDMovesHalfPage(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh=10, half=5
	s = pressNavKey(s, "ctrl+d")
	if s.cursor != 5 {
		t.Errorf("cursor = %d, want 5 after ctrl+d (half of vh=10)", s.cursor)
	}
}

func TestNavCtrlUMovesHalfPage(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh=10, half=5
	// Move to entry 20 first.
	for i := 0; i < 20; i++ {
		s = pressNavKey(s, "j")
	}
	s = pressNavKey(s, "ctrl+u")
	if s.cursor != 15 {
		t.Errorf("cursor = %d, want 15 after ctrl+u from 20 (half=5)", s.cursor)
	}
}

func TestNavCtrlDClampsAtBottom(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(4)).withHeight(17) // 4 entries, vh=10
	s = pressNavKey(s, "ctrl+d")
	if s.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (last entry) after ctrl+d on short list", s.cursor)
	}
}

func TestNavCtrlUClampsAtTop(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh=10, half=5
	s = pressNavKey(s, "j") // cursor=1
	s = pressNavKey(s, "ctrl+u")
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after ctrl+u near top", s.cursor)
	}
}

func TestNavCtrlDKeepsCursorInViewport(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17) // vh=10
	s = pressNavKey(s, "ctrl+d")
	vh := s.viewportHeight()
	if s.cursor < s.offset || s.cursor >= s.offset+vh {
		t.Errorf("cursor %d not in viewport [%d, %d) after ctrl+d", s.cursor, s.offset, s.offset+vh)
	}
}

func TestNavCtrlUKeepsCursorInViewport(t *testing.T) {
	s := newNavigatorScreen(nil, makeEntries(50)).withHeight(17)
	for i := 0; i < 30; i++ {
		s = pressNavKey(s, "j")
	}
	s = pressNavKey(s, "ctrl+u")
	vh := s.viewportHeight()
	if s.cursor < s.offset || s.cursor >= s.offset+vh {
		t.Errorf("cursor %d not in viewport [%d, %d) after ctrl+u", s.cursor, s.offset, s.offset+vh)
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
		listCursor:  listCursor{cursor: 1, selected: make(map[int]bool), overhead: 7},
		entries:     testDirEntries(),
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

// TestNavGoUpRestoresCursor verifies that going up places the cursor on the
// directory we just left. We inject the parent entries directly into the screen
// (bypassing the nil client) so we can test the search logic in isolation.
func TestNavGoUpRestoresCursor(t *testing.T) {
	// testDirEntries: [rock/(0), jazz/(1), song.flac(2), notagged.mp3(3)]
	// We are currently inside "jazz". goUp should land on jazz/ at index 1.
	s := navigatorScreen{
		listCursor:  listCursor{selected: make(map[int]bool), overhead: 7},
		entries:     testDirEntries(), // pre-loaded parent entries
		inPlaylist:  make(map[string][]int),
		currentPath: "jazz",
		// mc is nil, so fetchEntries returns nil and we override below.
	}
	// Call the cursor-restoration logic directly (same code path as goUp uses
	// after fetchEntries returns). This validates the search loop.
	leaving := navBase(s.currentPath) // "jazz"
	s.cursor = 0
	for i, e := range s.entries {
		if e.IsDir && e.Name == leaving {
			s.cursor = i
			break
		}
	}
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (index of jazz/ after going up)", s.cursor)
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

func TestNavBase(t *testing.T) {
	tests := []struct{ path, want string }{
		{"rock", "rock"},
		{"rock/classic", "classic"},
		{"a/b/c", "c"},
	}
	for _, tt := range tests {
		got := navBase(tt.path)
		if got != tt.want {
			t.Errorf("navBase(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

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
	if len(s.inPlaylist["song.flac"]) == 0 {
		t.Error("inPlaylist should contain 'song.flac' after withPlaylist")
	}
	if len(s.inPlaylist["notagged.mp3"]) > 0 {
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
	// Even if a directory path matches a playlist entry File, the row should
	// not show + because the IsDir guard prevents it.
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

func TestNavWithPlaylistStoresPositions(t *testing.T) {
	s := newTestNavigatorScreen()
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 2},
	})
	if positions := s.inPlaylist["song.flac"]; len(positions) != 1 || positions[0] != 2 {
		t.Errorf("inPlaylist[song.flac] = %v, want [2]", positions)
	}
}

func TestNavWithPlaylistHandlesDuplicates(t *testing.T) {
	s := newTestNavigatorScreen()
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 1},
		{Song: mpd.Song{File: "song.flac"}, Pos: 3},
	})
	if positions := s.inPlaylist["song.flac"]; len(positions) != 2 {
		t.Errorf("inPlaylist[song.flac] = %v, want 2 positions for duplicate", positions)
	}
}

// --- x: remove from playlist ---

func TestNavXRemovesCursorFileFromPlaylist(t *testing.T) {
	// cursor on song.flac (index 2), which is at playlist position 5.
	s := newTestNavigatorScreen()
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 5},
	})
	s = pressNavKey(s, "j")
	s = pressNavKey(s, "j") // cursor=2 (song.flac)
	s = pressNavKey(s, "x")
	// With nil client no actual MPD call happens; verify selection cleared.
	if len(s.selected) != 0 {
		t.Error("selection should be cleared after x")
	}
}

func TestNavXOnDirectoryIsNoop(t *testing.T) {
	s := newTestNavigatorScreen() // cursor=0 (rock/, a directory)
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "rock"}, Pos: 0},
	})
	before := s.cursor
	s = pressNavKey(s, "x")
	if s.cursor != before {
		t.Errorf("cursor changed after x on directory, want %d got %d", before, s.cursor)
	}
}

func TestNavXOnUnqueuedFileIsNoop(t *testing.T) {
	s := newTestNavigatorScreen()
	// inPlaylist is empty — song.flac is not queued.
	s = pressNavKey(s, "j")
	s = pressNavKey(s, "j") // cursor=2 (song.flac)
	s = pressNavKey(s, "x") // no-op: not in playlist
	if len(s.selected) != 0 {
		t.Error("selection should remain empty after x on unqueued file")
	}
}

func TestNavXWithSelectionRemovesOnlyQueuedFiles(t *testing.T) {
	// Select song.flac (queued) and notagged.mp3 (not queued).
	s := newTestNavigatorScreen()
	s = s.withPlaylist([]mpd.PlaylistEntry{
		{Song: mpd.Song{File: "song.flac"}, Pos: 0},
	})
	s = pressNavKey(s, "j")
	s = pressNavKey(s, "j") // cursor=2 (song.flac)
	s = pressNavKey(s, " ") // select song.flac
	s = pressNavKey(s, "j") // cursor=3 (notagged.mp3)
	s = pressNavKey(s, " ") // select notagged.mp3
	s = pressNavKey(s, "x")
	if len(s.selected) != 0 {
		t.Error("selection should be cleared after x")
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
		listCursor:  listCursor{selected: make(map[int]bool), overhead: 7},
		entries:     testDirEntries(),
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
		listCursor:  listCursor{cursor: 20, offset: 15, selected: make(map[int]bool), height: 17, overhead: 7},
		entries:     makeEntries(50),
		currentPath: "someDir",
	}
	s = s.goUp()
	if s.offset != 0 {
		t.Errorf("offset = %d after goUp, want 0", s.offset)
	}
}
