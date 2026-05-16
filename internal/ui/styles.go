package ui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle         = lipgloss.NewStyle().Bold(true)
	stylePlaceholder   = lipgloss.NewStyle().Italic(true).Faint(true)
	styleNowPlaying    = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("237")).Foreground(lipgloss.Color("255"))
	styleProgressFill  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleProgressEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleTime          = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleTabActive     = lipgloss.NewStyle().Bold(true).Underline(true)
	styleTabInactive   = lipgloss.NewStyle().Faint(true)
)
