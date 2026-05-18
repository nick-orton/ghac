=======
# Enhancements

## Change Song Position (`Ctrl+J` / `Ctrl+K`)

[issue #5](https://github.com/nick-orton/ghac/issues/5)

### Summary

Allows the user to reorder songs in the MPD playback queue directly
from the Playlist Control screen. Pressing `Ctrl+J` moves the song
under the cursor down one position; pressing `Ctrl+K` moves it up one
position. The cursor follows the moved song so that repeated presses
continue moving the same song without re-navigating.

---

### Scope

| Screen             | In scope |
| ------------------ | -------- |
| Player Volume      | No       |
| Playlist Control   | Yes      |
| Library Navigator  | No       |
| Help Modal         | No       |

---

### Behaviour Specification

1. The user presses `Ctrl+J` on a song in the Playlist Control screen.
2. The song moves down one position in the playlist (swaps with the
   song immediately below it).
3. The cursor moves down one position, staying on the moved song.
4. The user presses `Ctrl+K` on a song.
5. The song moves up one position (swaps with the song immediately
   above it).
6. The cursor moves up one position, staying on the moved song.
7. If the song is already at the bottom of the list, `Ctrl+J` is a
   no-op — the song and cursor do not move.
8. If the song is already at the top of the list, `Ctrl+K` is a
   no-op — the song and cursor do not move.
9. The move is sent to MPD using the `move` command, which
   repositions the song by its current queue index.

---

### Design Decisions (confirmed)

1. **Cursor follows the moved song** — repeated presses continue
   moving the same song without requiring the user to re-navigate,
   making multi-step reordering fluid.
2. **No-op at list boundaries** — pressing `Ctrl+J` at the last
   position or `Ctrl+K` at the first position produces no action and
   no error, consistent with how cursor movement behaves at boundaries
   elsewhere in the app.
3. **Operates on the cursor song regardless of selection state** —
   selection (`space`) marks songs for bulk removal; it has no meaning
   for positional moves. `Ctrl+J`/`Ctrl+K` always act on the single
   song under the cursor.

---

## Support Long Playlists (`ctrl+u` / `ctrl+d`)

[issue #6](https://github.com/nick-orton/ghac/issues/6)

### Summary

The playlist screen currently renders all entries without a viewport, which
means long playlists overflow the terminal. This feature adds half-page
scrolling to the playlist screen via `ctrl+u` (up half page) and `ctrl+d`
(down half page), matching the behaviour already present in the Library
Navigator screen.

---

### Scope

| Screen             | In scope |
| ------------------ | -------- |
| Playlist Control   | Yes      |
| Library Navigator  | No (already implemented) |
| Player Volume      | No       |

---

### Behaviour Specification

1. The playlist screen tracks a viewport `offset` (index of the first visible
   row) and a `height` (terminal height in rows).
2. `viewportHeight()` computes how many rows fit between the fixed chrome
   (now-playing bar, tab strip, border) and renders only that window.
3. `ctrl+d` advances the cursor by half the viewport height, clamping at the
   last entry.
4. `ctrl+u` retreats the cursor by half the viewport height, clamping at 0.
5. After every cursor move, `clampOffset()` adjusts the viewport so the
   cursor row is always visible.
6. The model forwards `tea.WindowSizeMsg` height to the playlist screen the
   same way it does for the navigator.
7. `View()` renders only the entries in the current viewport window.

#### Overhead line count

| Line              | Count |
| ----------------- | ----- |
| Now-playing bar   | 1     |
| Separator         | 1     |
| Tab strip         | 1     |
| Separator         | 1     |
| Border top        | 1     |
| Border bottom     | 1     |
| **Total overhead**| **6** |

---

### Design Decisions (confirmed)

1. **Match navigator implementation** — identical field names (`offset`,
   `height`), method signatures (`viewportHeight`, `clampOffset`, `withHeight`),
   and key bindings (`ctrl+u`, `ctrl+d`) keep the two screens consistent and
   make the code easy to follow.
2. **Default viewport before first WindowSizeMsg** — `viewportHeight()` returns
   24 when `height < 10`, matching the navigator's fallback.

---
