# ghac — Technical Architecture

## 1. Overview

ghac is a Go TUI application structured around three concerns:

1. **Backend clients** — long-lived TCP connections to MPD and
   SnapCast that send commands and receive async notifications.
2. **Application state** — a central model that the UI reads and
   the backend clients mutate via message passing.
3. **TUI layer** — renders state to the terminal and translates
   keystrokes into actions.

The architecture uses the Bubble Tea framework (charmbracelet/
bubbletea) which provides an Elm-style update loop: the model
receives messages, produces an updated model, and returns a view
command. External events (MPD idle, SnapCast notifications) are
injected into this loop as messages via Go channels.

## 2. Technology Choices

| Concern        | Library / Tool                        |
| -------------- | ------------------------------------- |
| TUI framework  | charmbracelet/bubbletea               |
| TUI styling    | charmbracelet/lipgloss                |
| MPD client     | fhs/gompd/v2                          |
| SnapCast       | Custom JSON-RPC client over TCP       |
| Config parsing | BurntSushi/toml                       |
| Build          | Standard `go build`; single binary    |

### 2.1 Why These Choices

- **Bubble Tea** is the dominant Go TUI framework. Its message-
  passing architecture maps naturally to the event-driven nature
  of MPD idle and SnapCast notifications.
- **gompd** is a mature, well-tested MPD client that exposes the
  idle subsystem directly.
- **SnapCast** has no established Go client library. Its JSON-RPC
  over TCP protocol is simple enough that a small custom client
  is preferable to pulling in a generic JSON-RPC library that
  would need adaptation for SnapCast's notification model.
- **BurntSushi/toml** is the reference TOML parser for Go.

## 3. Project Layout

```text
ghac/
├── cmd/
│   └── ghac/
│       └── main.go                       # Entry point: config, connect, run
├── internal/
│   ├── config/
│   │   └── config.go                     # TOML parsing, validation
│   ├── mpd/
│   │   ├── client.go                     # MPD connection, commands
│   │   └── messages.go                   # Bubble Tea messages from MPD
│   ├── snapcast/
│   │   ├── client.go                     # JSON-RPC TCP client, commands
│   │   ├── messages.go                   # Bubble Tea messages from SnapCast
│   │   └── client_integration_test.go    # Integration tests (build tag)
│   └── ui/
│       ├── model.go           # Root model, screen routing, screen border
│       ├── nowplaying.go      # Now-playing bar component
│       ├── styles.go          # Shared lipgloss styles (color vars, reassigned by applyTheme)
│       ├── theme.go           # Theme type, applyTheme, XDG state I/O; embeds themes.toml
│       ├── themes.toml        # Built-in theme definitions (embedded at build time)
│       ├── thememodal.go      # Theme selector modal screen
│       ├── volume.go          # Player Volume screen (SnapCast clients)
│       ├── playlist.go        # Playlist Control screen
│       ├── navigator.go       # Library Navigator screen (library browser)
│       └── help.go            # Help screen
├── docs/
│   ├── requirements.md
│   ├── design.md
│   ├── architecture.md
│   ├── plan.md
│   └── ux.md
├── go.mod
└── go.sum
```

All application code lives under `internal/` to prevent external
import. The `cmd/ghac/` package contains only bootstrap logic.

## 4. Application Lifecycle

```text
main.go
  ├── Load config from $HOME/.config/.ghacrc
  ├── Connect to MPD (TCP)
  ├── Connect to SnapCast (TCP)
  ├── Build initial model (fetch current state from both servers)
  ├── Start Bubble Tea program
  │     ├── Spawn MPD idle listener (goroutine → tea.Msg)
  │     └── Spawn SnapCast notification listener (goroutine → tea.Msg)
  └── Block until program exits
```

### 4.1 Startup

1. Parse config. Exit with error if file missing or invalid.
2. Dial MPD. Exit with error on failure.
3. Fetch initial MPD state: player status, playlist, music
   library root listing.
4. Dial SnapCast. Exit with error on failure.
5. Fetch initial SnapCast client list.
6. Construct root model via `ui.New(NewParams{...})` and start
   the TUI with alt-screen mode.

### 4.2 Shutdown

