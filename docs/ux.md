# ghac — UX Standards

This document records the visual and interaction design standards for
the ghac TUI. Follow these guidelines when implementing new screens,
components, or styling changes.

## 1. Color Palette

ghac uses 256-color terminal codes via lipgloss. The canonical values
are defined in `internal/ui/styles.go` as package-level `var` declarations
and must not be duplicated elsewhere — reference the style variables
directly. At runtime, `applyTheme()` in `theme.go` reassigns the
color-bearing vars to match the active theme; the values below are for
the built-in `default` theme.

| Role               | Color code | Style variable              | Usage                                  |
| ------------------ | ---------- | --------------------------- | -------------------------------------- |
| Bar background     | `237`      | `styleNowPlaying`           | Now-playing bar background (dark gray) |
| Bar foreground     | `255`      | `styleNowPlaying`           | Now-playing bar text (near-white)      |
| Accent / progress  | `6`        | `styleProgressFill`         | Progress bar filled portion (cyan)     |
| Progress empty     | `240`      | `styleProgressEmpty`        | Progress bar unfilled portion          |
| Secondary text     | `245`      | `styleTime`, `styleNavMeta` | Time display, navigator metadata       |
| Volume bar unmuted | `2`        | `styleVolumeBarFillUnmuted` | Player Volume bar fill when unmuted (green) |
| Volume bar muted   | `1`        | `styleVolumeBarFillMuted`   | Player Volume bar fill when muted (red)     |
| Current song       | `6`        | `stylePlaylistCurrent`      | Playlist Control currently-playing row (bold cyan) |

For elements that do not need a specific color, use lipgloss modifiers
(`Bold`, `Faint`, `Underline`, `Italic`) rather than introducing new
color codes. New color codes require a deliberate decision and a doc
update here.

**Style variables without a color code (modifier-only):**

| Variable           | Modifiers         | Usage                                     |
| ------------------ | ----------------- | ----------------------------------------- |
| `styleTitle`       | Bold              | Screen border title text                  |
| `stylePlaceholder` | Italic + Faint    | Empty state messages, breadcrumb line     |
| `styleTabActive`   | Bold + Underline  | Active tab in tab strip                   |
| `styleTabInactive` | Faint             | Inactive tabs in tab strip                |
| `styleHelpSection` | Bold + Underline  | Help screen section headers               |
| `styleHelpKey`     | Bold              | Help screen key column                    |
| `styleHelpDesc`    | Faint             | Help screen description column            |
| `styleRowActive`   | Bold              | Cursor-highlighted row (all screens)      |
| `styleNavDir`      | Bold              | Directory names in navigator              |

## 2. Global Layout

The screen is composed top-to-bottom in `Model.View()`:

```text
 Now-playing bar  (always visible, full terminal width)
 Tab strip        (always visible)
┌─ Screen Name ──────────────────────────────────────────┐
│ Active screen content                                  │
└────────────────────────────────────────────────────────┘
```

The now-playing bar and tab strip are separated from each other and
from the screen border by a single `\n`. Do not add extra blank lines
between zones — vertical space is scarce in a TUI.

## 3. Now-Playing Bar

The bar spans the full terminal width and is rendered by
`NowPlayingView()` in `internal/ui/nowplaying.go`.

**Content order (left to right):**
random indicator (when on) → state icon → song name → progress
bar → elapsed / total time

**Styling:**

- Background `237`, foreground `255`, bold — applied to the whole bar
  as a single `styleNowPlaying.Render()` call.
- Nested styles inside the bar must be compatible with the dark
  background; avoid `Faint` (invisible on dark bg) and `Reverse`
  (double-inverts). Use explicit foreground colors instead.

**Random indicator:**

- When random mode is on: `[Z] ` (4 characters including
  trailing space) prepended before the state icon.
- When random mode is off: nothing — no space is reserved.

**State icons:**

- Playing: `▶`
- Paused: `⏸`
- Stopped / idle: show placeholder `[ No song playing ]`

**Progress bar characters:**
- Filled: `█` styled with the accent color (`6`)
- Unfilled: `░` styled with color `240`
- Fixed width: 20 characters

**Degraded-width behaviour:** when the terminal is too narrow to fit
the progress bar and time, omit them and show only the icon and a
truncated song name. Never panic or produce garbled output at any
terminal width.

**Song name format:** `Title – Artist – Album` (en-dash `–`, U+2013).
Degrade gracefully: `Title – Artist`, then `Title`, then filename.

## 4. Tab Strip

The tab strip is rendered by `Model.tabStripView()` in
`internal/ui/model.go`. It shows every screen at all times so the
user always knows both where they are and how to navigate.

**Tabs (in order):** `1:Volume`, `2:Playlist`, `3:Library`, `?:Help`

**Styling:**
- Active tab: `styleTabActive` — bold + underline.
- Inactive tabs: `styleTabInactive` — faint.
- Separator between tabs: two spaces.

When adding a new named screen, add a corresponding tab entry in
`tabStripView()` and document its keybinding in `help.go`.

## 5. Screen Titles

