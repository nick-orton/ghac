# ghac — Phased Build Plan

## Principles

- Each phase delivers a working, runnable binary.
- Phases build on each other; no phase leaves dead code or
  stubs that must be replaced later.
- Every phase ends with a testing step that adds automated
  tests for the functionality introduced in that phase.
- "Deliver half a thing, not a half-assed thing."

## Phase 1: Skeleton, Config, and TUI Shell (Complete)

**Goal:** A running TUI that reads configuration, switches
between placeholder screens, and quits cleanly.

### 1.1 Deliverables

- Project scaffolding: `go.mod`, directory structure per
  architecture doc (`cmd/ghac/`, `internal/config/`,
  `internal/ui/`).
- `internal/config/config.go` — parse and validate
  `$HOME/.config/.ghacrc` (TOML). Exit with descriptive
  error on missing file or invalid content.
- `internal/ui/model.go` — root Bubble Tea model with screen
  routing. Keys `1`, `2`, `3` switch screens; `?` opens help;
  `q` and `Ctrl-C` quit.
- Placeholder screens for Player Volume, Playlist Control,
  and Library Navigator that display their title and a
  "not yet connected" message.
- `internal/ui/help.go` — help screen showing all planned
  keybindings organized by section. `Esc` returns to the
  previous screen.
- `cmd/ghac/main.go` — entry point that loads config and
  starts the TUI.
- README.md which tells the user how to build, run, and use
  the application.

### 1.2 What Works at the End

The user can run `ghac`, see screens, switch between them
with `1`/`2`/`3`, open help with `?`, return with `Esc`, and
quit with `q`. Config errors produce clear messages.

### 1.3 Testing

- Table-driven unit tests for config parsing: valid file,
  missing file, missing fields, invalid TOML.
- Unit tests for root model: send `tea.KeyMsg` for screen
  switches and assert active screen changes.
- Unit tests for help screen: enter from each screen, verify
  `Esc` returns to the correct origin.

---

## Phase 2: MPD Connection, Now-Playing Bar, Play/Pause (Complete)

**Goal:** Connect to a real MPD server, display the
now-playing bar on every screen, and support play/pause.

### 2.1 Deliverables

- `internal/mpd/client.go` — MPD client wrapping `gompd/v2`.
  Two connections: one for commands, one for idle. Methods:
  `Connect()`, `Close()`, `Status()`, `CurrentSong()`,
  `Play()`, `Pause()`, `Ping()`.
- `internal/mpd/messages.go` — Bubble Tea message types:
  `MsgPlayerState`, `MsgTick`, `MsgError`.
- MPD idle listener goroutine that watches for player events
  and emits `MsgPlayerState`.
- Progress ticker goroutine (1-second interval) emitting
  `MsgTick`.
- `internal/ui/nowplaying.go` — now-playing bar component
  rendering song metadata (title/artist/album with filename
  fallback) and a progress bar with `elapsed / total` time.
- `cmd/ghac/main.go` updated to connect to MPD on startup
  (exit with error on failure), fetch initial state, and
  wire the idle listener into the Bubble Tea program.
- Global `p` key toggles play/pause.

### 2.2 What Works at the End

The user runs `ghac` pointed at a real MPD server. The
now-playing bar shows the current song and updates in real
time. Pressing `p` toggles playback. All screen switching
from Phase 1 still works.

### 2.3 Testing

- Unit tests for `MsgPlayerState` and `MsgTick` handling in
  the root model (synthetic messages, assert state changes).
- Unit tests for now-playing view rendering: playing state,
  paused state, missing metadata (filename fallback), empty
  playlist.
- Integration tests (behind `//go:build integration` tag)
  for MPD client connect, status, play, pause against a
  containerized or real MPD instance.

---

## Phase 3: SnapCast Client and Player Volume Screen (Complete)

**Goal:** Connect to SnapCast, display real client volumes,
and allow volume/mute control.