On `q` or `Ctrl-C`, the Bubble Tea program returns. `main.go`
closes both connections (via deferred `Close()` calls) and exits
cleanly.

### 4.3 Connection Loss

If either backend disconnects after startup, the respective
listener goroutine sends a fatal error message (`mpd.MsgError`
or `snapcast.MsgError`) into the Bubble Tea loop. The root model
stores the error in `errMsg`, renders it, and issues `tea.Quit`.

## 5. Message Flow

The Bubble Tea update loop is the single synchronization point.
No shared mutable state exists outside the model.

```text
┌──────────────┐          ┌──────────────┐
│ MPD idle     │────Msg──▶│              │
│ goroutine    │          │              │
└──────────────┘          │  Bubble Tea  │
                          │  Update Loop │
┌──────────────┐          │              │
│ SnapCast     │────Msg──▶│  (Model)     │
│ goroutine    │          │              │
└──────────────┘          └──────┬───────┘
                                 │
                          ┌──────▼───────┐
                          │    View()    │
                          │  (render)    │
                          └──────────────┘
```

### 5.1 Message Types

**From MPD (`internal/mpd/messages.go`):**

- `MsgPlayerState` — play/pause/stop status, current song,
  elapsed time, current song's playlist position (`SongPos`;
  -1 when nothing is playing), and random mode flag (`Random`).
- `MsgPlaylistChanged` — the playlist was modified; carries the
  full updated `[]PlaylistEntry`.
- `MsgTick` — periodic tick for progress bar updates (1-second
  interval, driven by `tea.Tick`).
- `MsgError` — the MPD connection was lost or a fatal error
  occurred; the root model displays it and quits.

**From SnapCast (`internal/snapcast/messages.go`):**

- `MsgClientsUpdated` — one or more clients changed volume,
  mute state, or connected/disconnected.
- `MsgError` — the SnapCast connection was lost or a fatal
  error occurred; the root model displays it and quits.

**From user input (built into Bubble Tea):**

- `tea.KeyMsg` — a key was pressed.
- `tea.WindowSizeMsg` — the terminal was resized.

## 6. Component Design

### 6.1 Root Model

The root model (`internal/ui/model.go`) owns:

- The active screen identifier (a `screenID` enum/int; three
  values: `screenVolume`, `screenPlaylist`, `screenNavigator`).
- `showHelp bool` — when true, a help modal is composited over
  the active screen via `placeOverlay()`. The active screen never
  changes when help opens; `?` and `Esc` toggle this flag.
- Terminal dimensions (`width`, `height`).
- Sub-models for each screen (`volume`, `playlist`, `navigator`,
  `help`).
- Pointers to the MPD and SnapCast clients (for issuing
  commands).
- Shared player state: `playerStatus`, `currentSong`, `elapsed`,
  `totalDuration`, `currentSongPos`, `randomOn`.
- `errMsg` for fatal error display.

The root model's `Update` method handles:

1. `tea.WindowSizeMsg` — stores dimensions and propagates width
   and height to the navigator screen.
2. `MsgPlayerState` — updates player state fields, updates the
   playlist screen's current position, re-subscribes to idle.
3. `MsgPlaylistChanged` — updates the playlist screen's entries,
   updates the navigator screen's in-playlist set, re-subscribes
   to idle.
4. `MsgTick` — increments elapsed time by 1 second when playing,
   re-subscribes to tick.
5. `mpd.MsgError` / `snapcast.MsgError` — stores error and quits.
6. `tea.KeyMsg` — checks `volume.showRename` first; when true,
   keys are delegated directly to the volume screen without
   processing global bindings (same guard pattern as
   `showHelp`). Otherwise handles global keys (screen
   switching, play/pause, quit) and delegates remaining keys to
   the active screen's sub-model via
   `delegateToActiveScreen()`.

### 6.2 Screen Sub-Models

Each screen is a struct with `Update` and `View` methods:

```go
func (s screenType) Update(msg tea.Msg) (screenType, tea.Cmd)
func (s screenType) View() string
```

Screens do not import each other. They receive state they need
from the root model or via messages. Each screen's `View()`
returns content only — no title or border. The root model wraps
it with `screenBorder()`.

### 6.3 Now-Playing Component

