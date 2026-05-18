# ghac — Great Home Audio Controller

A keyboard-driven terminal user interface for controlling a multi-room
home audio system. ghac integrates two backend services:

- **Music Player Daemon (MPD)** — library browsing, playlist management,
  and playback control.
- **SnapCast** — synchronized multi-room audio streaming and per-client
  volume control.

## Requirements

- Go 1.25 or later
- A running MPD instance
- A running SnapCast server instance

## Configuration

ghac reads its configuration from `$HOME/.config/.ghacrc` (TOML format).
The file is required; ghac will print a descriptive error and exit if it
is missing or invalid.

A template is included in the repository at `ghacrc.example`. Copy it to
the config location and edit the values to match your setup:

```sh
cp ghacrc.example ~/.config/.ghacrc
```

| Field             | Description                    |
| ----------------- | ------------------------------ |
| `snapserver.ip`   | SnapCast server IP address     |
| `snapserver.port` | SnapCast server port           |
| `mpd.ip`          | MPD server IP address          |
| `mpd.port`        | MPD server port                |
| `theme`           | Optional. Theme name to use (e.g. `"ocean"`). Overridden by `--theme` flag. |
| `[[themes]]`      | Optional. One or more user-defined theme blocks (see `ghacrc.example`). |

## Building

```sh
make build
```

## Running

```sh
./ghac
./ghac --theme ocean   # start with a specific theme
```

ghac reads `~/.config/.ghacrc` on startup. If the config file is missing
or invalid it prints an error to stderr and exits with a non-zero status.

## Usage

### Global Keys

| Key      | Action                              |
| -------- | ----------------------------------- |
| `1`      | Switch to Player Volume screen      |
| `2`      | Switch to Playlist Control screen   |
| `3`      | Switch to Library Navigator screen  |
| `?`      | Toggle help overlay                 |
| `ctrl+t` | Open theme selector                 |
| `p`      | Toggle play / pause                 |
| `z`      | Toggle random (shuffle) mode        |
| `q`      | Quit                                |
| `Ctrl-C` | Quit                                |
| `Esc`    | Close modal (reverts theme if theme selector is open) |

### Screens

**Player Volume (1)** — Adjust volume and mute state for each SnapCast
client.

| Key      | Action                                  |
| -------- | --------------------------------------- |
| `j/k`    | Move cursor down / up                   |
| `h/l`    | Decrease / increase focused volume 5%   |
| `m`      | Toggle mute on focused client           |
| `H/L`    | Decrease / increase all volumes by 5%   |
| `M`      | Toggle mute on all clients              |
| `Ctrl-R` | Rename focused client                   |

**Playlist Control (2)** — View and manage the MPD playback queue.

| Key          | Action                                    |
| ---------------- | ----------------------------------------- |
| `j/k`            | Move cursor down / up                     |
| `gg/G`           | Move cursor to top / bottom               |
| `f <letter>`     | Jump to first entry starting with letter  |
| `Ctrl-D/U`       | Move cursor down / up half a page         |
| `Ctrl-J/Ctrl-K`  | Move song under cursor down / up          |
| `space`          | Toggle selection on song under cursor     |
| `x`              | Remove selected song(s) or song at cursor |
| `X`              | Clear the entire playlist                 |
| `enter`          | Start playing the song under cursor       |

**Library Navigator (3)** — Browse the MPD music library by directory.

| Key          | Action                                     |
| ------------ | ------------------------------------------ |
| `j/k`        | Move cursor down / up                      |
| `gg/G`       | Move cursor to top / bottom                |
| `f <letter>` | Jump to first entry starting with letter   |
| `Ctrl-D/U`   | Move cursor down / up half a page          |
| `h`          | Navigate to parent directory               |
| `l`          | Enter directory under cursor               |
| `space`      | Toggle selection on entry under cursor     |
| `x`          | Remove selected file(s) from playlist      |
| `enter`      | Enqueue selected entries (or cursor entry) |

**Help (?)** — Quick-reference for all keybindings. Appears as a modal
overlay on top of the current screen. Press `?` or `Esc` to close it.

**Theme (ctrl+t)** — Color theme selector. Lists all built-in themes
(`default`, `ocean`, `forest`, `rose`, `mono`, `vampire`, `cyberpunk`,
`matrix`). Use `j`/`k` to preview themes in real time, `Enter` to
confirm, `Esc` to revert. The selected theme is saved across sessions.

## Running Tests

```sh
make test
```

Integration tests (requiring live MPD/SnapCast instances) are gated
behind the `integration` build tag and are not run by default:

```sh
make test-integration
```
