package ui

// Render symbols used throughout the UI. These default to Unicode characters
// that require a UTF-8 capable terminal. Call UseASCIISymbols() at startup
// to switch to ASCII-only equivalents for legacy terminals (VT220, etc.).
var (
	symCursor    = "▶ "          // list row cursor indicator (2 chars including trailing space)
	symPlay      = "▶"           // now-playing play state icon
	symPause     = "⏸"           // now-playing pause state icon
	symFilled    = "█"           // progress/volume bar filled character
	symEmpty     = "░"           // progress/volume bar empty character
	symEllipsis  = "…"           // ellipsis for truncation (always 1 rune)
	symSeparator = " \u2013 "    // field separator: space + en-dash + space
)

// UseASCIISymbols switches all render symbols to ASCII-only equivalents.
// Call this before creating the root model when legacy mode is active.
func UseASCIISymbols() {
	symCursor    = "> "
	symPlay      = ">"
	symPause     = "|"
	symFilled    = "#"
	symEmpty     = "-"
	symEllipsis  = "."
	symSeparator = " - "
}
