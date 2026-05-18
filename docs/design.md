# ghac — Great Home Audio Controller: Product Design

## 1. Purpose

ghac is a terminal user interface (TUI) written in Go for
controlling a multi-room home audio system. It integrates two
backend services:

- **Music Player Daemon (MPD)** — music library browsing,
  playlist management, and playback control.
- **SnapCast** — synchronized multi-room audio streaming and
  per-client volume control.

The application targets users who prefer keyboard-driven,
vim-style interfaces and want a single pane of glass for their
home audio system.

## 2. Configuration

### 2.1 Config File

- Location: `$HOME/.config/.ghacrc`
- Format: TOML

### 2.2 Required Fields

| Field              | Description                    |
| ------------------ | ------------------------------ |
| `snapserver.ip`    | SnapCast server IP address     |
| `snapserver.port`  | SnapCast server port           |
| `mpd.ip`           | Music Player Daemon IP address |
| `mpd.port`         | Music Player Daemon port       |

### 2.3 Startup Behavior

On launch, ghac reads the config file and attempts to connect
to both the SnapCast server and the MPD server. If either
connection fails, the application prints a descriptive error
message to stderr and exits with a non-zero status code.

## 3. Global UI Elements

### 3.1 Now-Playing Bar

A persistent bar appears at the top of every screen. It
displays:

- **Random indicator** — `[Z]` prefix shown when MPD random
  (shuffle) mode is active; absent when random is off.
- **Song information** — metadata fields (title, artist,
  album) when available via MPD; falls back to the filename
  when metadata is absent.
- **Progress indicator** — a visual progress bar showing the
  proportion of the song elapsed, accompanied by a time
  display in `elapsed / total` format (e.g., `2:34 / 5:01`).

The now-playing bar updates in real time as the song
progresses.

### 3.2 Tab Strip

Below the now-playing bar, a persistent tab strip lists every
screen at all times:

```
1:Volume  2:Playlist  3:Library  ?:Help
```

The active screen's tab is highlighted (bold + underline);
inactive tabs are dimmed. This gives the user a permanent
reminder of both the current location and how to navigate.

### 3.3 Screen Title and Border

Each screen is wrapped in a single-line border with the screen
name embedded in the top edge:

```
┌─ Player Volume ──────────────────────────────────────────┐
│ screen content                                           │
└──────────────────────────────────────────────────────────┘
```

The title is rendered bold. This is produced by `screenBorder()`
in `model.go`; screen `View()` methods return content only.

### 3.4 Global Keybindings

These keybindings are active on every screen:

| Key      | Action                              |
| -------- | ----------------------------------- |
| `1`      | Switch to Player Volume screen      |
| `2`      | Switch to Playlist Control screen   |
| `3`      | Switch to Library Navigator screen     |
| `?`      | Open the Help screen                |
| `p`      | Toggle play / pause                 |
| `z`      | Toggle random (shuffle) mode        |
| `q`      | Quit the application                |
| `Ctrl-C` | Quit the application                |

## 4. Screens

### 4.1 Player Volume (Screen 1)

Controls the volume and mute state of every SnapCast client
connected to the server.

Each SnapCast client occupies one row:

- **Client name** — left-aligned, using the name reported by
  the SnapCast server (falls back to hostname if no configured
  name). Truncated to 20 characters with an ellipsis if longer.
- **Volume bar** — a horizontal bar graph to the right of the
  name, representing the current volume as a percentage
  (0–100). Fixed width of 20 characters.
- **Mute indicator** — when a client is muted, the volume bar
  is rendered in red (unmuted clients use green) and a `[M]`
  symbol appears to the right of the percentage.

The row under the cursor is highlighted bold to indicate focus.

**Player Volume keybindings:**

| Key | Action                                  |
| --- | --------------------------------------- |
| `j` | Move cursor down one client             |
| `k` | Move cursor up one client               |
| `h` | Decrease volume of focused client by 5% |
| `l` | Increase volume of focused client by 5% |
| `m` | Toggle mute on focused client           |
| `H` | Decrease volume of all clients by 5%    |
| `L` | Increase volume of all clients by 5%    |
| `M` | Toggle mute on all clients              |

Volume values clamp to the 0–100 range. A decrease below 0
stays at 0; an increase above 100 stays at 100.

