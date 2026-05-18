# Toggle Random (`z`)

[issue #16](https://github.com/nick-orton/ghac/issues/16)

## Summary

Pressing `z` from any screen toggles MPD's random (shuffle) mode
on or off. A small visual cue in the now-playing bar shows
whether random mode is currently active, so the user always knows
the playback state at a glance.

---

## Scope

| Screen             | In scope |
| ------------------ | -------- |
| All screens        | Yes      |

---

## Behaviour Specification

1. `z` is a global keybinding active on every screen (including
   when the help modal is closed; swallowed when help is open).
2. Pressing `z` calls MPD's `random` command to toggle the
   current state: off → on, on → off.
3. The new random state is reflected in the next `MsgPlayerState`
   message received from the MPD idle listener. The idle watcher
   subscribes to the `options` subsystem (in addition to `player`
   and `playlist`); when `options` fires it re-queries MPD status
   and returns a `MsgPlayerState` containing the updated `Random`
   field.
4. The now-playing bar shows `[Z]` to the left of the state icon
   when random mode is on. Nothing is shown when it is off
   (no extra space reserved).
5. The indicator degrades gracefully at narrow terminal widths —
   if there is insufficient space, the indicator is omitted along
   with the progress bar and time.

---

## Design Decisions (confirmed)

1. **Global key `z`** — consistent with `p` (play/pause) as a
   global playback-control key, not bound to a specific screen.
2. **`[Z]` indicator in the now-playing bar** — the bar is always
   visible and is the natural home for persistent playback-state
   indicators. `Z` matches the key used to toggle the mode.
3. **No extra space when off** — omitting the indicator when off
   keeps the bar uncluttered; the presence of `[Z]` is
   self-explanatory.
4. **State sourced from MPD idle (`options` subsystem)** — random
   state is read from MPD `Status()` (the `random` field) on
   every `options` event, so ghac always reflects the true server
   state even if another client changes it.

---
