# Home Audio Client Requirements

## Overview 

The Home Audio Client is a TUI written in Go that allows the user to control a 
multi-room audio setup that uses 
[Music Player Daemon](https://www.musicpd.org/) and 
[SnapCast](https://github.com/snapcast/snapcast).

It allows the user to control the volume of the home’s SnapCast Clients.
It also allows the user to browse their music collection via a file-system 
navigator and manage a playlist

Executable is `ghac`

## Layout

- The application has 4 screens that are accessed with the number keys 1-3.
  - Each screen lists the screen title at the top

- Player Volume - SnapCast Client controller
- Playlist Control - The playlist queue for MPD
- Music Navigator - Filesystem navigator for MPD

## Configuration

- The application is configured through a .ghacrc file in $HOME/.config
- The file is in .toml format
- The file allows the user to set the SnapServer IP and PORT
- The file allows the user to set the MusicPD IP and PORT

## Screens

- At the top of every screen is the current song playing and an indicator of 
  the position in the song.  
- The ‘p’ button from any screen will toggle play/pause. 

### Player Volume

- The Player Volume screen controlls the SnapCast Clients
- Clients are listed 1 per row.  Next to the client name is a bar-graph 
  illustrating the volume by percentage 1-100
- The cursor sits on a row at a time.  Moving between rows is done with 
  ‘j’ and ‘k’ (down and up respectively).
- Color of the client name and bar graph changes to illustrate which one has 
  the ‘cursor’ sitting on it
- Changing volume is done with ‘h’ and ‘l’ (down and up respectively)
- Toggling mute for the current client is done with ‘m’
- Capital letters ‘M’ ‘H’ and ‘L’ allow the muting and volume change for all 
  the clients at once

### Playlist Control

- The playlist control screen controls the musicpd server's playlist
- The current playlist is listed one song per line
- The cursor navigates up and down the list with ‘j’ and ‘k’
- The song selected by the cursor is highlighted
- ’x’ removes the currently selected from the playlist (or multiple if multiple
  selected)
- ‘X’ (capital X) clears the entire playlist.
- ’space’ allows a song to be selected for removal
- ‘enter’ on the currently selected song starts playing that song

### Library Navigator
- The Library Navigator Screen allows the user to navigate the MusicPD library and
  add songs to the playlist
- The Library Navigator screen allows the user to navigate the music directory via
  the file-structure view.  
- The look and feel should approximate a tab in the `nnn` program
- Keys ‘j’ and ‘k’ navigate down and up the navigator
- Key ‘h’ goes up a directory
- Key ‘l’ goes into a subdirectory
- The ‘space’ key selects a song
- ‘The ‘enter’ key enqueues the song at the end of the playlist

### Help Screen
- The help screen is entered with the ‘?’ button from all screens
- The help screen is exited with the ‘esc’ button and goes back to the screen 
  that the help screen was entered from
- The help screen shows all buttons on all screens and what they do.