A stateless view function (`NowPlayingView` in `nowplaying.go`)
that accepts a `PlayerState` struct and terminal width, returning
a rendered string. It is called once by `Model.View()` to
compose the top bar; individual screen `View()` methods do not
call it.

### 6.4 Theme System

Themes come from two sources:

1. **Built-in** — `internal/ui/themes.toml` is embedded at build time via
   `//go:embed` and parsed by `init()` into `Themes []Theme`.
2. **User-defined** — `[[themes]]` blocks in `.ghacrc` are loaded by
   `config.Load()` as `[]ThemeConfig`, converted to `[]ui.Theme` in
   `main.go`, and appended via `ui.AppendThemes()` before theme
   resolution. User themes appear after built-ins in the selector.

An unnamed user theme block is skipped with a stderr warning.

`applyTheme(t Theme)` reassigns the color-bearing style variables
in `styles.go` (e.g., `styleNowPlaying`, `styleProgressFill`)
to match the chosen theme. Because all UI rendering executes on
the single Bubble Tea goroutine, this reassignment is safe with
no locking required.

The theme selector is a modal overlay (`themeScreen` in
`thememodal.go`) using the same `placeOverlay()` mechanism as
the help modal. Cursor movement triggers `applyTheme()` immediately
for live preview. The root model handles `Enter` (confirm + save)
and `Esc` / `t` (revert to original). Theme state is persisted to
`$XDG_STATE_HOME/ghac/theme`. The active theme is also selectable
via the `theme` config field or the `--theme` CLI flag; the flag
takes highest priority.

## 7. Backend Client Design

### 7.1 MPD Client (`internal/mpd/`)

Wraps `gompd` and exposes two concerns:

1. **Command interface** — methods called synchronously from the
   Bubble Tea `Update` in response to user input (MPD commands
   are fast, sub-millisecond on LAN):
   - `Connect(addr)` — dials MPD, returns a `*Client`.
   - `Close()` — closes both connections.
   - `Ping()` — keep-alive.
   - `Status()` — returns `MsgPlayerState`.
   - `CurrentSong()` — returns `Song`.
   - `Play()` — resumes playback (no position argument).
   - `Pause()` — pauses playback.
   - `PlaylistInfo()` — returns `[]PlaylistEntry` for the full
     queue.
   - `PlayAt(pos)` — starts playing at a 0-indexed position.
   - `Delete(pos)` — removes one song at a 0-indexed position.
   - `Clear()` — removes all songs and stops playback.
   - `ListInfo(path)` — lists a directory's contents, returning
     `[]DirEntry`. Empty string for the music library root.
     Note: gompd's `ListInfo` lowercases all attribute keys,
     unlike other query methods that preserve MPD's original
     capitalization. Playlist entries in the MPD response are
     skipped.
   - `Add(uri)` — appends a song or directory (recursively) to
     the playback queue.
   - `Random(on)` — enables or disables MPD's random (shuffle)
     mode.

2. **Idle listener** — a long-running goroutine that calls
   `gompd`'s `Watch` to block on MPD idle events. On each event
   it queries the relevant subsystem and emits a `tea.Msg` via
   a `tea.Cmd` channel. Watches both `player` and `playlist`
   subsystems; returns `MsgPlayerState` on player events and
   `MsgPlaylistChanged` on playlist events. Returns `MsgError`
   on connection loss.

The MPD protocol requires a separate connection for idle
watching (it monopolizes the connection). The client therefore
maintains two TCP connections: one for commands, one for idle.

### 7.2 SnapCast Client (`internal/snapcast/`)

A custom client implementing SnapCast's JSON-RPC over TCP:

1. **Command interface** — `GetServerStatus()`,
   `SetVolume(clientID, vol, muted)`, `SetMute(clientID,
   muted, currentVol)`, `SetName(clientID, name)` that send
   JSON-RPC requests and block until the response arrives
   (5-second timeout). Because the SnapCast protocol encodes
   volume and muted state in a single field, both values are
   always supplied together. `SetName` sends the
   `Client.SetName` RPC to rename a SnapCast client.

