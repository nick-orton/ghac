# Playlist Editing Confirmation

[issue #23](https://github.com/nick-orton/ghac/issues/23)

## Summary

When a bulk playlist edit would affect more than 50 songs, or when a
directory is being enqueued, ghac asks the user to confirm before
proceeding. The prompt shows the song count so the user knows the
scope of the operation.

---

## Scope

| Screen             | In scope |
| ------------------ | -------- |
| Playlist Control   | Yes      |
| Library Navigator  | Yes      |
| Player Volume      | No       |

---

## Behaviour Specification

### Confirmation threshold

| Operation                          | Triggers confirmation when…       |
| ---------------------------------- | --------------------------------- |
| Playlist `x` — remove selected     | removing > 50 songs               |
| Playlist `X` — clear entire queue  | always (no-op when playlist empty)|
| Navigator `x` — remove from queue  | removing > 50 playlist positions  |
| Navigator `enter` — enqueue files  | > 50 files selected               |
| Navigator `enter` — enqueue dir(s) | always (directory selected)       |

When the threshold is not exceeded the operation executes immediately
with no prompt, as today.

### Prompt format

An inline line appended at the bottom of the screen's content area
(inside the border box):

```
Remove 73 songs? [y/n]
Add 2 directories to playlist? [y/n]
```

For mixed file+directory selections the directory rule takes
precedence and the prompt always appears.

### Input handling

While the prompt is visible:

- `y` — execute the operation, clear the prompt.
- `n` or `esc` — cancel the operation, clear the prompt.
- All other keys (including global keys such as `p`, `z`, `1`/`2`/`3`)
  are swallowed. The screen returns `capturesAllInput() = true` so the
  key-handler chain delegates directly to the screen.
- `q` / `ctrl+c` — **not** swallowed; quit still works.

### Song count for the prompt message

- **Playlist `x`** — exact count of selected positions (or 1 for
  cursor song).
- **Playlist `X`** — total number of songs in the playlist.
- **Navigator `x`** — total playlist positions that would be deleted
  (a song queued multiple times counts once per occurrence).
- **Navigator `enter` files** — number of selected file entries.
- **Navigator `enter` directories** — number of selected directory
  entries (the prompt says "directories" not "songs").

---

## Design Decisions (confirmed)

1. **Inline prompt, not a modal overlay** — a single line inside the
   existing border box is lighter weight than a centered modal for a
   simple y/n question.
2. **Directories always prompt** — we do not recurse into directories
   to count songs; speed is preferred over precision for directory
   adds.
3. **`X` always prompts (except empty playlist)** — clearing the
   entire playlist is irreversible and always warrants a confirmation.
   Exception: when the playlist is already empty, `X` is a silent
   no-op (confirming "Clear all 0 songs?" is pointless).
4. **`q`/`ctrl+c` not swallowed** — the quit keys are handled before
   `capturesAllInput` in the key-handler chain and remain available.
5. **`capturesAllInput`** — reuse the existing mechanism (also used by
   the volume screen rename modal) to gate global keys during
   confirmation.
6. **Singular/plural in navigator prompt** — when exactly 1 directory
   is in the selection, the prompt reads "Add 1 directory to
   playlist? [y/n]" (singular). Two or more uses "directories".
7. **Prompt styled with `styleRowActive` (bold)** — visually distinct
   from regular rows without introducing a new color code, per the
   UX palette rules in `docs/ux.md`.

---
