//go:build integration

package snapcast

import (
	"os"
	"testing"
)

// snapTestAddr returns the SnapCast address to test against.
// Defaults to localhost:1705; override with SNAPCAST_TEST_ADDR env var.
func snapTestAddr() string {
	if addr := os.Getenv("SNAPCAST_TEST_ADDR"); addr != "" {
		return addr
	}
	return "localhost:1705"
}

func TestIntegrationConnect(t *testing.T) {
	c, err := Connect(snapTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
}

func TestIntegrationGetServerStatus(t *testing.T) {
	c, err := Connect(snapTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	clients, err := c.GetServerStatus()
	if err != nil {
		t.Fatalf("GetServerStatus: %v", err)
	}
	t.Logf("GetServerStatus returned %d client(s)", len(clients))
	for _, cl := range clients {
		t.Logf("  client id=%q name=%q volume=%d muted=%v", cl.ID, cl.Name, cl.Volume, cl.Muted)
	}
}

func TestIntegrationSetVolume(t *testing.T) {
	c, err := Connect(snapTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	clients, err := c.GetServerStatus()
	if err != nil {
		t.Fatalf("GetServerStatus: %v", err)
	}
	if len(clients) == 0 {
		t.Skip("no SnapCast clients connected; skipping SetVolume test")
	}

	target := clients[0]
	originalVol := target.Volume

	newVol := originalVol + 5
	if newVol > 100 {
		newVol = originalVol - 5
	}

	if err := c.SetVolume(target.ID, newVol, target.Muted); err != nil {
		t.Fatalf("SetVolume: %v", err)
	}

	// Restore original volume.
	if err := c.SetVolume(target.ID, originalVol, target.Muted); err != nil {
		t.Errorf("SetVolume (restore): %v", err)
	}
}

func TestIntegrationSetMute(t *testing.T) {
	c, err := Connect(snapTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	clients, err := c.GetServerStatus()
	if err != nil {
		t.Fatalf("GetServerStatus: %v", err)
	}
	if len(clients) == 0 {
		t.Skip("no SnapCast clients connected; skipping SetMute test")
	}

	target := clients[0]

	// Toggle mute and restore.
	if err := c.SetMute(target.ID, !target.Muted, target.Volume); err != nil {
		t.Fatalf("SetMute: %v", err)
	}
	if err := c.SetMute(target.ID, target.Muted, target.Volume); err != nil {
		t.Errorf("SetMute (restore): %v", err)
	}
}