2. **Reader goroutine** (`readLoop`) — a persistent goroutine
   started by `Connect()` that reads the TCP stream, decodes
   each JSON-RPC message, and dispatches it: responses go to
   the waiting caller's channel (keyed by request ID);
   server-initiated notifications go to an internal `notify`
   channel (buffered, capacity 16).

3. **Notification listener** (`ListenNotifications()`) —
   returns a `tea.Cmd` that blocks on the `notify` channel.
   On each notification it calls `GetServerStatus()` to fetch
   the full updated client list and returns
   `MsgClientsUpdated`. Returns `MsgError` on connection loss.
   The root model re-calls it from `Update` to keep
   listening — the same re-subscribe pattern used by the MPD
   idle listener.

The SnapCast protocol multiplexes commands and notifications on
a single connection. The client uses a mutex-protected request
map keyed by JSON-RPC ID to correlate responses with pending
requests, while notifications (no matching ID) go to the
`notify` channel.

## 8. State Model

State is distributed across the root model and per-screen
sub-models rather than collected in a single struct. Types live
in the package that owns them.

**Root model** (`internal/ui/Model`) owns:

- Player state fields (`playerStatus`, `currentSong`,
  `elapsed`, `totalDuration`, `currentSongPos`, `randomOn`) —
  populated from MPD messages.
- Terminal dimensions (`width`, `height`).
- `errMsg` — set on fatal errors; when non-empty, `View()`
  renders only the error and `Update()` quits.
- `showHelp bool` — when true, `View()` composites the help
  modal over the active screen using `placeOverlay()`.
- `showTheme bool` — when true, `View()` composites the theme
  selector modal. Mutually exclusive with `showHelp`.
- `activeThemeIdx`, `originalThemeIdx` — current and pre-open
  theme indices used for live preview and revert on cancel.
- `themeModal themeScreen` — the theme selector sub-model.
- Sub-models for each screen (`volume`, `playlist`,
  `navigator`, `help`).
- Pointers to the MPD and SnapCast clients.

**MPD types** (`internal/mpd/messages.go`):

```go
type Song struct {
    Title  string
    Artist string
    Album  string
    File   string // fallback display
}

type PlaylistEntry struct {
    Song
    Pos int // 0-indexed position in the playlist
}

type DirEntry struct {
    Name  string // basename for display
    Path  string // full MPD URI (relative to music root)
    IsDir bool
    Song  Song   // populated for files; zero value for directories
}
```

**SnapCast types** (`internal/snapcast/messages.go`):

```go
type SnapClient struct {
    ID     string
    Name   string
    Volume int  // 0–100
    Muted  bool
}
```

**Volume screen** (`volumeScreen`) owns `[]SnapClient` and a
cursor index. It also owns three fields for the rename modal:
`showRename bool` (whether the modal is open), `renameInput
[]rune` (the editable name buffer), and `renameCursor int`
(the cursor position within the buffer). When `showRename` is
true, `volumeScreen.Update()` handles only text-editing keys,
`Ctrl-S` (save), and `Esc` (cancel). Updates when
`MsgClientsUpdated` arrives via `withClients()`. Holds a
pointer to the SnapCast client for issuing volume/mute/rename
commands.

**Playlist screen** (`playlistScreen`) owns `[]PlaylistEntry`,
a cursor index, a viewport `offset` (index of the first visible
row), a `height` (terminal height in rows), a `map[int]bool`
selection set, `pendingG` and `pendingF` booleans for two-key
sequences (`gg` and `f<letter>`), and the `currentPos` of the
playing song. `f<letter>` is handled in `Update` before the
normal key switch: when `pendingF` is set the next keystroke is
consumed; if it is a letter, `jumpToLetter` searches forward from
`cursor+1` (wrapping) by the first character of
`entryDisplayName()` and calls `clampOffset()` after a match.
Non-letter keys cancel with no action. `removeSongs()` also calls
`clampOffset()` after clamping the cursor so the viewport is
correct following bulk removal. Updates when `MsgPlaylistChanged`
arrives via `withEntries()`. The root model forwards
`tea.WindowSizeMsg` height via `withHeight()`, matching the
navigator pattern. Holds a pointer to the MPD client for issuing
playlist commands.

