package ui

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

//go:embed themes.toml
var themesData string

// Theme defines the color palette used to render the TUI.
// Modifier-only styles (Bold, Faint, Underline) are not included
// because they do not vary between themes.
type Theme struct {
	Name          string `toml:"name"`
	BarBG         string `toml:"bar_bg"`
	BarFG         string `toml:"bar_fg"`
	Accent        string `toml:"accent"`
	ProgressEmpty string `toml:"progress_empty"`
	Secondary     string `toml:"secondary"`
	VolumeUnmuted string `toml:"volume_unmuted"`
	VolumeMuted   string `toml:"volume_muted"`
}

// Themes is the ordered list of built-in themes loaded from themes.toml
// at build time. Add new themes by editing internal/ui/themes.toml.
var Themes []Theme

func init() {
	var parsed struct {
		Themes []Theme `toml:"themes"`
	}
	if _, err := toml.Decode(themesData, &parsed); err != nil {
		panic("ui: failed to parse themes.toml: " + err.Error())
	}
	Themes = parsed.Themes
}

// AppendThemes adds user-defined themes to the end of the Themes slice so
// they appear after the built-ins in the theme selector. Call this once
// from main, after loading config and before constructing the root model.
func AppendThemes(extra []Theme) {
	Themes = append(Themes, extra...)
}

// ThemeByName looks up a theme by name (case-insensitive).
// Returns the theme, its index in Themes, and whether it was found.
func ThemeByName(name string) (Theme, int, bool) {
	lower := strings.ToLower(name)
	for i, t := range Themes {
		if strings.ToLower(t.Name) == lower {
			return t, i, true
		}
	}
	return Theme{}, 0, false
}

// applyTheme reassigns the color-bearing style variables in styles.go
// to match t. It is safe to call from the Bubble Tea update goroutine.
func applyTheme(t Theme) {
	styleNowPlaying = lipgloss.NewStyle().Bold(true).
		Background(lipgloss.Color(t.BarBG)).Foreground(lipgloss.Color(t.BarFG))
	styleProgressFill = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent))
	styleProgressEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color(t.ProgressEmpty))
	styleTime = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary))
	styleVolumeBarFillUnmuted = lipgloss.NewStyle().Foreground(lipgloss.Color(t.VolumeUnmuted))
	styleVolumeBarFillMuted = lipgloss.NewStyle().Foreground(lipgloss.Color(t.VolumeMuted))
	stylePlaylistCurrent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.Accent))
	styleNavMeta = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secondary))
}

// xdgStateDir returns the XDG_STATE_HOME directory, falling back to
// $HOME/.local/state per the XDG Base Directory Specification.
func xdgStateDir() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}

// themeStatePath returns the XDG state path for theme persistence.
func themeStatePath() (string, error) {
	dir, err := xdgStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghac", "theme"), nil
}

// SaveThemeState writes the theme name to the XDG state file.
// Errors are silently discarded — the theme remains active for the session.
func SaveThemeState(name string) error {
	path, err := themeStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name), 0o644)
}

// LoadThemeState reads the saved theme name.
// Returns "default" if the file is missing or unreadable.
func LoadThemeState() string {
	path, err := themeStatePath()
	if err != nil {
		return "default"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "default"
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "default"
	}
	return name
}
