package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds all configuration for ghac.
type Config struct {
	SnapServer ServerConfig  `toml:"snapserver"`
	MPD        ServerConfig  `toml:"mpd"`
	Theme      string        `toml:"theme"`  // optional; selects a built-in or user theme by name
	Themes     []ThemeConfig `toml:"themes"` // optional; user-defined themes appended after built-ins
}

// ServerConfig holds the connection details for a backend server.
type ServerConfig struct {
	IP   string `toml:"ip"`
	Port int    `toml:"port"`
}

// ThemeConfig defines a user-supplied color theme. Field names match the
// built-in themes.toml format so users can copy entries from that file.
// All color values are xterm-256 color numbers (0–255) expressed as strings.
type ThemeConfig struct {
	Name          string `toml:"name"`
	BarBG         string `toml:"bar_bg"`
	BarFG         string `toml:"bar_fg"`
	Accent        string `toml:"accent"`
	ProgressEmpty string `toml:"progress_empty"`
	Secondary     string `toml:"secondary"`
	VolumeUnmuted string `toml:"volume_unmuted"`
	VolumeMuted   string `toml:"volume_muted"`
}

// Load reads and validates the config file at the given path.
// It returns a descriptive error if the file is missing, not valid TOML,
// or has missing/invalid fields.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file not found: %s", path)
		}
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.MPD.IP == "" {
		return fmt.Errorf("config: mpd.ip is required")
	}
	if cfg.MPD.Port <= 0 || cfg.MPD.Port > 65535 {
		return fmt.Errorf("config: mpd.port must be between 1 and 65535")
	}
	if cfg.SnapServer.IP == "" {
		return fmt.Errorf("config: snapserver.ip is required")
	}
	if cfg.SnapServer.Port <= 0 || cfg.SnapServer.Port > 65535 {
		return fmt.Errorf("config: snapserver.port must be between 1 and 65535")
	}
	return nil
}
