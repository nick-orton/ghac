package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "ghacrc")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoad(t *testing.T) {
	validTOML := `
[snapserver]
ip = "192.168.1.10"
port = 1705

[mpd]
ip = "192.168.1.10"
port = 6600
`
	tests := []struct {
		name    string
		content string
		path    string // if non-empty, overrides temp file (for missing-file test)
		wantErr string
		wantCfg *Config
	}{
		{
			name:    "valid config",
			content: validTOML,
			wantCfg: &Config{
				SnapServer: ServerConfig{IP: "192.168.1.10", Port: 1705},
				MPD:        ServerConfig{IP: "192.168.1.10", Port: 6600},
			},
		},
		{
			name:    "missing file",
			path:    filepath.Join(t.TempDir(), "nonexistent"),
			wantErr: "config file not found",
		},
		{
			name:    "invalid TOML",
			content: "this is not = valid [toml",
			wantErr: "parsing config",
		},
		{
			name:    "missing mpd.ip",
			content: "[snapserver]\nip = \"192.168.1.10\"\nport = 1705\n[mpd]\nport = 6600\n",
			wantErr: "config: mpd.ip is required",
		},
		{
			name:    "missing mpd.port",
			content: "[snapserver]\nip = \"192.168.1.10\"\nport = 1705\n[mpd]\nip = \"192.168.1.10\"\n",
			wantErr: "config: mpd.port must be between 1 and 65535",
		},
		{
			name:    "invalid mpd.port zero",
			content: "[snapserver]\nip = \"192.168.1.10\"\nport = 1705\n[mpd]\nip = \"192.168.1.10\"\nport = 0\n",
			wantErr: "config: mpd.port must be between 1 and 65535",
		},
		{
			name:    "invalid mpd.port too large",
			content: "[snapserver]\nip = \"192.168.1.10\"\nport = 1705\n[mpd]\nip = \"192.168.1.10\"\nport = 99999\n",
			wantErr: "config: mpd.port must be between 1 and 65535",
		},
		{
			name:    "missing snapserver.ip",
			content: "[snapserver]\nport = 1705\n[mpd]\nip = \"192.168.1.10\"\nport = 6600\n",
			wantErr: "config: snapserver.ip is required",
		},
		{
			name:    "missing snapserver.port",
			content: "[snapserver]\nip = \"192.168.1.10\"\n[mpd]\nip = \"192.168.1.10\"\nport = 6600\n",
			wantErr: "config: snapserver.port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = writeTemp(t, tt.content)
			}

			cfg, err := Load(path)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if got := err.Error(); !strings.Contains(got, tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", got, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantCfg != nil {
				if cfg.MPD.IP != tt.wantCfg.MPD.IP {
					t.Errorf("MPD.IP = %q, want %q", cfg.MPD.IP, tt.wantCfg.MPD.IP)
				}
				if cfg.MPD.Port != tt.wantCfg.MPD.Port {
					t.Errorf("MPD.Port = %d, want %d", cfg.MPD.Port, tt.wantCfg.MPD.Port)
				}
				if cfg.SnapServer.IP != tt.wantCfg.SnapServer.IP {
					t.Errorf("SnapServer.IP = %q, want %q", cfg.SnapServer.IP, tt.wantCfg.SnapServer.IP)
				}
				if cfg.SnapServer.Port != tt.wantCfg.SnapServer.Port {
					t.Errorf("SnapServer.Port = %d, want %d", cfg.SnapServer.Port, tt.wantCfg.SnapServer.Port)
				}
			}
		})
	}
}