### 3.1 Deliverables

- `internal/snapcast/client.go` — custom JSON-RPC over TCP
  client. Methods: `Connect()`, `Close()`,
  `GetServerStatus()`, `SetVolume(clientID, vol, muted)`,
  `SetMute(clientID, muted, currentVol)`. Mutex-protected
  request map for correlating responses by JSON-RPC ID.
- `internal/snapcast/messages.go` — `MsgClientsUpdated` and
  `MsgError` message types.
- SnapCast notification listener goroutine that reads the
  TCP stream for server-pushed events and emits
  `MsgClientsUpdated`.
- `internal/ui/volume.go` — Player Volume screen:
  - One row per SnapCast client: name + volume bar graph
    (0–100) + mute indicator.
  - Cursor navigation with `j`/`k`.
  - `h`/`l` adjust focused client volume by 5%.
  - `m` toggles mute on focused client.
  - `H`/`L`/`M` adjust or mute all clients.
  - Highlighted row for cursor focus.
  - Real-time updates from SnapCast notifications.
- `cmd/ghac/main.go` updated to connect to SnapCast on
  startup, fetch initial client list, and wire the
  notification listener.

### 3.2 What Works at the End

The user sees real SnapCast clients on Screen 1, adjusts
volume and mute per-client or globally, and sees changes
from other controllers reflected in real time.

### 3.3 Testing

- Unit tests for the volume screen sub-model: cursor
  movement, volume clamp at 0 and 100, mute toggle,
  all-client operations, `MsgClientsUpdated` handling.
- Unit tests for volume bar rendering at various levels
  and mute states.
- Integration tests (behind `//go:build integration` tag)
  for SnapCast client: connect, get status, set volume,
  set mute.

---

## Phase 4: Playlist Control Screen (Complete)

**Goal:** Display and manage the MPD playlist.

### 4.1 Deliverables

- Additional MPD client methods: `PlaylistInfo()`,
  `PlayAt(pos)`, `Delete(pos)`, `Clear()`.
- `MsgPlaylistChanged` message type. MPD idle listener
  extended to watch playlist subsystem and emit this
  message.
- `internal/ui/playlist.go` — Playlist Control screen:
  - One song per line with metadata (title/artist) or
    filename fallback.
  - Currently-playing song marked with `>` prefix and
    bold cyan style.
  - Cursor navigation with `j`/`k`.
  - `space` toggles selection on songs (marked with `*`).
  - `x` removes selected songs (or cursor song if none
    selected). Cursor repositions per spec.
  - `X` clears the entire playlist and stops playback.
  - `enter` starts playing the song under cursor.
  - Real-time updates via `MsgPlaylistChanged`.

### 4.2 What Works at the End

The user can view their playlist on Screen 2, select and
remove songs, clear the playlist, and jump to any song.
Changes from other MPD clients appear in real time.

### 4.3 Testing

- Unit tests for playlist sub-model: cursor movement,
  selection toggle, remove-selected vs remove-cursor,
  clear, cursor repositioning after removal, enter-to-play.
- Unit tests for rendering: playing song marker, selected
  song indicator, empty playlist message.
- Integration tests for the new MPD client methods.

---

## Phase 5: Library Navigator Screen (Complete)

**Goal:** Browse the MPD library by directory structure and
enqueue songs to the playlist.

### 5.1 Deliverables

- Additional MPD client methods: `ListInfo(path)` to list
  directory contents, `Add(uri)` to enqueue a song or
  directory.