Volume and mute changes made by other controllers (other ghac
instances, SnapCast's own web UI, etc.) are reflected in the
display in real time. ghac subscribes to the SnapCast server's
event stream for this purpose.

### 4.2 Playlist Control (Screen 2)

Views and manages the MPD playback queue.

The playlist is shown as a vertical list, one song per line.
Each line shows song metadata ("Title – Artist") when available,
falling back to the filename. The currently-playing song is
visually distinguished with a `>` prefix marker and bold cyan
styling. The song under the cursor is highlighted bold.

Songs can be individually toggled into a "selected" state with
`space`. Selected songs are marked with a `*` prefix character.
The selection is independent of the cursor position — the cursor
can move freely after selecting songs.

**Prefix layout** (5 characters before the song text):

```
▶ >*  →  cursor + playing + selected
  >   →  playing only
   *  →  selected only
      →  none
```

**Playlist Control keybindings:**

| Key          | Action                                         |
| ------------ | ---------------------------------------------- |
| `j`          | Move cursor down one song                      |
| `k`          | Move cursor up one song                        |
| `gg`         | Move cursor to the first song                  |
| `G`          | Move cursor to the last song                   |
| `Ctrl-D`     | Move cursor down half a page                   |
| `Ctrl-U`     | Move cursor up half a page                     |
| `Ctrl-J`     | Move song under cursor down one position       |
| `Ctrl-K`     | Move song under cursor up one position         |
| `f <letter>` | Jump to next song starting with letter (wraps) |
| `space`      | Toggle selection on the song under cursor      |
| `x`          | Remove selected song(s) from the playlist      |
| `X`          | Clear the entire playlist                      |
| `enter`      | Start playing the song under cursor            |

**Viewport scrolling:**

The playlist implements viewport scrolling. Only as many entries
as fit on screen are rendered at once. The viewport offset
adjusts automatically as the cursor moves so the cursor is
always visible. `Ctrl-D` and `Ctrl-U` move the cursor by half
the viewport height.

**Fast-jump behavior (`f <letter>`):**

- `f` followed by a letter (case-insensitive) moves the cursor
  to the next song whose display name ("Title – Artist" or
  filename) begins with that letter.
- The search starts from the entry after the cursor and wraps
  around to the top of the list if no match is found before
  the end.
- If no song in the list starts with the letter, the cursor
  does not move.
- `f` followed by a non-letter key cancels the sequence with
  no action.
- After a successful jump, the viewport scrolls so the cursor
  remains visible.

**Removal behavior:**

- If one or more songs are selected (via `space`), `x`
  removes all selected songs.
- If no songs are selected, `x` removes the song under the
  cursor.
- After removal, the cursor stays at its current index; if
  that index now exceeds the list end, it moves to the new
  last entry.
- `X` clears the entire playlist and stops playback.

### 4.3 Library Navigator (Screen 3)

Browses the MPD music library using its directory structure
and adds songs or directories to the playlist.

The navigator presents a directory listing styled to
approximate a tab in the `nnn` file manager:

- **Directories** are rendered bold with a trailing `/`.
- **Files** display the filename left-aligned and metadata
  ("Title – Artist") right-aligned when the terminal is wide
  enough. When there is insufficient space for right-aligned
  metadata, only the filename is shown.
- The entry under the cursor is highlighted bold.
- A breadcrumb line at the top shows the current directory
  path (e.g., `Path: Artists/Pink Floyd`) or `Path: / (root)`
  when at the library root.
- Files that are already in the playback queue are marked with
  a `+` prefix character.

**Prefix layout** (5 characters before the name):

```
▶ *+  →  cursor + selected + in-playlist
  *   →  selected only
   +  →  in-playlist only (file already queued)
      →  none
```

**Viewport scrolling:** The navigator implements viewport
scrolling. Only as many entries as fit on screen are rendered.
The viewport offset is adjusted automatically as the cursor
moves so that the cursor is always visible.

Individual files and directories can be toggled into a
"selected" state with `space`, similar to Playlist Control.
Selected entries are marked with a `*` prefix character.

**Library Navigator keybindings:**

| Key          | Action                                               |
| ------------ | ---------------------------------------------------- |
| `j`          | Move cursor down one entry                           |
| `k`          | Move cursor up one entry                             |
| `gg`         | Move cursor to the first entry                       |
| `G`          | Move cursor to the last entry                        |
| `f <letter>` | Jump to next entry starting with letter (wraps)      |
| `Ctrl-D`     | Move cursor down half a page                         |
| `Ctrl-U`     | Move cursor up half a page                           |
| `h`          | Navigate up one directory (back / parent)            |
| `l`          | Enter the directory under cursor                     |
| `space`      | Toggle selection on the entry under cursor           |
| `x`          | Remove selected file(s) from the playlist            |
| `enter`      | Enqueue selected entries to the playlist             |

**Fast-jump behavior (`f <letter>`):**

- `f` followed by a letter (case-insensitive) moves the cursor
  to the next entry whose filename or directory name begins
  with that letter.
- The search starts from the entry after the cursor and wraps
  around to the top of the list if no match is found before
  the end.
- If no entry in the current directory starts with the letter,
  the cursor does not move.
- `f` followed by a non-letter key cancels the sequence with
  no action.

**Navigation behavior:**

- `h` navigates to the parent directory. The cursor is placed
  on the subdirectory that was just exited so the user can see
  where they came from. No-op at the music library root.
- `l` enters a directory. No-op on files.

**Remove behavior (`x`):**

- If one or more entries are selected, `x` removes all
  selected files that are currently in the playlist.
- If no entries are selected, `x` removes the file under
  the cursor if it is in the playlist.
- Directories and files not in the playlist are silently
  skipped — `x` is never an error.
- When a song appears in the playlist multiple times, all
  occurrences are removed.
- After removal the selection is cleared.

**Enqueue behavior:**

- If one or more entries are selected (via `space`), `enter`
  enqueues all selected entries and clears the selection.
- If no entries are selected, `enter` enqueues the single
  entry under the cursor.
- Enqueuing a directory adds all songs within it recursively
  (handled by MPD).
- Enqueued songs are appended to the end of the MPD playlist.

### 4.4 Help Modal

Provides a quick-reference for all keybindings across all
screens. Lists every keybinding organized by section: Global,
Player Volume, Playlist Control, and Library Navigator. Each
entry shows the key and a short description of its action.

Help appears as a centered modal overlay on top of the current
screen. The underlying screen content remains visible around
the edges. `?` toggles the modal open/closed; `Esc` also
closes it. The active screen tab remains highlighted while the
modal is open.

| Key   | Action              |
| ----- | ------------------- |
| `?`   | Toggle help modal   |
| `Esc` | Close help modal    |

## 5. Connection Model

### 5.1 MPD

ghac connects to MPD over TCP using the MPD protocol. It
subscribes to the MPD idle subsystem to receive real-time
notifications for player state changes (play, pause, stop,
seek) and playlist modifications. The idle watcher monitors
both the `player` and `playlist` subsystems.

### 5.2 SnapCast

ghac connects to the SnapCast server using its JSON-RPC API
over TCP. It subscribes to server notifications to receive
real-time updates for client volume changes, mute state
changes, and client connect/disconnect events.

### 5.3 Reconnection

If a connection is lost after a successful startup, ghac
displays an error in the UI and exits. The user must restart
the application.

## 6. Interaction Summary

```text
▶ Artist – Title              ████████████░░░░░░░░  2:34 / 5:01
1:Volume  2:Playlist  3:Library  ?:Help
┌─ Player Volume ─────────────────────────────────┐
│                                                 │
│  (screen-specific content)                      │
│                                                 │
└─────────────────────────────────────────────────┘
```

Global keys active on every screen: `p` play/pause, `z` toggle
random, `q` quit, `Ctrl-C` quit, `1`/`2`/`3` switch screens,
`?` help.

## 7. Edge Cases

- **Empty playlist** — Playlist Control shows "Playlist is
  empty" (italic/faint). The now-playing bar shows
  "[ No song playing ]".
- **No SnapCast clients** — Player Volume shows "No clients
  connected" (italic/faint).
- **Empty directory** — Library Navigator shows "Directory is
  empty" (italic/faint). `h` still navigates to the parent
  directory.
- **Root of music library** — `h` does nothing when already
  at the top-level music directory.
- **Volume at boundaries** — pressing `l`/`L` at 100% or
  `h`/`H` at 0% produces no change and no error.
- **Removing the playing song** — if the user removes the
  currently-playing song with `x`, MPD stops playback and
  advances to the next song per its default behavior.
