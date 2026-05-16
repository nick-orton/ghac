# ghac — UX Standards

This document records the visual and interaction design standards for
the ghac TUI. Follow these guidelines when implementing new screens,
components, or styling changes.

## 1. Color Palette

ghac uses 256-color terminal codes via lipgloss. The canonical values
are defined in `internal/ui/styles.go` and must not be duplicated
elsewhere — reference the style variables directly.

| Role              | Color code | Usage                                  |
| ----------------- | ---------- | -------------------------------------- |
| Bar background    | `237`      | Now-playing bar background (dark gray) |
| Bar foreground    | `255`      | Now-playing bar text (near-white)      |
| Accent / progress | `6`        | Progress bar filled portion (cyan)     |
| Progress empty    | `240`      | Progress bar unfilled portion          |
| Secondary text    | `245`      | Time display, de-emphasized info       |

For elements that do not need a specific color, use lipgloss modifiers
(`Bold`, `Faint`, `Underline`, `Italic`) rather than introducing new
color codes. New color codes require a deliberate decision and a doc
update here.

## 2. Global Layout

The screen is composed top-to-bottom in `Model.View()`:

```text
┌────────────────────────────────────────────────────────┐
│ Now-playing bar  (always visible, full terminal width) │
├────────────────────────────────────────────────────────┤
│ Tab strip        (always visible)                      │
├────────────────────────────────────────────────────────┤
│ Active screen content                                  │
└────────────────────────────────────────────────────────┘
```

Each zone is separated by a single `\n`. Do not add extra blank lines
between zones — vertical space is scarce in a TUI.

## 3. Now-Playing Bar

The bar spans the full terminal width and is rendered by
`NowPlayingView()` in `internal/ui/nowplaying.go`.

**Content order (left to right):**
state icon → song name → progress bar → elapsed / total time

**Styling:**

- Background `237`, foreground `255`, bold — applied to the whole bar
  as a single `styleNowPlaying.Render()` call.
- Nested styles inside the bar must be compatible with the dark
  background; avoid `Faint` (invisible on dark bg) and `Reverse`
  (double-inverts). Use explicit foreground colors instead.

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

**Tabs (in order):** `1:Volume`, `2:Playlist`, `3:Navigator`, `?:Help`

**Styling:**
- Active tab: `styleTabActive` — bold + underline.
- Inactive tabs: `styleTabInactive` — faint.
- Separator between tabs: two spaces.

When adding a new named screen, add a corresponding tab entry in
`tabStripView()` and document its keybinding in `help.go`.

## 5. Screen Titles

Each screen renders its own title as the first line of `View()`,
using `styleTitle` (bold). No additional decoration is required at
this time, but a separator rule may be added in the future — keep
title rendering in the screen's own `View()` so it can be updated
in one place per screen.

## 6. Help Screen

Key bindings are displayed in `internal/ui/help.go` organized into
named sections.

**Column alignment:** the key column is fixed-width (pad with spaces
so descriptions align). Keep the key column width consistent across
all sections. Current width: 12 characters (see `helpRow()`).

**Styling:**
- Section headers: `styleHelpSection` — bold + underline.
- Keys: `styleHelpKey` — bold.
- Descriptions: `styleHelpDesc` — faint.

When adding a new keybinding, add it to the correct section in
`help.go` and keep the sections sorted by screen (global first, then
per-screen in tab order).

## 7. Typography Conventions

| Element           | Characters to use                              |
| ----------------- | ---------------------------------------------- |
| Field separator   | ` – ` (en-dash with spaces, U+2013)            |
| Ellipsis          | `…` (U+2026, single character)                 |
| Progress filled   | `█` (U+2588 FULL BLOCK)                        |
| Progress unfilled | `░` (U+2591 LIGHT SHADE)                       |
| Horizontal rule   | `─` (U+2500 BOX DRAWINGS LIGHT HORIZONTAL)     |

Do not use ASCII hyphens (`-`) as separators in displayed text, or
`...` (three periods) as an ellipsis. The above Unicode characters
render correctly on any UTF-8 terminal, which is the minimum
requirement for running ghac.

## 8. Terminal Compatibility Assumptions

- UTF-8 encoding: required.
- 256-color support: required (color codes above 15 are used).
- True-color (24-bit): not assumed; do not use hex color strings.
- Minimum width: 80 columns (components default to this when
  `width == 0`).
- Alt-screen mode: always active; the TUI owns the full display.

## 9. Adding a New Screen

1. Create `internal/ui/<name>.go` with a struct implementing
   `Update(tea.Msg) (<ScreenType>, tea.Cmd)` and `View() string`.
2. Add a `screenID` constant in `model.go`.
3. Add a key binding in `model.go`'s `Update()`.
4. Wire the screen into `delegateToActiveScreen()` and `View()`.
5. Add a tab entry in `tabStripView()`.
6. Document the screen's keybindings in `help.go`.
7. Update `docs/architecture.md` to describe the new screen's
   responsibilities.