- `DirEntry` type in `internal/mpd/messages.go`.
- `internal/ui/navigator.go` — Library Navigator screen:
  - Directory listing styled after `nnn`: directories
    rendered bold with trailing `/`; files show filename
    left-aligned, metadata ("Title – Artist") right-aligned
    when terminal width allows.
  - Breadcrumb line showing the current directory path.
  - Cursor navigation with `j`/`k`.
  - `Ctrl-D`/`Ctrl-U` for half-page jumps.
  - `h` navigates to parent directory (cursor placed on the
    directory just exited). No-op at root.
  - `l` enters directory under cursor (no-op on files).
  - `space` toggles selection on entries (marked with `*`).
  - `enter` enqueues selected entries (or cursor entry if
    none selected). Directories enqueue recursively.
    Clears selection after enqueue.
  - `+` prefix marker on files already in the playlist
    (tracked via `inPlaylist map[string]bool`, rebuilt on
    every `MsgPlaylistChanged`).
  - Viewport scrolling: only the entries that fit on screen
    are rendered; the viewport offset auto-adjusts as the
    cursor moves. Viewport height is calculated from the
    terminal height minus the fixed UI overhead (7 lines).

### 5.2 What Works at the End

The user can browse their music library on Screen 3,
navigate directories, select files and folders, and add
them to the playlist. The in-playlist marker shows which
files are already queued. All five screens (Volume,
Playlist, Navigator, Help, Now-Playing bar) are fully
functional.

### 5.3 Testing

- Unit tests for navigator sub-model: cursor movement,
  directory enter/exit, root boundary, selection toggle,
  enqueue single vs enqueue selected, enqueue directory,
  viewport offset clamping, half-page jumps.
- Unit tests for rendering: directory vs file styling,
  breadcrumb display, empty directory message, in-playlist
  marker.
- Integration tests for `ListInfo` and `Add` against MPD.

---

## Phase 6: Edge Cases, Error Handling, and Final Polish (Complete)

**Goal:** Harden the application, handle all documented edge
cases, and ensure comprehensive test coverage.

### 6.1 Deliverables

- **Edge case handling:**
  - Empty playlist state message in Playlist Control and
    now-playing bar.
  - No SnapCast clients state message in Player Volume.
  - Empty directory state message in Library Navigator.
  - Removing the currently-playing song (MPD default
    behavior: advance to next).
  - Volume boundary clamping already in place; verify
    silently no-ops.
- **Connection loss handling:** MPD and SnapCast listener
  goroutines detect disconnection and send a fatal error
  message into the Bubble Tea loop. The TUI renders the
  error and exits cleanly.
- **Startup error messages:** verify all failure paths
  (bad config, MPD refused, SnapCast refused) produce
  clear, descriptive messages on stderr.
- **Visual polish:** review all screens for consistent
  styling with lipgloss, correct terminal-width handling,
  and clean layout at various terminal sizes.

### 6.2 Testing

- Unit tests for every documented edge case (empty states,
  boundary conditions, removal of playing song).
- Unit tests for connection-loss message handling in the
  root model (assert error display and quit behavior).
- Integration test suite: end-to-end workflow covering
  config load, connect to both servers, navigate all
  screens, perform key operations, and disconnect
  gracefully.
- Manual smoke test checklist:
  - Launch with valid config, verify all screens.
  - Launch with missing config, verify error.
  - Launch with unreachable MPD, verify error.
  - Launch with unreachable SnapCast, verify error.
  - Adjust volume from another client, verify real-time
    update.
  - Modify playlist from another client, verify real-time
    update.
  - Kill MPD mid-session, verify error and clean exit.
  - Kill SnapCast mid-session, verify error and clean
    exit.

---

## Phase Summary

| Phase | Focus                    | Status   | Screens Working        |
| ----- | ------------------------ | -------- | ---------------------- |
| 1     | Skeleton, config, shell  | Complete | Shell + Help           |
| 2     | MPD + now-playing        | Complete | Now-playing bar + p    |
| 3     | SnapCast + volume        | Complete | Player Volume          |
| 4     | Playlist control         | Complete | Playlist Control       |
| 5     | Song navigator           | Complete | Library Navigator         |
| 6     | Edge cases + polish      | Complete | All (hardened)         |
