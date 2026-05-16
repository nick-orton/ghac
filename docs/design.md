# ghac — Go Home Audio Controller: Product Design

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
1:Volume  2:Playlist  3:Navigator  ?:Help
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
| `3`      | Switch to Song Navigator screen     |
| `?`      | Open the Help screen                |
| `p`      | Toggle play / pause                 |
| `q`      | Quit the application                |
| `Ctrl-C` | Quit the application                |

## 4. Screens

### 4.1 Player Volume (Screen 1)

Controls the volume and mute state of every SnapCast client
connected to the server.

Each SnapCast client occupies one row:

- **Client name** — left-aligned, using the name reported by
  the SnapCast server.
- **Volume bar** — a horizontal bar graph to the right of the
  name, representing the current volume as a percentage
  (0–100).
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
Each line shows song metadata (title, artist) when available,
falling back to the filename. The currently-playing song is
visually distinguished with a marker (e.g., a `>` prefix) and
a distinct color or style (e.g., bold). The song under the
cursor is highlighted.

Songs can be individually toggled into a "selected" state with
`space`. Selected songs are visually marked (e.g., with a
different background or a selection indicator). The selection
is independent of the cursor position — the cursor can move
freely after selecting songs.

**Playlist Control keybindings:**

| Key     | Action                                    |
| ------- | ----------------------------------------- |
| `j`     | Move cursor down one song                 |
| `k`     | Move cursor up one song                   |
| `space` | Toggle selection on the song under cursor  |
| `x`     | Remove selected song(s) from the playlist |
| `X`     | Clear the entire playlist                 |
| `enter` | Start playing the song under cursor       |

**Removal behavior:**

- If one or more songs are selected (via `space`), `x`
  removes all selected songs.
- If no songs are selected, `x` removes the song under the
  cursor.
- After removal, the cursor moves to the next song in the
  list. If the removed song was the last entry, the cursor
  moves up to the new last entry.
- `X` clears the entire playlist and stops playback.

### 4.3 Song Navigator (Screen 3)

Browses the MPD music library using its directory structure
and adds songs or directories to the playlist.

The navigator presents a directory listing styled to
approximate a tab in the `nnn` file manager:

- **Directories** are visually distinct from files (e.g.,
  trailing `/`, different color, or bold).
- **Files** display the filename left-aligned and metadata
  (artist, title, album) right-aligned when available.
- The entry under the cursor is highlighted.
- A breadcrumb or path indicator shows the current directory
  location.

Individual files and directories can be toggled into a
"selected" state with `space`, similar to Playlist Control.
Selected entries are visually marked.

**Song Navigator keybindings:**

| Key     | Action                                     |
| ------- | ------------------------------------------ |
| `j`     | Move cursor down one entry                 |
| `k`     | Move cursor up one entry                   |
| `h`     | Navigate up one directory (back / parent)  |
| `l`     | Enter the directory under cursor           |
| `space` | Toggle selection on the entry under cursor |
| `enter` | Enqueue selected entries to the playlist   |

**Enqueue behavior:**

- If one or more entries are selected (via `space`), `enter`
  enqueues all selected entries and clears the selection.
- If no entries are selected, `enter` enqueues the single
  entry under the cursor.
- Enqueuing a directory adds all songs within it recursively,
  in filesystem sort order.
- Enqueued songs are appended to the end of the MPD playlist.
- `l` on a file does nothing (only directories can be
  entered).

### 4.4 Help Screen

Provides a quick-reference for all keybindings across all
screens. Lists every keybinding organized by section: Global,
Player Volume, Playlist Control, and Song Navigator. Each
entry shows the key and a short description of its action.

| Key   | Action                               |
| ----- | ------------------------------------ |
| `Esc` | Return to the screen that invoked it |

The help screen remembers which screen the user came from and
returns to that screen on exit.

## 5. Connection Model

### 5.1 MPD

ghac connects to MPD over TCP using the MPD protocol. It
subscribes to the MPD idle subsystem to receive real-time
notifications for player state changes (play, pause, stop,
seek), playlist modifications, and database updates.

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
1:Volume  2:Playlist  3:Navigator  ?:Help
┌─ Player Volume ─────────────────────────────┐
│                                             │
│  (screen-specific content)                  │
│                                             │
└─────────────────────────────────────────────┘
```

Global keys active on every screen: `p` play/pause, `q` quit,
`Ctrl-C` quit, `1`/`2`/`3` switch screens, `?` help.

## 7. Edge Cases

- **Empty playlist** — Playlist Control shows an empty state
  message. The now-playing bar shows no song information and
  an empty progress bar.
- **No SnapCast clients** — Player Volume shows an empty
  state message indicating no clients are connected.
- **Empty directory** — Song Navigator shows an empty state
  message. `h` still navigates to the parent directory.
- **Root of music library** — `h` does nothing when already
  at the top-level music directory.
- **Volume at boundaries** — pressing `l`/`L` at 100% or
  `h`/`H` at 0% produces no change and no error.
- **Removing the playing song** — if the user removes the
  currently-playing song with `x`, MPD stops playback and
  advances to the next song per its default behavior.
