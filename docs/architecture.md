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
| TUI components | charmbracelet/bubbles                 |
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
│       └── main.go            # Entry point: config, connect, run
├── internal/
│   ├── config/
│   │   └── config.go          # TOML parsing, validation
│   ├── mpd/
│   │   ├── client.go          # MPD connection, commands
│   │   └── messages.go        # Bubble Tea messages from MPD
│   ├── snapcast/
│   │   ├── client.go          # JSON-RPC connection, commands
│   │   └── messages.go        # Bubble Tea messages from SnapCast
│   └── ui/
│       ├── model.go           # Root model, screen routing
│       ├── nowplaying.go      # Now-playing bar component
│       ├── volume.go          # Player Volume screen
│       ├── playlist.go        # Playlist Control screen
│       ├── navigator.go       # Song Navigator screen
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
  elapsed time.
- `MsgPlaylistChanged` — the playlist was modified; carry new
  playlist contents.
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

A stateless view function that accepts current player state and
terminal width, returning a rendered string. It is called by
every screen's `View()` to compose the top bar.

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

1. **Command interface** — methods like `SetVolume(clientID,
   vol)`, `SetMute(clientID, muted)` that send JSON-RPC
   requests and await responses.

2. **Notification listener** — reads the TCP stream for
   server-initiated notifications (volume changes, client
   connect/disconnect) and emits them as `tea.Msg`.

The SnapCast protocol multiplexes commands and notifications on
a single connection. The client uses a mutex-protected request
map keyed by JSON-RPC ID to correlate responses with pending
requests, while notifications (which lack an ID field matching
a pending request) are forwarded to the message channel.

## 8. State Model

```go
type AppState struct {
    // Player
    PlayerStatus   string  // "play", "pause", "stop"
    CurrentSong    Song
    Elapsed        time.Duration
    TotalDuration  time.Duration

    // SnapCast
    Clients []SnapClient

    // Playlist
    Playlist []PlaylistEntry

    // Navigator
    CurrentPath string
    DirEntries  []DirEntry
}

type Song struct {
    Title    string
    Artist   string
    Album    string
    File     string // fallback display
}

type SnapClient struct {
    ID     string
    Name   string
    Volume int  // 0-100
    Muted  bool
}

type PlaylistEntry struct {
    Position int
    Song     Song
}

type DirEntry struct {
    Name  string
    IsDir bool
    Song  Song // populated only for files with metadata
}
```

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

The application uses exactly three goroutines beyond the main
Bubble Tea loop:

1. **MPD idle watcher** — blocks on idle, queries state, sends
   message, repeats.
2. **SnapCast notification reader** — reads TCP stream, parses
   JSON-RPC notifications, sends messages.
3. **Progress ticker** — sends a tick message every second for
   the now-playing progress bar.

All goroutines communicate exclusively through Bubble Tea's
command/message system. There is no shared mutable state and no
explicit locking in application code (the SnapCast client's
internal request map is the sole exception, as noted in 7.2).

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
