package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/mpd"
)

// testEntries returns a repeatable playlist for use in tests.
func testEntries() []mpd.PlaylistEntry {
	return []mpd.PlaylistEntry{
		{Song: mpd.Song{Title: "Alpha", Artist: "Band A", File: "a.flac"}, Pos: 0},
		{Song: mpd.Song{Title: "Beta", Artist: "Band B", File: "b.flac"}, Pos: 1},
		{Song: mpd.Song{Title: "Gamma", Artist: "Band C", File: "c.flac"}, Pos: 2},
		{Song: mpd.Song{Title: "Delta", Artist: "Band D", File: "d.flac"}, Pos: 3},
	}
}

func newTestPlaylistScreen() playlistScreen {
	return newPlaylistScreen(nil, testEntries(), -1)
}

func pressPlaylistKey(s playlistScreen, key string) playlistScreen {
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

// --- Cursor movement ---

func TestPlaylistCursorMoveDown(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, "j")
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after j", s.cursor)
	}
}

func TestPlaylistCursorMoveUp(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, "j") // cursor=1
	s = pressPlaylistKey(s, "k") // cursor=0
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after j then k", s.cursor)
	}
}

func TestPlaylistCursorDoesNotGoAboveZero(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, "k")
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after k at top", s.cursor)
	}
}

func TestPlaylistCursorDoesNotGoBelowBottom(t *testing.T) {
	s := newTestPlaylistScreen() // 4 entries
	for i := 0; i < 10; i++ {
		s = pressPlaylistKey(s, "j")
	}
	if s.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (last entry) after pressing j past bottom", s.cursor)
	}
}

// --- Selection toggle ---

func TestPlaylistToggleSelection(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, " ")
	if !s.selected[0] {
		t.Error("entry 0 should be selected after space")
	}
	s = pressPlaylistKey(s, " ")
	if s.selected[0] {
		t.Error("entry 0 should be deselected after second space")
	}
}

func TestPlaylistSelectionIndependentOfCursor(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, " ") // select entry 0
	s = pressPlaylistKey(s, "j") // move cursor to 1
	if !s.selected[0] {
		t.Error("selection on entry 0 should persist after cursor moves to 1")
	}
	if s.selected[1] {
		t.Error("entry 1 should not be selected after cursor moves there")
	}
}

// --- Remove (x): cursor song when no selection ---

func TestPlaylistRemoveCursorSong(t *testing.T) {
	s := newTestPlaylistScreen() // cursor=0, 4 entries
	s = pressPlaylistKey(s, "x")
	if len(s.entries) != 3 {
		t.Fatalf("entries len = %d, want 3 after removing cursor song", len(s.entries))
	}
	// Alpha (was pos 0) should be gone; Beta is now first.
	if s.entries[0].Title != "Beta" {
		t.Errorf("entries[0].Title = %q, want %q after removing Alpha", s.entries[0].Title, "Beta")
	}
}

func TestPlaylistRemoveCursorSongClampsToLast(t *testing.T) {
	s := newTestPlaylistScreen()
	// Move cursor to last entry.
	for i := 0; i < 3; i++ {
		s = pressPlaylistKey(s, "j")
	}
	if s.cursor != 3 {
		t.Fatalf("cursor = %d, want 3 before removing last entry", s.cursor)
	}
	s = pressPlaylistKey(s, "x") // remove last entry
	if s.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (clamped to new last) after removing last entry", s.cursor)
	}
}

// --- Remove (x): selected songs ---

func TestPlaylistRemoveSelectedSongs(t *testing.T) {
	s := newTestPlaylistScreen()
	// Select entries 0 and 2.
	s = pressPlaylistKey(s, " ") // select 0
	s = pressPlaylistKey(s, "j")
	s = pressPlaylistKey(s, "j") // cursor at 2
	s = pressPlaylistKey(s, " ") // select 2
	s = pressPlaylistKey(s, "x")

	if len(s.entries) != 2 {
		t.Fatalf("entries len = %d, want 2 after removing 2 selected songs", len(s.entries))
	}
	// Only Beta (1) and Delta (3) should remain.
	if s.entries[0].Title != "Beta" {
		t.Errorf("entries[0].Title = %q, want Beta", s.entries[0].Title)
	}
	if s.entries[1].Title != "Delta" {
		t.Errorf("entries[1].Title = %q, want Delta", s.entries[1].Title)
	}
}

func TestPlaylistRemoveClearsSelection(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, " ") // select 0
	s = pressPlaylistKey(s, "x")
	if len(s.selected) != 0 {
		t.Error("selection should be cleared after x")
	}
}

// --- Clear (X) ---

func TestPlaylistClearAll(t *testing.T) {
	s := newTestPlaylistScreen()
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if len(s.entries) != 0 {
		t.Errorf("entries len = %d, want 0 after X", len(s.entries))
	}
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after X", s.cursor)
	}
}

// --- Enter to play ---

func TestPlaylistEnterWithNoClientIsNoop(t *testing.T) {
	s := newTestPlaylistScreen() // mc is nil
	s = pressPlaylistKey(s, "j") // cursor=1
	// Should not panic.
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 unchanged after enter with nil client", s.cursor)
	}
}

// --- withEntries ---

func TestPlaylistWithEntriesReplacesEntries(t *testing.T) {
	s := newTestPlaylistScreen()
	newEntries := []mpd.PlaylistEntry{
		{Song: mpd.Song{Title: "New Song", Artist: "New Artist", File: "new.flac"}, Pos: 0},
	}
	s = s.withEntries(newEntries, 0)
	if len(s.entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(s.entries))
	}
	if s.entries[0].Title != "New Song" {
		t.Errorf("entries[0].Title = %q, want New Song", s.entries[0].Title)
	}
}