The screen title is rendered by `screenBorder()` in `model.go`,
embedded in the top edge of the border:

```text
┌─ Player Volume ────────────────────────────────────────┐
```

The title is styled with `styleTitle` (bold). Screen `View()` methods
must **not** render their own title — they return content only,
starting directly with the first content element (no leading blank
line). `screenBorder()` is the single place to update title styling.

## 6. Player Volume Screen

Each client row follows this layout:

```text
▶ ClientName            ████████████████░░░░  74%  [M]
  ClientName            ████████████████░░░░  74%
```

- Cursor indicator: `▶ ` (2 chars) or `  ` (2 spaces).
- Client name: left-aligned, padded to 20 characters. Truncated with
  `…` if longer.
- Volume bar: 20 characters fixed width. Green (`2`) when unmuted,
  red (`1`) when muted. Unfilled portion uses color `240`.
- Percentage: right-aligned, 3 digits + `%`.
- Mute indicator: `[M]` when muted, 3 spaces otherwise.
- Cursor row is styled with `styleRowActive` (bold).

### 6.1 Rename Modal

Pressing `Ctrl-R` opens a rename modal centered over the Player
Volume screen. The modal is composited via `placeOverlay()` /
`modalBorder()` in `model.go`, the same infrastructure used by the
help modal.

**Modal sizing:**
- Width: `min(50, terminalWidth − 4)`, with a floor of 30.
- Height: sized to content (name field + hint line) plus 2 border
  lines.
- Position: centered horizontally and vertically.

**Layout:**

```text
┌─ Rename Client ──────────────────┐
│                                  │
│  Name: ClientName_               │
│                                  │
│  Ctrl-S: save   Esc: cancel      │
└──────────────────────────────────┘
```

- The `Name:` label is followed by the current input buffer. A `_`
  character is rendered at the cursor position to indicate the
  insertion point.
- The hint line uses `styleHelpDesc` (faint) to keep it visually
  subordinate to the input field.

**Interaction:** while the modal is open, all keys other than
text-editing, `Ctrl-S`, and `Esc` are swallowed. `Ctrl-R` is a
no-op when no clients are connected.

## 7. Bulk-Edit Confirmation Prompt

When a bulk playlist edit exceeds the confirmation threshold (more than
50 songs affected, or any directory being enqueued), an inline prompt
appears as the last line of the screen's content area, inside the
border box:

```text
Remove 73 songs? [y/n]
Add 2 directories to playlist? [y/n]
Clear all 120 songs? [y/n]
```

**Styling:** `styleRowActive` (bold, no color) — visually distinct
from regular list rows without introducing a new color code.

**Interaction:** `y` executes the operation; `n` or `Esc` cancels.
All other keys are swallowed while the prompt is visible (the screen
returns `capturesAllInput() = true`). `q` and `Ctrl-C` are not
swallowed because `handleQuit` fires before `capturesAllInput` in the
key-handler chain.

**No modal overlay** — the prompt is a single line appended to the
screen's `View()` output, not a centered overlay. This keeps the
interaction lightweight for simple y/n decisions.

## 8. Playlist Control Screen

Each song row follows this prefix layout (5 characters):

```text
▶ >*  →  cursor(2) + playing(1) + selected(1) + space(1)
```

- Cursor: `▶ ` or `  `
- Playing: `>` for the currently-playing song, ` ` otherwise
- Selected: `*` for selected songs, ` ` otherwise
- Then the song display name: "Title – Artist" or filename fallback

**Row styling:**
- Cursor row: `styleRowActive` (bold)
- Playing row (not cursor): `stylePlaylistCurrent` (bold + cyan `6`)
- Normal row: unstyled

**Empty state:** `stylePlaceholder` renders "Playlist is empty".

### 7.3 Viewport Scrolling

The playlist implements vertical viewport scrolling. The viewport
height is calculated as `terminal_height - 6` (overhead:
nowplaying, separator, tabstrip, separator, border top, border
bottom). Before the first `WindowSizeMsg` arrives, a default of
24 rows is used.

The viewport offset auto-adjusts so the cursor is always visible:
- If the cursor moves above the viewport, the offset snaps to the
  cursor position.
- If the cursor moves below the viewport, the offset advances so
  the cursor is the last visible row.

`Ctrl-D` and `Ctrl-U` move the cursor by half the viewport height.
Fast-jump (`f<letter>`) and song removal (`x`) also trigger
`clampOffset()` so the viewport is always correct after those
operations.

## 9. Library Navigator Screen

### 8.1 Breadcrumb Line

The first line of content is a breadcrumb rendered with
`stylePlaceholder`:

- Root: `Path: / (root)`
- Subdirectory: `Path: Artists/Pink Floyd`

### 8.2 Entry Row Layout

Each entry row follows this prefix layout (5 characters):

```text
cursor(2) + selected(1) + queued(1) + space(1)
```

- Cursor: `▶ ` or `  `
- Selected: `*` or ` `
- Queued (files only): `+` if the file's MPD URI is in the current
  playlist, ` ` otherwise. Directories always show ` `.
