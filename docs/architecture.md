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
│       ├── model.go           # Root model, screen routing
│       ├── nowplaying.go      # Now-playing bar component
│       ├── styles.go          # Shared lipgloss styles
│       ├── volume.go          # Player Volume screen (SnapCast clients)
│       ├── playlist.go        # Playlist Control screen (placeholder)
│       ├── navigator.go       # Song Navigator screen (placeholder)
│       └── help.go            # Help screen
├── docs/
│   ├── requirements.md
│   ├── design.md
│   └── architecture.md
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
3. Dial SnapCast. Exit with error on failure.
4. Query initial state: current song, playlist, SnapCast clients.
5. Construct root model and start the TUI.

### 4.2 Shutdown

On `q` or `Ctrl-C`, the Bubble Tea program returns. `main.go`
closes both connections and exits cleanly.

### 4.3 Connection Loss

If either backend disconnects after startup, the respective
listener goroutine sends a fatal error message into the Bubble
Tea loop. The program displays the error and exits.

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

**From MPD:**

- `MsgPlayerState` — play/pause/stop status, current song,
  elapsed time, and current song's playlist position (`SongPos`;
  -1 when nothing is playing).
- `MsgPlaylistChanged` — the playlist was modified; carries the
  full updated `[]PlaylistEntry`.
- `MsgTick` — periodic tick for progress bar updates (1s
  interval).

**From SnapCast:**

- `MsgClientsUpdated` — one or more clients changed volume,
  mute state, or connected/disconnected.

**From user input (built into Bubble Tea):**

- `tea.KeyMsg` — a key was pressed.

## 6. Component Design

### 6.1 Root Model

The root model (`internal/ui/model.go`) owns:

- The active screen identifier (an enum/int).
- The previous screen (for help-screen return).
- Sub-models for each screen.
- References to the MPD and SnapCast clients (for issuing
  commands).
- Shared state: current player status, now-playing info.

The root model's `Update` method handles global keys (screen
switching, play/pause, quit) and delegates remaining keys to
the active screen's sub-model.

### 6.2 Screen Sub-Models

Each screen implements a consistent interface:

```go
type Screen interface {
    Update(msg tea.Msg) (Screen, tea.Cmd)
    View() string
}
```

Screens do not import each other. They receive state they need
from the root model or via messages.

### 6.3 Now-Playing Component

A stateless view function (`NowPlayingView` in `nowplaying.go`)
that accepts current player state and terminal width, returning
a rendered string. It is called once by `Model.View()` to
compose the top bar; individual screen `View()` methods do not
call it.

## 7. Backend Client Design

### 7.1 MPD Client (`internal/mpd/`)

Wraps `gompd` and exposes two concerns:

1. **Command interface** — methods like `Play()`, `Pause()`,
   `Remove(pos int)`, `Add(uri string)` that issue MPD commands.
   Called synchronously from the Bubble Tea `Update` in response
   to user input (MPD commands are fast, sub-millisecond on
   LAN).

2. **Idle listener** — a long-running goroutine that calls
   `gompd`'s `Watch` to block on MPD idle events. On each event
   it queries the relevant subsystem and emits a `tea.Msg` via
   a `tea.Cmd` channel.

The MPD protocol requires a separate connection for idle
watching (it monopolizes the connection). The client therefore
maintains two TCP connections: one for commands, one for idle.

### 7.2 SnapCast Client (`internal/snapcast/`)

A custom client implementing SnapCast's JSON-RPC over TCP:

1. **Command interface** — `GetServerStatus()`,
   `SetVolume(clientID, vol, muted)`, `SetMute(clientID,
   muted, currentVol)` that send JSON-RPC requests and block
   until the response arrives (5-second timeout). Because the
   SnapCast protocol encodes volume and muted state in a single
   field, both values are always supplied together.

2. **Reader goroutine** (`readLoop`) — a persistent goroutine
   started by `Connect()` that reads the TCP stream, decodes
   each JSON-RPC message, and dispatches it: responses go to
   the waiting caller's channel (keyed by request ID);
   server-initiated notifications go to an internal `notify`
   channel.

3. **Notification listener** (`ListenNotifications()`) —
   returns a `tea.Cmd` that blocks on the `notify` channel.
   On each notification it calls `GetServerStatus()` to fetch
   the full updated client list and returns
   `MsgClientsUpdated`. The root model re-calls it from
   `Update` to keep listening — the same re-subscribe pattern
   used by the MPD idle listener.

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
  `elapsed`, `totalDuration`, `currentSongPos`) — populated
  from MPD messages.
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

The volume screen sub-model owns `[]SnapClient` and updates it
when `MsgClientsUpdated` arrives. The playlist screen sub-model
owns `[]PlaylistEntry` and updates it when `MsgPlaylistChanged`
arrives. Navigator state types (`DirEntry`) will be added in
Phase 5.

## 9. Configuration

```go
type Config struct {
    SnapServer ServerConfig `toml:"snapserver"`
    MPD        ServerConfig `toml:"mpd"`
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
- **Runtime connection loss**: send a fatal message into the
  Bubble Tea loop; the program renders the error and quits.
- **Command failures** (e.g., MPD returns an error for a remove
  operation): log the error in a transient status area or
  silently ignore if the failure is benign (like volume at
  boundary).

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
- Theme/color configuration.
- Mouse support.

The screen interface and message-passing architecture
accommodate these without structural changes.