func TestPlaylistWithEntriesClearSelection(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, " ") // select 0
	s = s.withEntries(testEntries(), -1)
	if len(s.selected) != 0 {
		t.Error("selection should be cleared after withEntries")
	}
}

func TestPlaylistWithEntriesClampsCursor(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, "j")
	s = pressPlaylistKey(s, "j")
	s = pressPlaylistKey(s, "j") // cursor=3

	// Shrink to 2 entries — cursor must clamp.
	s = s.withEntries(testEntries()[:2], -1)
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after shrinking entries to 2", s.cursor)
	}
}

func TestPlaylistWithEntriesUpdatesCurrentPos(t *testing.T) {
	s := newTestPlaylistScreen()
	s = s.withEntries(testEntries(), 2)
	if s.currentPos != 2 {
		t.Errorf("currentPos = %d, want 2 after withEntries", s.currentPos)
	}
}

// --- withCurrentPos ---

func TestPlaylistWithCurrentPos(t *testing.T) {
	s := newTestPlaylistScreen() // currentPos = -1
	s = s.withCurrentPos(1)
	if s.currentPos != 1 {
		t.Errorf("currentPos = %d, want 1 after withCurrentPos", s.currentPos)
	}
}

// --- Root model integration ---

func TestRootModelHandlesMsgPlaylistChanged(t *testing.T) {
	m := newTestModel()
	entries := []mpd.PlaylistEntry{
		{Song: mpd.Song{Title: "Song A", Artist: "Artist A", File: "a.flac"}, Pos: 0},
		{Song: mpd.Song{Title: "Song B", Artist: "Artist B", File: "b.flac"}, Pos: 1},
	}
	updated, _ := m.Update(mpd.MsgPlaylistChanged{Entries: entries})
	m = updated.(Model)

	if len(m.playlist.entries) != 2 {
		t.Fatalf("playlist.entries len = %d, want 2", len(m.playlist.entries))
	}
	if m.playlist.entries[0].Title != "Song A" {
		t.Errorf("entries[0].Title = %q, want Song A", m.playlist.entries[0].Title)
	}
}

func TestRootModelMsgPlayerStateUpdatesSongPos(t *testing.T) {
	m := newTestModel()
	msg := mpd.MsgPlayerState{
		Status:  "play",
		Song:    mpd.Song{Title: "Track"},
		SongPos: 3,
	}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.currentSongPos != 3 {
		t.Errorf("currentSongPos = %d, want 3", m.currentSongPos)
	}
	if m.playlist.currentPos != 3 {
		t.Errorf("playlist.currentPos = %d, want 3", m.playlist.currentPos)
	}
}

// --- View rendering ---

func TestPlaylistViewEmptyPlaylist(t *testing.T) {
	s := newPlaylistScreen(nil, nil, -1)
	view := s.View()
	if !strings.Contains(view, "Playlist is empty") {
		t.Errorf("empty playlist view should contain 'Playlist is empty', got: %q", view)
	}
}

func TestPlaylistViewShowsSongTitles(t *testing.T) {
	s := newTestPlaylistScreen()
	view := s.View()
	if !strings.Contains(view, "Alpha") {
		t.Error("view should contain song title 'Alpha'")
	}
	if !strings.Contains(view, "Beta") {
		t.Error("view should contain song title 'Beta'")
	}
}

func TestPlaylistViewShowsPlayingMarker(t *testing.T) {
	s := newPlaylistScreen(nil, testEntries(), 1) // entry 1 is playing
	view := s.View()
	if !strings.Contains(view, ">") {
		t.Error("view should contain '>' marker for currently-playing song")
	}
}

func TestPlaylistViewShowsSelectedMarker(t *testing.T) {
	s := newTestPlaylistScreen()
	s = pressPlaylistKey(s, " ") // select entry 0
	view := s.View()
	if !strings.Contains(view, "*") {
		t.Error("view should contain '*' marker for selected song")
	}
}

func TestPlaylistViewShowsFilenameAsFallback(t *testing.T) {
	entries := []mpd.PlaylistEntry{
		{Song: mpd.Song{File: "no-metadata.flac"}, Pos: 0},
	}
	s := newPlaylistScreen(nil, entries, -1)
	view := s.View()
	if !strings.Contains(view, "no-metadata.flac") {
		t.Errorf("view should show filename when no title metadata: %q", view)
	}
}

// --- entryDisplayName ---

func TestEntryDisplayNameTitleAndArtist(t *testing.T) {
	e := mpd.PlaylistEntry{Song: mpd.Song{Title: "My Song", Artist: "My Artist", File: "f.flac"}}
	got := entryDisplayName(e)
	want := "My Song \u2013 My Artist"
	if got != want {
		t.Errorf("entryDisplayName = %q, want %q", got, want)
	}
}

func TestEntryDisplayNameTitleOnly(t *testing.T) {
	e := mpd.PlaylistEntry{Song: mpd.Song{Title: "Only Title", File: "f.flac"}}
	got := entryDisplayName(e)
	if got != "Only Title" {
		t.Errorf("entryDisplayName = %q, want %q", got, "Only Title")
	}
}

func TestEntryDisplayNameFilenameFallback(t *testing.T) {
	e := mpd.PlaylistEntry{Song: mpd.Song{File: "raw-file.mp3"}}
	got := entryDisplayName(e)
	if got != "raw-file.mp3" {
		t.Errorf("entryDisplayName = %q, want %q", got, "raw-file.mp3")
	}
}
