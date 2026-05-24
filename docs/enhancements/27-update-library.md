# Update Library (`U`)

[issue #27](https://github.com/nick-orton/ghac/issues/27)

## Summary

In the Library Navigator, pressing `U` (shift-U) sends the MPD
`update` command scoped to the current directory. This rescans
that directory and its subdirectories for new, changed, or
removed files and updates the MPD database accordingly. The
command is a no-op while a confirmation prompt is pending.

---

## Scope

| Screen             | In scope |
| ------------------ | -------- |
| Player Volume      | No       |
| Playlist Control   | No       |
| Library Navigator  | Yes      |

---

## Behaviour Specification

1. Pressing `U` in the Library Navigator calls the MPD `update`
   command with the current directory path as the URI argument.
2. When the user is at the library root (`currentPath == ""`),
   the update is scoped to the full library (MPD update with an
   empty URI updates everything).
3. Immediately after sending the update command, a transient
   status line reading `Updating library...` appears at the
   bottom of the screen. It is styled italic/faint (same as
   other informational placeholders). It clears automatically
   after 3 seconds.
4. The MPD job ID returned by the update command is discarded.
   MPD notifies clients of database changes via its idle
   subsystem; no special handling beyond what already exists is
   needed.
5. `U` is a no-op while a confirmation prompt is pending
   (consistent with how other keys are swallowed in that state).
6. No confirmation prompt is shown before triggering the update.

---

## Design Decisions (confirmed)

1. **Scoped to current directory** — the update URI is always
   `currentPath`, which restricts the MPD rescan to the subtree
   the user is currently browsing. At root this becomes a full
   library update.
2. **Transient status message** — A 3-second `Updating
   library...` notice confirms the command was sent without
   blocking the UI or requiring a permanent spinner. It uses
   the same italic/faint style as other informational text so
   it is clearly non-interactive.
3. **Key choice `U`** — uppercase U (shift-U) is the canonical
   "update" key in many file managers (e.g. ranger, nnn). It is
   unambiguous and does not conflict with any existing binding.

---
