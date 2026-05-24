# Intent

`ghac` (Great Home Audio Controller) is a keyboard-driven terminal UI
for controlling a multi-room home audio system. It targets home users
running [Music Player Daemon (MPD)](https://www.musicpd.org/) and
[SnapCast](https://github.com/badaix/snapcast) who want a single,
fast interface for managing both music playback and room-level audio
distribution.

## Goals

1. **Unified control** — one binary, one config, both backends.
2. **Keyboard-first** — every action reachable without a mouse.
3. **Low overhead** — single static binary, no runtime dependencies
   beyond a running MPD and SnapCast daemon.
4. **Extensible appearance** — built-in themes plus user-defined
   themes via `~/.config/.ghacrc`.
