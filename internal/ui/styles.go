package ui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle         = lipgloss.NewStyle().Bold(true)
	stylePlaceholder   = lipgloss.NewStyle().Italic(true).Faint(true)
	styleNowPlaying    = lipgloss.NewStyle().Bold(true).Reverse(true)
	styleProgressFill  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleProgressEmpty = lipgloss.NewStyle().Faint(true)
	styleTime          = lipgloss.NewStyle().Faint(true)
)
