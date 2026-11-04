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
digest:             # optional: the agentic import (see Importing with an agent)
  enabled: true     # off by default
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
- **`~/.config/lazynote/templates/daily.md`**: a markdown file to make each new daily note from (`b`, `t` and `T`). It is found by its place, not named in `config.yaml`. The new note keeps the template's properties and text but gets its own `uid`, the day's date and its file name as the title. If the template is an example of one day, with that day in its `date` property, that date and the weekday written after it (such as `(Monday)`) are replaced by the new day's wherever they appear; or write `{{date}}` and `{{weekday}}` where the day goes, and only those are replaced. A daily that exists is never changed. If the template cannot be read, you get a warning and an empty daily.
- **`digest`**: the agentic import, off until `enabled: true`. The other keys are `command` (default `claude`), `model` (default `sonnet`), `timeout` (default `5m`, at most `1h`), `prompt` and `merge_prompt` (the two prompt files, by default `import-prompt.md` and `import-merge-prompt.md` in the config folder). See Importing with an agent.
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
| `0` | **Content** | The selected note's text, with a cursor line. |
| `4` | **Info** | The note's kind, source, date, tags, people and any other properties. |
| | Title | The note's title, led by its kind badge. Not focusable. |

Move between panels with `0`-`4` or `Tab`/`Shift+Tab`. The open notebook's name is shown in the status bar. Press `?` for the full key list of the focused panel. In that list, `/` searches the keys and descriptions (`n` and `N` step through matches).

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
| `link` | A link to a web page, saved by its address (the `source`) |

Add your own kinds, or change how these look, with the `kinds` list in the config file. Each entry needs a `name`; `color` is a hex value such as `#ff8800`, `meaning` is a short description shown in the pickers, and `badge` is up to four characters for the PARA list (an added kind defaults to the first four letters of its name). An entry named like a built-in kind (`fact`, `link` and so on) changes that kind; any other name adds one. The pickers list the kinds in the order of your `kinds` list, followed by any built-in kinds you did not list, in their usual order. Built-in kinds cannot be removed. A note whose kind is not in the list reads as unset and is left alone, so nothing is lost if you remove one.

## Keys

**Anywhere**

| Key | Action |
| --- | --- |
| `q` / `ctrl+c` | Quit |
| `+` | Zoom the focused panel to full screen, or back |
| `?` | Help for the focused panel |
| `0`-`4`, `Tab` | Focus a panel |
| `B` | Choose the notebook to open |
| `ctrl+f` | Find a note by path (fuzzy). Results are ranked: the whole file name, then names that start with what you typed, then names where it starts a word, then names that contain it, then scattered letters in the name, then matches only in a folder; shorter names first. The folders are drawn grey |
| `ctrl+/` or `F` | Search text in every note, as grep -i does: type words, and every line holding all of them, exactly and ignoring case, is listed with its note, line and column and a preview of the note beside it. `↑` `↓` move, `pgup` `pgdown` page, `ctrl+d` `ctrl+u` scroll the preview, `enter` opens the note at the line, `esc` cancels |
| `s` | Pin or unpin the selected note as a sticky |
| `A` | Quick Add a note in the folder PARA shows |
| `I` | Import the clipboard, a web page or a file with the agent; on an `import_` folder, run that import again |
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

**Content**: `↑` `↓` move the cursor line, `space` picks lines, `v`/`V` select a range, `ctrl+d` and `ctrl+u` move half a page (`pgdown` and `pgup` a whole page), `{` and `}` jump between paragraphs, `ctrl+a` selects all, `m` switches between the rendered Markdown view and plain text (the rendered view has a cursor row that keeps your place, and is read-only: `space`, `v`, `V`, `ctrl+a` and `y` leave it for plain text so lines can be picked), `y` copies the selection (or the line) to the system clipboard, `e` edits the note in `$EDITOR`, `r` renames it, and `enter` extracts the selection into a new Inbox note, after you choose its kind. **Related**: `enter` jumps to the note, `l` adds a reference, `d` removes one. **Stickies** (focused): `enter` jumps, `d` unpins.

The mouse works too: click a panel to focus it, click a row, tab or Content line to select it, and use the wheel to move. While a popup is open, only the wheel (in lists) and a click to dismiss a notice do anything, so a stray click never discards what you typed. Mouse reporting changes native text selection in most terminals: hold `Shift` (or `Option` on macOS) while dragging to select text.