- Then a space separator before the entry name.

### 8.3 Directory Rows

Directory names are rendered with `styleNavDir` (bold) and a
trailing `/`:

```text
▶ *  Albums/
```

### 8.4 File Rows

Files show the filename left-aligned. When the terminal is wide
enough (gap >= 2 chars between name and metadata), metadata
("Title – Artist") is right-aligned on the same line, styled with
`styleNavMeta` (color `245`):

```text
   +  track01.flac              Dark Side of the Moon – Pink Floyd
```

The available width for name + metadata is calculated as:
`terminal_width - 9` (4 for border padding + 5 for prefix).

When terminal width is insufficient for right-aligned metadata,
only the filename is shown.

### 8.5 Viewport Scrolling

The navigator implements vertical viewport scrolling. The viewport
height is calculated as `terminal_height - 7` (overhead: nowplaying,
separator, tabstrip, separator, border top, breadcrumb, border
bottom). Before the first `WindowSizeMsg` arrives, a default of 24
rows is used.

The viewport offset auto-adjusts so the cursor is always visible:
- If the cursor moves above the viewport, the offset snaps to the
  cursor position.
- If the cursor moves below the viewport, the offset advances so
  the cursor is the last visible row.

`Ctrl-D` and `Ctrl-U` move the cursor by half the viewport height.

### 8.6 Cursor Row Styling

The cursor row is styled with `styleRowActive` (bold), applied to
the entire row string.

**Empty state:** `stylePlaceholder` renders "Directory is empty".

## 10. Help Modal

Key bindings are displayed in `internal/ui/help.go` organized into
named sections. Help appears as a centered modal overlay; the
underlying screen remains visible around it.

**Modal sizing:**
- Width: `min(82, terminalWidth − 4)` — wide enough for the longest
  help row (12-char key column + 2 spaces + ~62-char description).
- Height: sized to content plus 2 border lines.
- Position: centered horizontally and vertically. Calculated as
  `x = (width − modalWidth) / 2`, `y = (height − modalLines) / 2`.

**Modal border:** rendered by `modalBorder()` in `model.go`. Uses the
same box-drawing characters and title-in-top-edge style as
`screenBorder()`. No new style variables are needed.

**Content column alignment:** the key column is fixed-width (pad with
spaces so descriptions align). Keep consistent across all sections.
Current width: 12 characters (see `helpRow()`).

**Styling:**
- Section headers: `styleHelpSection` — bold + underline.
- Keys: `styleHelpKey` — bold.
- Descriptions: `styleHelpDesc` — faint.

**Sections (in order):**
1. Global
2. Player Volume
3. Playlist Control
4. Library Navigator

When adding a new keybinding, add it to the correct section in
`help.go` and keep the sections sorted by screen (global first, then
per-screen in tab order). The Player Volume section includes
`Ctrl-R` for the rename modal.

**Tab strip:** `?:Help` remains in the tab strip as a permanent
keybinding hint but is never rendered as active. Help is a modal
overlay, not a peer screen.

## 11. Typography Conventions

| Element           | Characters to use                              |
| ----------------- | ---------------------------------------------- |
| Field separator   | ` – ` (en-dash with spaces, U+2013)            |
| Ellipsis          | `…` (U+2026, single character)                 |
| Progress filled   | `█` (U+2588 FULL BLOCK)                        |
| Progress unfilled | `░` (U+2591 LIGHT SHADE)                       |
| Horizontal rule   | `─` (U+2500 BOX DRAWINGS LIGHT HORIZONTAL)     |
| Border corners    | `┌` `┐` `└` `┘` (U+250C/10/14/18)             |
| Border sides      | `│` (U+2502 BOX DRAWINGS LIGHT VERTICAL)       |
| Cursor indicator  | `▶` (U+25B6 BLACK RIGHT-POINTING TRIANGLE)     |

Do not use ASCII hyphens (`-`) as separators in displayed text, or
`...` (three periods) as an ellipsis. The above Unicode characters
render correctly on any UTF-8 terminal, which is the minimum
requirement for running ghac.

## 12. Terminal Compatibility Assumptions

- UTF-8 encoding: required.
- 256-color support: required (color codes above 15 are used).
- True-color (24-bit): not assumed; do not use hex color strings.
- Minimum width: 80 columns (components default to this when
  `width == 0` or `width < 4`).
- Alt-screen mode: always active; the TUI owns the full display.

## 13. Adding a New Screen

1. Create `internal/ui/<name>.go` with a struct implementing
   `Update(tea.Msg) (<ScreenType>, tea.Cmd)` and `View() string`.
   `View()` returns content only — no title, no leading blank line.
2. Add a `screenID` constant in `model.go`.
3. Add a key binding in `model.go`'s `Update()`.
4. Wire the screen into `delegateToActiveScreen()` and `View()`,
   supplying the screen's title string to `screenBorder()`.
5. Add a tab entry in `tabStripView()`.
6. Document the screen's keybindings in `help.go`.
7. Update `docs/architecture.md` to describe the new screen's
   responsibilities.
