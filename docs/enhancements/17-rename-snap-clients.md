# Rename SnapCast Client (`n`)

[issue #17](https://github.com/nick-orton/ghac/issues/17)

## Summary

On the Player Volume screen, pressing `ctl-r` opens a modal text-input
window that lets the user rename the focused SnapCast client. The
input field is pre-populated with the client's current name.
`Ctrl-S` commits the rename via the SnapCast `Client.SetName` RPC
call; `Esc` dismisses the modal without making any change. Available
commands are listed at the bottom of the modal window.

---

## Scope

| Screen              | In scope |
| ------------------- | -------- |
| Player Volume       | Yes      |
| Playlist Control    | No       |
| Library Navigator   | No       |
| Help modal          | No       |

---

## Behaviour Specification

1. Press `ctl-r` on the Player Volume screen — a rename modal appears
   centered over the current screen, pre-filled with the focused
   client's current name and the cursor at the end of the text.
2. The modal title is **Rename Client**.
3. The user edits the name using standard terminal input:
   - Printable characters append to / insert into the field.
   - `Backspace` / `Delete` removes the preceding character.
   - Left / Right arrow keys move the cursor within the field.
   - `Home` / `Ctrl-A` moves the cursor to the start of the field.
   - `End` / `Ctrl-E` moves the cursor to the end of the field.
4. Press `Ctrl-S` — if the field is non-empty, sends
   `Client.SetName` to SnapCast with the new name, then closes
   the modal. If the field is empty, `Ctrl-S` is a no-op (the
   modal stays open).
5. Press `Esc` — closes the modal without sending any command.
   The client name is unchanged.
6. While the modal is open, all other key events (volume, mute,
   cursor movement, global screen-switch keys) are swallowed. Only
   `Ctrl-S`, `Esc`, and text-editing keys are processed.
7. The SnapCast server will broadcast a notification after a
   successful rename; the existing `ListenNotifications` /
   `MsgClientsUpdated` path will refresh the display automatically.
8. The bottom of the modal shows a command hint line:
   `Ctrl-S: save   Esc: cancel`
9. `ctl-r` is a no-op when no clients are connected.

### Modal layout

```
┌─ Rename Client ──────────────────┐
│                                  │
│  Name: ClientName_               │
│                                  │
│  Ctrl-S: save   Esc: cancel      │
└──────────────────────────────────┘
```

- Modal width: `min(50, terminalWidth − 4)`, floor of 30.
- The text field renders the current input with a trailing cursor
  character (`_`) at the insertion point.
- The hint line uses `styleHelpDesc` (faint) to keep it visually
  subordinate.

---

## Design Decisions (confirmed)

1. **`Client.SetName` RPC** — SnapCast exposes this method with
   params `{"id": "<clientID>", "name": "<newName>"}`. A new
   `SetName(clientID, name string) error` method is added to
   `internal/snapcast/client.go` following the same pattern as
   `SetVolume`.
2. **Modal owned by `volumeScreen`** — the rename state
   (`showRename bool`, `renameInput string`, `renameCursor int`)
   lives on `volumeScreen`, not the root model. The root model
   checks `volume.showRename` before delegating keys, so global
   keys are still swallowed while the modal is open (same pattern
   as `showHelp`).
3. **Overlay reuse** — the modal is rendered via the existing
   `placeOverlay()` / `modalBorder()` infrastructure in `model.go`.
   `volumeScreen.View()` returns the normal client list;
   `Model.View()` composites the rename modal on top when
   `volume.showRename` is true.
4. **No external text-input library** — the rename input is
   implemented inline in `volumeScreen.Update()` using a simple
   `[]rune` buffer and integer cursor index. The feature does not
   warrant pulling in `charmbracelet/bubbles` for a single field.
5. **Empty name blocked** — `Ctrl-S` does nothing if the input
   field is empty, preventing unnamed clients.

---
