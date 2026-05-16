package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/snapcast"
)

// testClients returns a repeatable slice of SnapClients for use in tests.
func testClients() []snapcast.SnapClient {
	return []snapcast.SnapClient{
		{ID: "aa", Name: "Living Room", Volume: 50, Muted: false},
		{ID: "bb", Name: "Kitchen", Volume: 80, Muted: true},
		{ID: "cc", Name: "Bedroom", Volume: 0, Muted: false},
	}
}

func newTestVolumeScreen() volumeScreen {
	return newVolumeScreen(nil, testClients())
}

func pressKey(s volumeScreen, key string) volumeScreen {
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

// --- Cursor movement ---

func TestCursorMoveDown(t *testing.T) {
	s := newTestVolumeScreen() // cursor=0
	s = pressKey(s, "j")
	if s.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after j", s.cursor)
	}
}

func TestCursorMoveUp(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "j") // cursor=1
	s = pressKey(s, "k") // cursor=0
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after j then k", s.cursor)
	}
}

func TestCursorDoesNotGoAboveZero(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "k") // already at 0
	if s.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after k at top", s.cursor)
	}
}

func TestCursorDoesNotGoBelowBottom(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "j")
	s = pressKey(s, "j")
	s = pressKey(s, "j") // already at last
	if s.cursor != 2 {
		t.Errorf("cursor = %d, want 2 after pressing j past the bottom", s.cursor)
	}
}

// --- Volume adjustment ---

func TestIncreaseVolumeFocusedClient(t *testing.T) {
	s := newTestVolumeScreen() // cursor=0, volume=50
	s = pressKey(s, "l")
	if s.clients[0].Volume != 55 {
		t.Errorf("volume = %d, want 55 after l", s.clients[0].Volume)
	}
	// Other clients unchanged.
	if s.clients[1].Volume != 80 {
		t.Errorf("client 1 volume changed unexpectedly to %d", s.clients[1].Volume)
	}
}

func TestDecreaseVolumeFocusedClient(t *testing.T) {
	s := newTestVolumeScreen() // cursor=0, volume=50
	s = pressKey(s, "h")
	if s.clients[0].Volume != 45 {
		t.Errorf("volume = %d, want 45 after h", s.clients[0].Volume)
	}
}

func TestVolumeClampAtMax(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "j") // cursor=1, volume=80
	// Press l enough times to exceed 100.
	for i := 0; i < 10; i++ {
		s = pressKey(s, "l")
	}
	if s.clients[1].Volume != 100 {
		t.Errorf("volume = %d, want 100 (clamped at max)", s.clients[1].Volume)
	}
}

func TestVolumeClampAtMin(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "j")
	s = pressKey(s, "j") // cursor=2, volume=0
	// Already at 0; pressing h should keep it at 0.
	s = pressKey(s, "h")
	if s.clients[2].Volume != 0 {
		t.Errorf("volume = %d, want 0 (clamped at min)", s.clients[2].Volume)
	}
}

func TestIncreaseAllVolumes(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "L")
	if s.clients[0].Volume != 55 {
		t.Errorf("client 0 volume = %d, want 55", s.clients[0].Volume)
	}
	if s.clients[1].Volume != 85 {
		t.Errorf("client 1 volume = %d, want 85", s.clients[1].Volume)
	}
	if s.clients[2].Volume != 5 {
		t.Errorf("client 2 volume = %d, want 5", s.clients[2].Volume)
	}
}

func TestDecreaseAllVolumes(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "H")
	if s.clients[0].Volume != 45 {
		t.Errorf("client 0 volume = %d, want 45", s.clients[0].Volume)
	}
	if s.clients[1].Volume != 75 {
		t.Errorf("client 1 volume = %d, want 75", s.clients[1].Volume)
	}
	// client 2 was at 0; clamped to 0.
	if s.clients[2].Volume != 0 {
		t.Errorf("client 2 volume = %d, want 0 (clamped)", s.clients[2].Volume)
	}
}

// --- Mute toggle ---

func TestToggleMuteFocusedClient(t *testing.T) {
	s := newTestVolumeScreen() // cursor=0, muted=false
	s = pressKey(s, "m")
	if !s.clients[0].Muted {
		t.Error("client 0 should be muted after m")
	}
	s = pressKey(s, "m")
	if s.clients[0].Muted {
		t.Error("client 0 should be unmuted after second m")
	}
}

func TestToggleMuteAllClients(t *testing.T) {
	s := newTestVolumeScreen()
	// Initial muted states: [false, true, false]
	s = pressKey(s, "M")
	if s.clients[0].Muted != true {
		t.Errorf("client 0 muted = %v, want true", s.clients[0].Muted)
	}
	if s.clients[1].Muted != false {
		t.Errorf("client 1 muted = %v, want false (was true, should toggle)", s.clients[1].Muted)
	}
	if s.clients[2].Muted != true {
		t.Errorf("client 2 muted = %v, want true", s.clients[2].Muted)
	}
}

// --- MsgClientsUpdated handling ---