**Navigator screen** (`navigatorScreen`) owns `[]DirEntry`,
a cursor index, a viewport `offset`, `pendingG` and `pendingF`
booleans for two-key sequences (`gg` and `f<letter>`), a
`map[int]bool` selection set, a `map[string][]int` `inPlaylist`
map (MPD URI → playlist positions), `currentPath` (current
directory URI), and terminal `width`/`height`. `f<letter>` uses
the same pending-key pattern as the playlist screen; `jumpToLetter`
searches forward from `cursor+1` (wrapping) by the first character
of `entry.Name` and calls `clampOffset()` after a match. The
entry list updates synchronously when the user navigates
directories (calling `ListInfo` directly from the `Update`
method). The `inPlaylist` map updates from `MsgPlaylistChanged`
via `withPlaylist()`. Holds a pointer to the MPD client for
browsing and enqueue commands.

## 9. Configuration

```go
type Config struct {
    SnapServer ServerConfig  `toml:"snapserver"`
    MPD        ServerConfig  `toml:"mpd"`
    Theme      string        `toml:"theme"`  // optional; selects a theme by name
    Themes     []ThemeConfig `toml:"themes"` // optional; user-defined themes
}

type ThemeConfig struct {
    Name          string `toml:"name"`
    BarBG         string `toml:"bar_bg"`
    BarFG         string `toml:"bar_fg"`
    Accent        string `toml:"accent"`
    ProgressEmpty string `toml:"progress_empty"`
    Secondary     string `toml:"secondary"`
    VolumeUnmuted string `toml:"volume_unmuted"`
    VolumeMuted   string `toml:"volume_muted"`
}

type ServerConfig struct {
    IP   string `toml:"ip"`
    Port int    `toml:"port"`
}
```

Example `.ghacrc`:

```toml
[snapserver]
ip = "192.168.1.10"
port = 1705

[mpd]
ip = "192.168.1.10"
port = 6600
```

Validation: IP must be non-empty, port must be 1–65535 for both
servers.

## 10. Concurrency Model

The application uses four goroutines beyond the main Bubble Tea
event loop:

1. **MPD idle listener** (tea-managed) — started by
   `ListenIdle()` returning a `tea.Cmd`. Watches both the
   `player` and `playlist` subsystems; returns `MsgPlayerState`
   on player events and `MsgPlaylistChanged` on playlist events.
   The root model re-calls it after every message to keep it
   running.

2. **SnapCast reader loop** (persistent) — started by
   `snapcast.Connect()`. Reads the TCP stream continuously,
   dispatching JSON-RPC responses to per-request channels and
   forwarding notifications to the internal `notify` channel.
   Exits only when the connection closes.

3. **SnapCast notification listener** (tea-managed) — started
   by `ListenNotifications()` returning a `tea.Cmd`. Blocks on
   the `notify` channel; on each notification it fetches the
   full server status and returns `MsgClientsUpdated`. The root
   model re-calls it after every message.

4. **Progress ticker** (tea-managed) — `tea.Tick` fires every
   second, producing `MsgTick` to advance the progress bar.

All goroutines communicate exclusively through Bubble Tea's
command/message system. No shared mutable state exists outside
the model (the SnapCast client's internal request map and write
mutex are the sole exception, as noted in 7.2).

## 11. Error Handling Strategy

- **Startup errors** (bad config, connection refused): print to
  stderr, exit non-zero.
- **Runtime connection loss**: send a fatal message (`MsgError`)
  into the Bubble Tea loop; the program stores it in `errMsg`,
  renders the error, and quits.
- **Command failures** (e.g., MPD returns an error for a remove
  operation): errors are silently discarded (the caller ignores
  the return via `_ = ...`). The server confirms the operation
  via a subsequent idle notification.

## 12. Testing Strategy

- **Backend clients**: integration tests against real or
  containerized MPD/SnapCast instances, gated behind a build
  tag (`//go:build integration`).
- **UI logic**: unit tests on sub-model `Update` methods by
  sending synthetic messages and asserting state changes.
- **Config**: table-driven unit tests for parsing and
  validation.

## 13. Future Considerations

These items are explicitly out of scope for the initial build
but inform architectural choices:

- Additional MPD features (search, random mode, repeat).
- Configurable keybindings.
- Mouse support.

The screen interface and message-passing architecture
accommodate these without structural changes.
