package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// themeScreen is the theme selector modal. It displays the list of
// built-in themes with a cursor. Moving the cursor applies each theme
// in real time so the user can preview before confirming.
//
// The root model (model.go) handles Enter (confirm) and Esc (revert);
// this struct handles only cursor movement.
type themeScreen struct {
	cursor int // index into Themes
}

func newThemeScreen(initial int) themeScreen {
	return themeScreen{cursor: initial}
}

// Update handles j/k cursor movement and applies the highlighted theme.
func (s themeScreen) Update(msg tea.Msg) (themeScreen, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch keyMsg.String() {
	case "j":
		if s.cursor < len(Themes)-1 {
			s.cursor++
			applyTheme(Themes[s.cursor])
		}
	case "k":
		if s.cursor > 0 {
			s.cursor--
			applyTheme(Themes[s.cursor])
		}
	}
	return s, nil
}

// View returns the theme list content (no border — modalBorder is added
// by the root model's View, following the same pattern as the help modal).
func (s themeScreen) View() string {
	var b strings.Builder
	for i, t := range Themes {
		var line string
		if i == s.cursor {
			line = styleRowActive.Render(symCursor + t.Name)
		} else {
			line = "  " + t.Name
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(styleHelpDesc.Render("  [enter] confirm  [esc] cancel"))
	b.WriteString("\n")
	return b.String()
}
