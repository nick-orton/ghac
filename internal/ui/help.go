package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// helpScreen displays all keybindings organized by section.
// Esc is handled by the root model to return to the previous screen.
type helpScreen struct{}

func newHelpScreen() helpScreen {
	return helpScreen{}
}

func (s helpScreen) Update(msg tea.Msg) (helpScreen, tea.Cmd) {
	return s, nil
}

func (s helpScreen) View() string {
	var b strings.Builder

	b.WriteString(styleHelpSection.Render("Global"))
	b.WriteString("\n")
	b.WriteString(helpRow("1", "Switch to Player Volume screen"))
	b.WriteString(helpRow("2", "Switch to Playlist Control screen"))
	b.WriteString(helpRow("3", "Switch to Library Navigator screen"))
	b.WriteString(helpRow("?", "Open help screen"))
	b.WriteString(helpRow("ctrl+t", "Open theme selector"))
	b.WriteString(helpRow("p", "Toggle play / pause"))
	b.WriteString(helpRow("z", "Toggle random (shuffle) mode"))
	b.WriteString(helpRow("q / Ctrl-C", "Quit"))
	b.WriteString(helpRow("Esc", "Close modal / return to screen"))
	b.WriteString("\n")

	b.WriteString(styleHelpSection.Render("Player Volume"))
	b.WriteString("\n")
	b.WriteString(helpRow("j / k", "Move cursor down / up"))
	b.WriteString(helpRow("h / l", "Decrease / increase focused client volume by 5%"))
	b.WriteString(helpRow("m", "Toggle mute on focused client"))
	b.WriteString(helpRow("H / L", "Decrease / increase all clients volume by 5%"))
	b.WriteString(helpRow("M", "Toggle mute on all clients"))
	b.WriteString(helpRow("Ctrl-R", "Rename focused client"))
	b.WriteString("\n")

	b.WriteString(styleHelpSection.Render("Playlist Control"))
	b.WriteString("\n")
	b.WriteString(helpRow("j / k", "Move cursor down / up"))
	b.WriteString(helpRow("gg / G", "Move cursor to top / bottom"))
	b.WriteString(helpRow("f <letter>", "Jump to next entry starting with letter"))
	b.WriteString(helpRow("Ctrl-J / Ctrl-K", "Move song under cursor down / up"))
	b.WriteString(helpRow("space", "Toggle selection on song under cursor"))
	b.WriteString(helpRow("x", "Remove selected song(s) (or cursor song if none selected)"))
	b.WriteString(helpRow("X", "Clear the entire playlist"))
	b.WriteString(helpRow("enter", "Start playing the song under cursor"))
	b.WriteString("\n")

	b.WriteString(styleHelpSection.Render("Library Navigator"))
	b.WriteString("\n")
	b.WriteString(helpRow("j / k", "Move cursor down / up"))
	b.WriteString(helpRow("gg / G", "Move cursor to top / bottom"))
	b.WriteString(helpRow("f <letter>", "Jump to next entry starting with letter"))
	b.WriteString(helpRow("Ctrl-D / Ctrl-U", "Move cursor down / up half a page"))
	b.WriteString(helpRow("h", "Navigate to parent directory"))
	b.WriteString(helpRow("l", "Enter directory under cursor"))
	b.WriteString(helpRow("space", "Toggle selection on entry under cursor"))
	b.WriteString(helpRow("x", "Remove selected file(s) from playlist (skips dirs / unqueued)"))
	b.WriteString(helpRow("enter", "Enqueue selected entries (or cursor entry if none selected)"))

	return b.String()
}

func helpRow(key, desc string) string {
	k := styleHelpKey.Render(fmt.Sprintf("%-12s", key))
	d := styleHelpDesc.Render(desc)
	return "  " + k + "  " + d + "\n"
}
