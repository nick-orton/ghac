# Theming (`ctrl+t`)

[issue #11](https://github.com/nick-orton/ghac/issues/11)

## Summary

ghac supports multiple color themes that are compiled into the binary.
Pressing `ctrl+t` opens a theme selector modal. Moving the cursor through
the list applies each theme in real time, so the user can preview before
committing. Pressing `Enter` confirms the selection and saves it for
future sessions; pressing `Esc` reverts to the previous theme and closes
the modal. The active theme is persisted via the XDG Base Directory
Specification and can also be pre-selected in `.ghacrc` or with the
`--theme` CLI flag.

---

## Scope

| Screen           | In scope |
| ---------------- | -------- |
| All screens      | Yes — theming affects the global color palette |
| Theme modal      | Yes — new modal overlay (like Help) |
| Config file      | Yes — optional `theme` field and `[[themes]]` custom theme blocks |
| CLI flag         | Yes — `--theme <name>` |
| State file       | Yes — XDG state persistence |

---

## Behaviour Specification

1. Pressing `ctrl+t` opens the theme selector modal as a centered overlay
   (same mechanism as the Help modal). The modal cannot be opened while
   the Help modal is already open, and vice versa.
2. The modal lists all themes by name — built-ins first, then any user
   themes from `.ghacrc` — one per line in definition order.
3. The cursor starts at the currently active theme.
4. Moving the cursor with `j` / `k` immediately applies the highlighted
   theme to the entire UI, giving a live preview.
5. Pressing `Enter` confirms the highlighted theme, saves the name to the
   XDG state file, and closes the modal. The selected theme remains
   active for the rest of the session.
6. Pressing `Esc` (or `ctrl+t` again) reverts the UI to the theme that
   was active when the modal was opened and closes the modal without
   saving.
7. A hint line at the bottom of the modal reads
   `[enter] confirm  [esc] cancel`. The modal is sized wide enough to
   display this line without clipping.
8. The theme is resolved on startup with the following priority (highest
   first):
   1. `--theme <name>` CLI flag
   2. `theme = "<name>"` in `.ghacrc`
   3. Saved name in the XDG state file
   4. Built-in default (`default`)
9. If an invalid or unknown theme name is given (via flag or config), the
   application falls back to the default theme and prints a warning to
   stderr.
10. The state file is stored at
    `$XDG_STATE_HOME/ghac/theme` (falling back to
    `$HOME/.local/state/ghac/theme`).

### Built-in Themes

| Name        | Character                     |
| ----------- | ----------------------------- |
| `default`   | Cyan accent, dark gray bar    |
| `ocean`     | Blue accent, navy bar         |
| `forest`    | Green accent, dark green bar  |
| `rose`      | Pink accent, dark red bar     |
| `mono`      | White accent, gray bar        |
| `vampire`   | Red accent, dark red bar      |
| `cyberpunk` | Magenta/yellow accent, navy bar |
| `matrix`    | Green accent, near-black bar  |

### Tab Strip

The theme modal does not appear in the tab strip. It is launched
exclusively via `ctrl+t` and indicated in the help overlay.

---

## Design Decisions (confirmed)

1. **Two-tier theme system** — built-in themes live in
   `internal/ui/themes.toml`, embedded at build time. User-defined themes
   are declared as `[[themes]]` blocks in `.ghacrc` and loaded at startup.
   User themes are appended after built-ins; both are selectable by name
   and visible in the theme selector.
2. **Package-level style vars are reassigned** — `applyTheme()` rewrites
   the affected style vars in `styles.go` in place. This is safe because
   all UI rendering happens on the Bubble Tea goroutine; no concurrency
   issue arises.
3. **Live preview via cursor movement** — applying the theme immediately
   on `j`/`k` is accomplished by calling `applyTheme()` directly from
   the theme modal's `Update()`.
4. **XDG state, not cache** — the theme name is user preference state,
   not derived data, so `$XDG_STATE_HOME` is the correct XDG directory.
5. **Silent save failure** — if the state file cannot be written (e.g.,
   read-only filesystem), the theme is still applied for the session; the
   error is silently discarded.
6. **`ctrl+t` toggles the modal** — pressing `ctrl+t` a second time
   while the modal is open behaves the same as `Esc` (reverts and
   closes).
7. **Modal width driven by hint line** — the modal is sized to fit
   `[enter] confirm  [esc] cancel` (the widest content), so the hint
   never wraps or gets clipped by the border.
8. **No tab strip entry** — unlike `?:Help`, the theme selector is not
   listed in the tab strip. It is documented only in the help overlay.

---
