package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"ghac/internal/config"
	"ghac/internal/mpd"
	"ghac/internal/snapcast"
	"ghac/internal/ui"
)

func main() {
	themeFlag := flag.String("theme", "", "theme name to use (overrides config and saved state)")
	flag.Parse()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	cfgPath := filepath.Join(home, ".config", ".ghacrc")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: %v\n", err)
		os.Exit(1)
	}

	// Register user-defined themes from config before resolving the active theme
	// so that user themes are available for selection and by name.
	if len(cfg.Themes) > 0 {
		extra := make([]ui.Theme, 0, len(cfg.Themes))
		for _, ct := range cfg.Themes {
			if ct.Name == "" {
				fmt.Fprintf(os.Stderr, "ghac: skipping unnamed theme in config\n")
				continue
			}
			extra = append(extra, ui.Theme{
				Name:          ct.Name,
				BarBG:         ct.BarBG,
				BarFG:         ct.BarFG,
				Accent:        ct.Accent,
				ProgressEmpty: ct.ProgressEmpty,
				Secondary:     ct.Secondary,
				VolumeUnmuted: ct.VolumeUnmuted,
				VolumeMuted:   ct.VolumeMuted,
			})
		}
		ui.AppendThemes(extra)
	}

	// Resolve theme: CLI flag > config file > saved state > default.
	themeName := ui.LoadThemeState()
	if cfg.Theme != "" {
		themeName = cfg.Theme
	}
	if *themeFlag != "" {
		themeName = *themeFlag
	}
	_, themeIdx, ok := ui.ThemeByName(themeName)
	if !ok {
		fmt.Fprintf(os.Stderr, "ghac: unknown theme %q, using default\n", themeName)
		themeIdx = 0
	}

	mpdAddr := fmt.Sprintf("%s:%d", cfg.MPD.IP, cfg.MPD.Port)
	mpdClient, err := mpd.Connect(mpdAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot connect to MPD at %s: %v\n", mpdAddr, err)
		os.Exit(1)
	}
	defer mpdClient.Close()

	initialState, err := mpdClient.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot fetch MPD status: %v\n", err)
		os.Exit(1)
	}

	initialPlaylist, err := mpdClient.PlaylistInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot fetch MPD playlist: %v\n", err)
		os.Exit(1)
	}

	initialNav, err := mpdClient.ListInfo("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot list MPD music library: %v\n", err)
		os.Exit(1)
	}

	snapAddr := fmt.Sprintf("%s:%d", cfg.SnapServer.IP, cfg.SnapServer.Port)
	snapClient, err := snapcast.Connect(snapAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot connect to SnapCast at %s: %v\n", snapAddr, err)
		os.Exit(1)
	}
	defer snapClient.Close()

	snapClients, err := snapClient.GetServerStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ghac: cannot fetch SnapCast status: %v\n", err)
		os.Exit(1)
	}

	m := ui.New(ui.NewParams{
		MPD:         mpdClient,
		MPDState:    initialState,
		Snapcast:    snapClient,
		SnapClients: snapClients,
		Playlist:    initialPlaylist,
		NavEntries:  initialNav,
		ThemeIdx:    themeIdx,
	})
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ghac: %v\n", err)
		os.Exit(1)
	}
}
