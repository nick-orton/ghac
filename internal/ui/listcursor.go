package ui

import "strings"

// listCursor manages the cursor, viewport, and selection state shared by all
// scrollable list screens. Embed it in a screen struct to gain the navigation
// fields and methods; screens supply their entry count and a name-getter for
// f<letter> jumping.
type listCursor struct {
	cursor   int
	offset   int
	pendingG bool
	pendingF bool
	selected map[int]bool
	height   int
	overhead int // fixed lines consumed by chrome (now-playing, tabs, border, extras)
}

func newListCursor(overhead int) listCursor {
	return listCursor{
		selected: make(map[int]bool),
		overhead: overhead,
	}
}

// viewportHeight returns the number of entry rows that fit on screen.
// Returns 24 as a sensible default before the first WindowSizeMsg arrives.
func (c listCursor) viewportHeight() int {
	if c.height < 10 {
		return 24
	}
	h := c.height - c.overhead
	if h < 1 {
		h = 1
	}
	return h
}

// clampOffset adjusts the viewport so the cursor row is always visible.
func (c listCursor) clampOffset() listCursor {
	vh := c.viewportHeight()
	if c.cursor < c.offset {
		c.offset = c.cursor
	} else if c.cursor >= c.offset+vh {
		c.offset = c.cursor - vh + 1
	}
	if c.offset < 0 {
		c.offset = 0
	}
	return c
}

// withHeight updates the terminal height and re-clamps the viewport.
func (c listCursor) withHeight(h int) listCursor {
	c.height = h
	return c.clampOffset()
}

// capturePending captures and clears the pending-G and pending-F flags.
// Call at the start of key handling; use the returned booleans to dispatch
// multi-key sequences.
func (c listCursor) capturePending() (pendingG, pendingF bool, updated listCursor) {
	pendingG, pendingF = c.pendingG, c.pendingF
	c.pendingG = false
	c.pendingF = false
	return pendingG, pendingF, c
}

// moveDown moves the cursor one entry towards the end of an n-entry list.
func (c listCursor) moveDown(n int) listCursor {
	if c.cursor < n-1 {
		c.cursor++
		return c.clampOffset()
	}
	return c
}

// moveUp moves the cursor one entry towards the top.
func (c listCursor) moveUp() listCursor {
	if c.cursor > 0 {
		c.cursor--
		return c.clampOffset()
	}
	return c
}

// moveToEnd moves the cursor to the last entry in an n-entry list.
func (c listCursor) moveToEnd(n int) listCursor {
	if n > 0 {
		c.cursor = n - 1
		return c.clampOffset()
	}
	return c
}

// moveToTop moves the cursor to the first entry.
func (c listCursor) moveToTop() listCursor {
	c.cursor = 0
	return c.clampOffset()
}

// halfPageDown moves the cursor down by half the viewport height.
func (c listCursor) halfPageDown(n int) listCursor {
	c.cursor += c.viewportHeight() / 2
	if c.cursor >= n {
		c.cursor = n - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
	return c.clampOffset()
}

// halfPageUp moves the cursor up by half the viewport height.
func (c listCursor) halfPageUp() listCursor {
	c.cursor -= c.viewportHeight() / 2
	if c.cursor < 0 {
		c.cursor = 0
	}
	return c.clampOffset()
}

// toggleSelected toggles the selection of entry i in an n-entry list.
// The selected map is copied before mutation to preserve value semantics.
func (c listCursor) toggleSelected(i, n int) listCursor {
	if i >= n {
		return c
	}
	sel := make(map[int]bool, len(c.selected))
	for k, v := range c.selected {
		sel[k] = v
	}
	if sel[i] {
		delete(sel, i)
	} else {
		sel[i] = true
	}
	c.selected = sel
	return c
}

// clearSelection returns a copy with an empty selection map.
func (c listCursor) clearSelection() listCursor {
	c.selected = make(map[int]bool)
	return c
}

// jumpToLetter moves the cursor to the next entry (wrapping) whose name begins
// with r (lower-cased). getName returns the display name for entry index i.
// Searches forward from cursor+1. No-op if no match exists.
func (c listCursor) jumpToLetter(r rune, getName func(int) string, n int) listCursor {
	for i := 1; i < n; i++ {
		idx := (c.cursor + i) % n
		name := strings.ToLower(getName(idx))
		if len(name) > 0 && rune(name[0]) == r {
			c.cursor = idx
			return c.clampOffset()
		}
	}
	return c
}
