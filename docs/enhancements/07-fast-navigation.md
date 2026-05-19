# Fast Navigation (`f<letter>`)

[issue #7](https://github.com/nick-orton/ghac/issues/7)

## Summary

Add a two-key jump shortcut to the **Playlist Control** and **Library
Navigator** screens. Pressing `f` followed by a letter moves the
cursor to the first entry whose display name starts with that letter
(case-insensitive).

---

## Scope

| Screen             | In scope |
| ------------------ | -------- |
| Playlist Control   | Yes      |
| Library Navigator  | Yes      |
| Player Volume      | No       |

---

## Behaviour Specification

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

### Display name used for matching

| Screen            | Match target                                         |
| ----------------- | ---------------------------------------------------- |
| Playlist Control  | `entryDisplayName()` — "Title – Artist" or filename |
| Library Navigator | `entry.Name` — filename or directory name            |

The navigator uses `entry.Name` because the filename is the
primary left-aligned identifier in the row. The song title (shown
right-aligned as metadata) is secondary context.

---

## Design Decisions (confirmed)

1. **Navigator match target** — `entry.Name` (filename/dirname).
2. **Search direction** — vim-style: search forward from `cursor+1`,
   wrapping to the top; no-op if no match exists anywhere in the list.
3. **Visual indicator** — none; silent, same as `gg` pending state.
4. **`ff` behaviour** — jumps to entries starting with `f`.

---
