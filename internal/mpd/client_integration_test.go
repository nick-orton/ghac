//go:build integration

package mpd

import (
	"os"
	"testing"
	"time"
)

// mpdTestAddr returns the MPD address to test against.
// Defaults to localhost:6600; override with MPD_TEST_ADDR env var.
func mpdTestAddr() string {
	if addr := os.Getenv("MPD_TEST_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6600"
}

func TestIntegrationConnect(t *testing.T) {
	c, err := Connect(mpdTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
}

func TestIntegrationPing(t *testing.T) {
	c, err := Connect(mpdTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	if err := c.Ping(); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestIntegrationStatus(t *testing.T) {
	c, err := Connect(mpdTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	state, err := c.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	validStates := map[string]bool{"play": true, "pause": true, "stop": true}
	if !validStates[state.Status] {
		t.Errorf("Status.Status = %q, want play/pause/stop", state.Status)
	}
}

func TestIntegrationCurrentSong(t *testing.T) {
	c, err := Connect(mpdTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	_, err = c.CurrentSong()
	if err != nil {
		t.Errorf("CurrentSong: %v", err)
	}
}

func TestIntegrationPlayPause(t *testing.T) {
	c, err := Connect(mpdTestAddr())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	before, err := c.Status()
	if err != nil {
		t.Fatalf("Status before: %v", err)
	}

	// Toggle: if playing → pause, if paused/stopped → play.
	if before.Status == "play" {
		if err := c.Pause(); err != nil {
			t.Fatalf("Pause: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
		after, err := c.Status()
		if err != nil {
			t.Fatalf("Status after Pause: %v", err)
		}
		if after.Status != "pause" {
			t.Errorf("after Pause: status = %q, want %q", after.Status, "pause")
		}
		// Restore.
		_ = c.Play()
	} else {
		if err := c.Play(); err != nil {
			t.Fatalf("Play: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
		after, err := c.Status()
		if err != nil {
			t.Fatalf("Status after Play: %v", err)
		}
		if after.Status != "play" {
			t.Errorf("after Play: status = %q, want %q", after.Status, "play")
		}
		// Restore.
		_ = c.Pause()
	}
}
