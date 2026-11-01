# lazynote

[![Build](https://github.com/tkilb/lazynote/actions/workflows/build.yml/badge.svg)](https://github.com/tkilb/lazynote/actions/workflows/build.yml)

A terminal note browser for a folder of markdown notes, written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It is modeled on [lazytask](https://github.com/tkilb/lazytask): a [lazygit](https://github.com/jesseduffield/lazygit)-style panel layout, with a [Yazi](https://yazi-rs.github.io/)-style file browser for the notes.

Notes are plain markdown files with YAML properties, so a notebook opens unchanged in Obsidian or any editor. Notes are organized with [PARA](https://fortelabs.com/blog/para/) (Projects, Areas, Resources, Archives) plus an Inbox for new material, and every note has a **kind** that says how far it can be trusted.

> **Windows is not supported.** Not in CI, not in releases, not in the install script.

## Install

Linux and macOS only.

```bash
curl -fsSL https://raw.githubusercontent.com/tkilb/lazynote/main/scripts/install.sh | bash
```

This downloads the latest release for your OS and architecture, checks its checksum, and installs it to `~/bin` (set `INSTALL_DIR=/some/path` to change that). Make sure the directory is on your `PATH`.

To build from source (Go 1.25 or newer):

```bash
make build      # writes ./lazynote
make install    # builds, then copies it to ~/bin
make test       # go test ./...
```

`lazynote version` prints the version.

## Setup

lazynote reads `~/.config/lazynote/config.yaml`. The only required key is the list of notebooks, each a folder of notes:

```yaml
notebooks:
  - ~/Notes/work
  - ~/Notes/personal
retention:
  daily: 30d        # optional, default 30d
ui:
  mouse: true       # optional, default true
sources:            # optional: the sources Quick Add offers
  - adhoc
  - meeting
  - conversation
kinds:              # optional: restyle kinds or add your own (see Kinds)
  - name: fact
    color: "#2ea043"
  - name: recipe
    color: "#ff8800"
    meaning: Something to cook
    badge: cook
```

- **`notebooks`**: the first available entry opens on the first run, and after that the notebook you used last. A path that is not an existing folder is listed as unavailable and cannot be opened. `~` is expanded.
- **`retention.daily`**: how long archived daily notes are kept, a whole number of days followed by `d`.
- **`ui.mouse`**: set to `false` to turn the mouse off.
- **`sources`**: up to 40 characters each, distinct ignoring case. Without it the list is adhoc, meeting, conversation, email, web, document.

A notebook needs no particular layout. These folders are used and created when first needed:

```
inbox/          new material
  daily/        daily notes (YYYY-MM-DD.md)
projects/
areas/
resources/
archives/
  daily/        archived daily notes
.lazynote/      hidden: the trash (see Syncing)
```

Where lazynote keeps its own state (your position in each notebook, your stickies, the last notebook) is `~/.local/state/lazynote/state.db`, or `$XDG_STATE_HOME/lazynote`, or the folder in `LAZYNOTE_STATE_DIR`. It is outside the notebooks.

## Syncing and version control

**Exclude `.lazynote/` from sync and version control.** Trashing a note with `d` moves it to `.lazynote/trash/` inside the notebook so that `u` can undo it, and that folder is not cleaned up automatically. Sync tools and git would otherwise pick it up. For git, add this line to the notebook's `.gitignore`:

```
.lazynote/
```

lazynote never edits files other than your notes, so it does not add that line for you. Obsidian already hides folders that start with a dot. Everything else lazynote keeps (positions, stickies) lives outside the notebook, so there is nothing else to exclude.

## The screen

| Key | Panel | What it shows |
| --- | --- | --- |
| `1` | **Stickies** | Notes you pinned with `s`. One row (`5 stickies`) until focused, then one row each, up to five. |
| `2` | **PARA** | Five tabs (Inbox, Projects, Areas, Resources, Archives), each a folder browser. Only the selected tab is spelled out. Inside a folder, the first row is its path. |
| `3` | **Related** | What the selected note references and what references it, from its `related` property. |
| `0` | **Body** | The selected note's text, with a cursor line. |
| `4` | **Info** | The note's kind, source, date, tags, aliases and any other properties. |
| | Title | The note's title, led by its kind badge. Not focusable. |

Move between panels with `0`-`4` or `Tab`/`Shift+Tab`. The open notebook's name is shown in the status bar. Press `?` for the full key list of the focused panel.

### Kinds

Every note has one kind, stored in its `kind` property and shown as a badge:

| Kind | Meaning |
| --- | --- |
| `fact` | Accurate information |
| `rumor` | Unconfirmed information |
| `proposal` | Documented desired outcome, which may not reflect reality |
| `fossil` | Is or was true, but may have drifted |
| `idea` | A loose thought |
| `draft` | Incomplete, still being revised |
| `web` | A web page saved by its address (the `source`) |

Add your own kinds, or change how these look, with the `kinds` list in the config file. Each entry needs a `name`; `color` is a hex value such as `#ff8800`, `meaning` is a short description shown in the pickers, and `badge` is up to four characters for the PARA list (an added kind defaults to the first four letters of its name). An entry named like a built-in kind (`fact`, `web` and so on) changes that kind; any other name adds one after the built-in ones. Built-in kinds cannot be removed. A note whose kind is not in the list reads as unset and is left alone, so nothing is lost if you remove one.

## Keys

**Anywhere**

| Key | Action |
| --- | --- |
| `q` / `ctrl+c` | Quit |
| `?` | Help for the focused panel |
| `0`-`4`, `Tab` | Focus a panel |
| `B` | Choose the notebook to open |
| `ctrl+f` | Find a note by path (fuzzy) |
| `ctrl+/` or `F` | Find text in any note (fuzzy); opens it at the line |
| `s` | Pin or unpin the selected note as a sticky |
| `A` | Quick Add a note to the Inbox |
| `I` | Import a file (a transcript or a markdown note) |
| `U` | Clip a web page |
| `o` | Open the web address in the selected note's `source` in the browser |
| `b` / `t` / `T` | Open yesterday's / today's / tomorrow's daily note |
| `u` / `ctrl+r` | Undo / redo |
| `ctrl+l` | Refresh from disk |
| `ctrl+z` | Suspend |

In every picker, type to narrow the list (words separated by spaces must all match), move with `↑`/`↓`, page with `pgup`/`pgdown` or `ctrl+u`/`ctrl+d`, choose with `enter`, cancel with `esc`.

**PARA**

| Key | Action |
| --- | --- |
| `↑` `↓` | Move |
| `←` / `→` or `enter` | Go to the parent folder / open a folder or note |
| `g` `g` / `G` | First / last entry |
| `ctrl+u` `ctrl+d` or `pgup` `pgdown` | Page up / down |
| `[` `]` | Previous / next tab |
| `space` | Select and move down |
| `v` / `V` | Select / unselect a range as you move |
| `ctrl+a` | Select all |
| `y` / `x` | Yank / cut into a buffer that lasts while you move between folders (`Y` / `X` clear it) |
| `p` / `P` | Paste / paste over |
| `d` / `D` | Move to the trash / delete permanently |
| `a` | New note (end the name with `/` for a folder) |
| `r` | Rename, updating references in other notes |
| `K` | Set the kind |
| `m` | Move to a tab: `i` Inbox, `p` Projects, `a` Areas, `r` Resources, `A` Archives |
| `e` | Edit in `$EDITOR` |
| `/` then `n` / `N` | Search names, then step through the matches |
| `.` | Show hidden files |

**Body**: `↑` `↓` move the cursor line, `space` picks lines, `v`/`V` select a range, `ctrl+a` selects all, `y` copies the selection (or the line) to the system clipboard, `e` edits the note in `$EDITOR`, `r` renames it, and `enter` extracts the selection into a new Inbox note, after you choose its kind. **Related**: `enter` jumps to the note, `l` adds a reference, `d` removes one. **Stickies** (focused): `enter` jumps, `d` unpins.

The mouse works too: click a panel to focus it, click a row, tab or Body line to select it, and use the wheel to move. While a popup is open, only the wheel (in lists) and a click to dismiss a notice do anything, so a stray click never discards what you typed. Mouse reporting changes native text selection in most terminals: hold `Shift` (or `Option` on macOS) while dragging to select text.

## Capturing notes

New material lands in the Inbox as a note with a `uid`, `title`, `kind`, `source`, `date` and `tags`.

- **`A` Quick Add** opens a form for the title, body, kind, source and tags.
- **`I` Import** reads a file: a transcript becomes a note holding the text unchanged (kind `rumor`), and a markdown note is copied in with its properties kept.
- **`U` Clip** asks for a web address and saves a `web` note, in the folder PARA is showing, named after the page's title, with the address in `source` and an empty body. `o` later opens it. Only `http` and `https` addresses are fetched, never anything on your own machine or private network.
- **Extract**: in Body, select lines and press `enter` to turn them into a new note that links back to the original.

The same capture works from a shell, without opening the interface. Both commands print the new note's path:

```bash
lazynote add "Water boils" "At 100C at sea level" --kind fact --tags physics
lazynote add --clipboard --source meeting
echo "an idea" | lazynote add --kind idea
lazynote import ~/Downloads/meeting.txt --type transcript
```

They save into the notebook the interface would open. Run `lazynote add -h` or `lazynote import -h` for the flags.

## Daily notes

`b`, `t` and `T` open yesterday's, today's and tomorrow's note in `inbox/daily/`, creating an empty one when the day has none. Digest a daily note (extract the useful lines, then archive it with `m` `A`, which moves it to `archives/daily/`). Archived daily notes are temporary: when a notebook is opened, those older than `retention.daily` are **permanently deleted** and the count is reported. A daily note still in `inbox/daily/` past that age is only shown in orange, never deleted.

## Links, undo and safety

- Notes link with `[[Name]]` entries in a `related` property, resolved by file name. Renaming a note rewrites the links to it.
- Every change to a note is undoable with `u`, except permanent deletion (`D`), the daily purge and editing in `$EDITOR`.
- All file operations stay inside the notebook, and a note with unreadable properties is never rewritten.

## Testing

```bash
go test ./...
```

The tests cover the engine packages (storage, state, undo, config, intake, the command line and the link index). The interface has no automated tests; each change ships with a manual check instead.

## More

`PLAN.md` holds the full design: layout, every key, the decisions behind them and the chunk-by-chunk build history. Releases are cut with `make release` (patch), `make release minor` or `make release major`, which asks before it commits, tags and pushes, and the Release workflow then publishes the binaries.
