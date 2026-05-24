# Legacy Terminal Support (`--legacy`)

[issue #29](https://github.com/nick-orton/ghac/issues/29)

## Summary

On BSD systems running without a window manager, ghac may be used on a
legacy terminal (VT220, FreeBSD console, etc.) that cannot render Unicode
block-drawing or box-drawing characters, displaying replacement glyphs
instead. ghac should detect this automatically via `$TERM` and switch to
a rendering mode that looks correct on such terminals: ASCII-only symbols,
no box-drawing borders, and a black-and-white (no color) style. A
`--legacy` CLI flag allows the user to force this mode when auto-detection
misses.

---

## Scope

| Screen             | In scope |
| ------------------ | -------- |
| Now-playing bar    | Yes      |
| Tab strip          | Yes      |
| Screen border      | Yes      |
| Player Volume      | Yes      |
| Playlist Control   | Yes      |
| Library Navigator  | Yes      |
| Theme modal        | Yes      |
| Help modal         | Yes (full-screen in legacy mode, no border) |
| Theme selector     | No (legacy mode is not a selectable theme) |

---

## Behaviour Specification

1. At startup, ghac checks whether the terminal is legacy by inspecting
   the `$TERM` environment variable. The following values (exact match or
   prefix match before a `-`) trigger legacy mode automatically:
   `vt220`, `vt100`, `vt102`, `vt52`, `ansi`, `dumb`, `cons25`,
   `wsvt25`, `cygwin`.
2. The `--legacy` flag forces legacy mode regardless of `$TERM`.
3. When legacy mode is active, all Unicode symbols are replaced with
   ASCII equivalents (see Symbol Map below).
4. Box-drawing borders are removed entirely. The screen area is rendered
   with a plain title header line and no surrounding box:
   ```
   -- Player Volume --
    content here
   ```
5. A black-and-white style is applied: all color codes are removed; only
   bold, faint, underline, and reverse-video attributes are used for
   visual distinction.
6. The now-playing bar uses reverse-video (`Reverse(true)`) instead of a
   background color to remain visually distinct.
7. In legacy mode, the help overlay is rendered full-screen (now-playing
   bar + tab strip + `-- Help --` header + content) rather than as a
   centered modal with box-drawing borders.
8. Legacy mode is never selectable via the theme modal and does not appear
   in the theme list. It is an independent rendering mode, not a theme.
9. In legacy mode, the `ctrl+t` theme modal still opens but applies
   colors only if the active theme is later confirmed on a capable
   terminal — the legacy style overrides any selected theme's colors
   while active.

### Symbol Map

| Element              | Default (Unicode) | Legacy (ASCII) |
| -------------------- | ----------------- | -------------- |
| Cursor indicator     | `▶ `              | `> `           |
| Play state icon      | `▶`               | `>`            |
| Pause state icon     | `⏸`               | `\|`           |
| Progress bar filled  | `█`               | `#`            |
| Progress bar empty   | `░`               | `-`            |
| Ellipsis             | `…`               | `.`            |
| Field separator      | ` – ` (en-dash)   | ` - `          |
| Border top-left      | `┌`               | (removed)      |
| Border top-right     | `┐`               | (removed)      |
| Border bottom-left   | `└`               | (removed)      |
| Border bottom-right  | `┘`               | (removed)      |
| Border horizontal    | `─`               | (removed)      |
| Border vertical      | `│`               | (removed)      |

### Legacy Screen Layout

```
Artist - Title              ################----  2:34 / 5:01
1:Volume  2:Playlist  3:Library  ?:Help
-- Player Volume --
> Client1     ####################  74%  [M]
  Client2     ####################  100%
```

The `-- Title --` header uses `styleTitle` (bold). No leading blank line.
Content immediately follows the title line.

---

## Design Decisions (confirmed)

1. **No color in legacy mode** — VT220 and similar terminals may not
   support color at all. All color styling is stripped; reverse-video
   provides bar distinction.
2. **Borders excluded, not replaced** — ASCII box corners (`+`) still
   look bad with dashed lines at varying widths. A plain title line is
   cleaner and requires no width math changes.
3. **Not a theme** — Legacy mode is a rendering-mode flag, not a named
   theme, so it cannot be accidentally selected or overwritten by the
   theme selector. `applyLegacyTheme()` is a separate function from
   `applyTheme()`.
4. **`--legacy` flag for missed detection** — Some terminals report a
   capable `$TERM` (e.g., `xterm`) but are actually incapable; the flag
   covers that case without requiring config-file changes.
5. **Ellipsis stays one character** — ASCII ellipsis is `.` (not `...`)
   so truncation width arithmetic is unchanged.