func TestMsgClientsUpdatedReplacesClientList(t *testing.T) {
	s := newTestVolumeScreen()
	newClients := []snapcast.SnapClient{
		{ID: "xx", Name: "New Room", Volume: 30, Muted: false},
	}
	s = s.withClients(newClients)
	if len(s.clients) != 1 {
		t.Fatalf("clients len = %d, want 1", len(s.clients))
	}
	if s.clients[0].Name != "New Room" {
		t.Errorf("client name = %q, want \"New Room\"", s.clients[0].Name)
	}
}

func TestMsgClientsUpdatedClampsCursor(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "j")
	s = pressKey(s, "j") // cursor=2

	// Shrink to 1 client — cursor must clamp.
	s = s.withClients([]snapcast.SnapClient{
		{ID: "yy", Name: "Only", Volume: 60},
	})
	if s.cursor != 0 {
		t.Errorf("cursor = %d after shrink, want 0", s.cursor)
	}
}

func TestMsgClientsUpdatedEmptyList(t *testing.T) {
	s := newTestVolumeScreen()
	s = pressKey(s, "j")

	s = s.withClients(nil)
	if s.cursor != 0 {
		t.Errorf("cursor = %d after empty update, want 0", s.cursor)
	}
	if len(s.clients) != 0 {
		t.Errorf("clients len = %d, want 0", len(s.clients))
	}
}

// --- Root model integration ---

func TestRootModelHandlesMsgClientsUpdated(t *testing.T) {
	m := newTestModel()
	clients := []snapcast.SnapClient{
		{ID: "z1", Name: "Zone 1", Volume: 42},
	}
	updated, cmd := m.Update(snapcast.MsgClientsUpdated{Clients: clients})
	m = updated.(Model)

	// cmd should be nil because snapClient is nil in the test model.
	_ = cmd

	if len(m.volume.clients) != 1 {
		t.Fatalf("volume.clients len = %d, want 1", len(m.volume.clients))
	}
	if m.volume.clients[0].Volume != 42 {
		t.Errorf("volume = %d, want 42", m.volume.clients[0].Volume)
	}
}

func TestRootModelSnapcastMsgErrorQuits(t *testing.T) {
	m := newTestModel()
	updated, cmd := m.Update(snapcast.MsgError{Err: errors.New("snapcast gone")})
	m = updated.(Model)

	if m.errMsg == "" {
		t.Error("errMsg should be set after snapcast.MsgError")
	}
	if cmd == nil {
		t.Fatal("expected quit cmd after snapcast.MsgError")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

// --- Volume bar rendering ---

func TestRenderVolumeBarFull(t *testing.T) {
	bar := renderVolumeBar(100, false)
	if !strings.Contains(bar, "█") {
		t.Error("full bar should contain filled blocks")
	}
}

func TestRenderVolumeBarEmpty(t *testing.T) {
	bar := renderVolumeBar(0, false)
	if strings.Contains(bar, "█") {
		t.Error("empty bar should not contain filled blocks")
	}
}

func TestRenderVolumeBarMidRange(t *testing.T) {
	// 50% of 20 chars = 10 filled, 10 empty.
	bar := renderVolumeBar(50, false)
	if !strings.Contains(bar, "█") {
		t.Error("50% bar should contain filled blocks")
	}
	if !strings.Contains(bar, "░") {
		t.Error("50% bar should contain unfilled blocks")
	}
}

func TestRenderVolumeBarMutedHasCorrectBlocks(t *testing.T) {
	// Muted bars use different colors but the same block structure.
	// 60% of 20 = 12 filled, 8 empty.
	for _, muted := range []bool{false, true} {
		bar := renderVolumeBar(60, muted)
		filled := strings.Count(bar, "█")
		empty := strings.Count(bar, "░")
		if filled != 12 {
			t.Errorf("muted=%v: filled = %d, want 12", muted, filled)
		}
		if empty != 8 {
			t.Errorf("muted=%v: empty = %d, want 8", muted, empty)
		}
	}
}

func TestTruncateName(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"toolongstring", 10, "toolongst…"},
	}
	for _, tt := range tests {
		got := truncateName(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateName(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestViewShowsNoClientsMessage(t *testing.T) {
	s := newVolumeScreen(nil, nil)
	view := s.View()
	if !strings.Contains(view, "No clients connected") {
		t.Error("empty client list should show 'No clients connected'")
	}
}

func TestViewShowsClientNames(t *testing.T) {
	s := newTestVolumeScreen()
	view := s.View()
	if !strings.Contains(view, "Living Room") {
		t.Error("view should contain client name 'Living Room'")
	}
	if !strings.Contains(view, "Kitchen") {
		t.Error("view should contain client name 'Kitchen'")
	}
}

func TestViewShowsMuteIndicator(t *testing.T) {
	s := newTestVolumeScreen() // client 1 is muted
	view := s.View()
	if !strings.Contains(view, "[M]") {
		t.Error("view should contain '[M]' mute indicator for muted client")
	}
}