## Capturing notes

New material lands in the Inbox as a note with a `uid`, `title`, `kind`, `source`, `date`, `tags`, `people` and `related`.

**Property standard.** Every note follows one order: `uid`, `title`, `kind`, `source`, `date`, `tags`, `people`, then any other properties as they were, then `related` last. lazynote applies it to every note it creates and to every note you view or open. Viewing a note adds any missing standard property (empty, never guessed), sets the title to the file name, puts the properties in order, and writes the file only when something changed. Your other properties and the text are kept, and a note whose properties cannot be parsed is left as it is.

- **`A` Quick Add** opens a form for the title, body, kind, source and tags, and saves the note in the folder PARA is showing (`lazynote add` still saves to the Inbox).
- **`I` Import** hands the clipboard, a web page or a file to an agent, which writes grouped notes (see Importing with an agent).
- **`U` Clip** opens a form with a Title (optional) and a URL. The URL is filled in from your clipboard only when it holds a web address and nothing else. It saves a `link` note in the folder PARA is showing, named after the title you typed or, when you leave it blank, the page's own title (fetched for you), with the address in `source` and an empty body. `o` later opens it. Only `http` and `https` addresses are fetched, never anything on your own machine or private network.
- **Extract**: in Content, select lines and press `enter` to turn them into a new note that links back to the original.

The same capture works from a shell, without opening the interface. The command prints the new note's path:

```bash
lazynote add "Water boils" "At 100C at sea level" --kind fact --tags physics
lazynote add --clipboard --source meeting
echo "an idea" | lazynote add --kind idea
```

It saves into the notebook the interface would open. Run `lazynote add -h` for the flags.

## Importing with an agent

`I` turns a meeting transcript, a Confluence page, a Slack thread, a web page or any UTF-8 text file into a folder of notes. It is off until you set `digest.enabled: true`, which also means you accept that imported content is sent to the `digest.command` (the Claude Code CLI, which uses your own login; lazynote handles no API key).

- **What to give it.** Press `I` and type nothing to import the clipboard, an `http` or `https` address to import a page, or a file path. `Tab` completes a path, `~` works, and a bare name such as `standup` is looked for in `inbox/` first. A line under the box previews what will be read. A page behind a login cannot be fetched: paste its text or save it as a file instead.
- **What comes out.** `inbox/import_YYYY-MM-DD_<topic>/` holds `_source.md` (your text, saved first so nothing is lost), one note for each cluster of related concepts and ideas, and `_actions.md` only when there are action items. Small talk is dropped. Every file has kind `import` (add it under `kinds`, or the files get no kind), a source, the date and suggested tags. The original file is never changed.
- **While it runs.** The status bar shows `Importing…` and the app stays usable. A success message clears itself after five seconds, and the cursor does not move. Sort the notes into PARA yourself.
- **Run it again.** `I` on an `import_` folder, or on a file inside it, re-runs the agent over `_source.md` at once. Every note in the folder except `_source.md` is replaced, and `u` brings the old ones back. If the agent fails, nothing is changed.
- **If the agent fails** the folder stays with its `_source.md` and a message says so; press `I` on it to try again.
- **Large sources.** A source over about 400 KiB is split at speaker turns and messages, each part is digested, and a second pass merges them. If only the merge fails, the notes of the parts are kept side by side with a warning.
- **The prompts are yours.** The two prompt files are written to the config folder on first use and are never overwritten. Each is the whole prompt, including the shape of the answer, so edit with care. Whatever the agent returns is checked, cleaned and size-limited before anything is written, and it can only write inside its own `import_` folder.
- **Good to know.** The CLI also sends your global Claude instructions with every call, so they can shape the notes; say what you want in the prompt files. There is no cancel key: a stuck run stops at `digest.timeout`.

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

The tests cover the engine packages (storage, state, undo, config, intake, the import engine and its source readers, the command line and the link index). None calls a real model or the network. The interface has no automated tests; each change ships with a manual check instead, such as `docs/import-qa.md` for the import.

## More

Releases are cut with `make release` (patch), `make release minor` or `make release major`, which asks before it commits, tags and pushes, and the Release workflow then publishes the binaries.
