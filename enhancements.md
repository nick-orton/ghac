# Enhancements

## Fast Navigation (`f<letter>`)

[issue #7](https://github.com/nick-orton/ghac/issues/7)

### Summary

Add a two-key jump shortcut to the **Playlist Control** and **Library
Navigator** screens. Pressing `f` followed by a letter moves the
cursor to the first entry whose display name starts with that letter
(case-insensitive).

---

### Scope

| Screen             | In scope |
| ------------------ | -------- |
| Playlist Control   | Yes      |
| Library Navigator  | Yes      |
| Player Volume      | No       |

---

### Behaviour Specification

1. Press `f` — the screen enters a "pending-f" state; the next
   keystroke is consumed as the search target.
2. Press a letter (`a`–`z`, `A`–`Z`) — the cursor jumps to the
   **first** entry in the list whose display name begins with that
   letter (case-insensitive). If no match exists, the cursor does
   not move.
3. Press any non-letter key after `f` — pending-f is cancelled; the
   key is **not** processed further (no cursor movement, no other
   action).
4. The jump always searches from the **top of the list** (index 0),
   not from the current cursor position. The issue wording "First
   instance" implies list-start semantics.
5. After the jump (match or no-match) the viewport is scrolled so the
   cursor is visible (`clampOffset()` in the navigator).

#### Display name used for matching

| Screen            | Match target                                         |
| ----------------- | ---------------------------------------------------- |
| Playlist Control  | `entryDisplayName()` — "Title – Artist" or filename |
| Library Navigator | `entry.Name` — filename or directory name            |

The navigator uses `entry.Name` because the filename is the
primary left-aligned identifier in the row. The song title (shown
right-aligned as metadata) is secondary context.

---

### Design Decisions (confirmed)

1. **Navigator match target** — `entry.Name` (filename/dirname).
2. **Search direction** — vim-style: search forward from `cursor+1`,
   wrapping to the top; no-op if no match exists anywhere in the list.
3. **Visual indicator** — none; silent, same as `gg` pending state.
4. **`ff` behaviour** — jumps to entries starting with `f`.

---

## Modal Help Overlay (`?`)

[issue #9](https://github.com/nick-orton/ghac/issues/9)

### Summary

Replace the full-screen Help screen with a modal overlay that floats
on top of the currently active screen. The underlying screen content
remains visible around the edges of the modal.

---

### Scope

| Area                          | Change                                             |
| ----------------------------- | -------------------------------------------------- |
| Root model (`model.go`)       | `screenHelp` → `showHelp bool`; remove `prevScreen`|
| View composition (`model.go`) | custom `placeOverlay()` composites modal over bg   |
| Tab strip (`model.go`)        | `?:Help` tab stays visible but never active        |
| Modal border (`model.go`)     | New `modalBorder()` mirrors `screenBorder()` style |
| Help content (`help.go`)      | No change to content or `helpRow()` rendering      |

---

### Behaviour Specification

1. Press `?` — the help modal appears centered over the current
   screen. The current screen's tab remains highlighted; `?:Help`
   is never shown as active.
2. The underlying screen (now-playing bar, tab strip, screen
   border and content) remains visible around the modal.
3. While the modal is open, only `q`, `Ctrl-C`, `Esc`, and `?`
   are processed. All other key events are swallowed. Backend
   messages (`MsgPlayerState`, `MsgTick`, etc.) and
   `WindowSizeMsg` continue to be processed normally.
4. Press `Esc` or `?` again — the modal closes and the underlying
   screen is fully restored.
5. `?` is a toggle: pressing it while the modal is open closes it.

#### Modal sizing

- **Width:** `min(82, terminalWidth − 4)` — gives a comfortable
  margin on narrow terminals while capping at 82 characters (wide
  enough to display the longest help row without truncation).
- **Height:** sized to the rendered content plus 2 border lines.
- **Position:** centered horizontally and vertically using
  custom `placeOverlay(x, y, modal, background)` in `model.go`
  where `x = (width − modalWidth) / 2` and
  `y = (height − modalLines) / 2`.
- **Minimum width floor:** 20 characters (defensive guard for
  very narrow terminals).

#### State model change

| Before                        | After                                   |
| ----------------------------- | --------------------------------------- |
| `activeScreen screenID` (4 values incl. `screenHelp`) | `activeScreen screenID` (3 values only) |
| `prevScreen screenID`         | removed                                 |
| —                             | `showHelp bool`                         |

---

### Design Decisions (confirmed)

1. **`screenHelp` removed** — help is no longer a peer screen;
   it is a transient overlay. `activeScreen` never holds a
   "help" value.
2. **`prevScreen` removed** — not needed; `activeScreen` never
   changes when help opens, so returning is a no-op.
3. **Tab `?:Help` kept** — serves as a permanent keybinding
   reminder; never rendered as the active tab.
4. **`modalBorder()`** — a new function mirroring `screenBorder()`
   but operating at modal width; keeps the same box-drawing
   characters and title-in-top-edge style.
5. **Key swallowing** — while the modal is open, non-global keys
   are silently discarded so the underlying screen does not
   accidentally mutate state.

---
