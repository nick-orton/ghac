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
