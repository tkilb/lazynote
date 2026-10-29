# lazynote - Plan

A TUI note browser in Go with Bubble Tea, modeled on `~/Git/lazytask`
(lazygit-style bordered panels, keyboard-first, `$EDITOR` for editing).

## Layout (from the paper mock)

```
┌[1] Notebook──┐┌[4] Info────────────┐
└──────────────┘└────────────────────┘
┌[2] PARA──────┐┌[0] Body────────────┐
│ Projects│Areas│Resources│Archives   │
│              ││                    │
└──────────────┘│                    │
┌[3] Related───┐│                    │
└──────────────┘└────────────────────┘
```

| Key | Panel    | Content                                                                                                      |
| --- | -------- | ------------------------------------------------------------------------------------------------------------ |
| `1` | Notebook | Active notebook (root directory).                                                                            |
| `2` | PARA     | Tabs: Projects / Areas / Resources / Archives. Each tab is a Yazi-style directory browser rooted at the tab's folder. Cycle tabs with `[` / `]`. |
| `3` | Related  | Notes related to the note selected in PARA; updates as the PARA cursor moves.                                |
| `4` | Info     | Title plus every frontmatter property of the selected note (Obsidian-style: `tags`, `aliases`, custom keys). |
| `0` | Body     | Read-only rendered preview of the selected note.                                                             |

Focus: `0`-`4`, `Tab`, `Shift+Tab`. Mouse support follows lazytask.

## Keybindings (consolidated; keys marked \* are new proposals)

Conventions carried over from lazytask: lowercase acts on the selection,
uppercase is the bulk or heavier variant, destructive actions confirm with
`y`/`enter` and cancel with `n`/`esc`.

**Global**

| Key                 | Action                                        |
| ------------------- | --------------------------------------------- |
| `q` / `ctrl+c`      | Quit                                          |
| `ctrl+z`            | Suspend                                       |
| `0`-`4`             | Focus Body / Notebook / PARA / Related / Info |
| `Tab` / `Shift+Tab` | Cycle panel focus                             |
| `?`                 | Help popup                                    |
| `a`                 | New note in the active PARA tab               |
| `shift+F`           | Quick fact (one-line prompt, `kind: fact`)    |
| `shift+R`           | Quick rumor (one-line prompt, `kind: rumor`)  |
| `i`                 | Quick idea (one-line prompt, `kind: idea`)    |
| `I`                 | Import transcript or note (path prompt)       |
| `shift+U`           | URL clip                                      |
| `t`                 | Append to today's daily note                  |
| `shift+T`           | Append to tomorrow's daily note               |
| `ctrl+t`            | Enter triage mode (deliberate chord)          |
| `u` / `ctrl+r`      | Undo / redo                                   |

**Notebook (1)**

| Key         | Action             |
| ----------- | ------------------ |
| `↑/k` `↓/j` | Select notebook    |
| `enter`     | Switch to notebook |

**PARA (2)** (Yazi parity; see "Yazi parity" below for markers and rules)

| Key              | Action                                                                         |
| ---------------- | ------------------------------------------------------------------------------ |
| `↑/k` `↓/j`      | Move selection; Related, Info and Body follow                                  |
| `←/h`            | Go to the parent directory (no-op at the tab root)                             |
| `→/l`            | Enter the selected directory only; no-op on a note                             |
| `enter`          | Enter the selected directory, or open a note (focus Body, 0)                   |
| `gg` / `G`       | Jump to first / last entry                                                     |
| `ctrl+u` / `ctrl+d` | Half page up / down                                                         |
| `ctrl+b` / `ctrl+f` | Page up / down                                                              |
| `space`          | Toggle selection on the hovered item and move down                             |
| `v` / `V`        | Visual mode: select a range / unselect a range                                 |
| `ctrl+a`         | Select all in the current directory                                            |
| `esc`            | Clear the selection                                                            |
| `y`              | Yank (copy) the selection, or the hovered item if none                         |
| `x`              | Cut the selection, or the hovered item if none                                 |
| `Y` / `X`        | Cancel the yank / cut                                                          |
| `p`              | Paste into the current directory (never overwrites)                            |
| `P`              | Paste, overwriting on name collision (confirm)                                 |
| `d`              | Delete the selection or hovered item to trash (confirm)                        |
| `D`              | Delete permanently (confirm)                                                   |
| `a`              | Create: a name ending in `/` makes a directory, otherwise a note               |
| `r`              | Rename the hovered item                                                        |
| `.`              | Toggle hidden files                                                            |
| `/`              | Find: jump to and highlight matches; `n` / `N` next / previous match           |
| `f`              | Filter: hide entries that do not match the typed text                          |
| `[` / `]`        | Previous / next tab (Projects, Areas, Resources, Archives, Inbox)              |
| `c`              | Cycle kind filter (all, six kinds, unset); clashes with Yazi `c` chords        |
| `shift+K`        | Set kind on the selected note (picker); clashes with Yazi `K` seek             |
| `e`              | Edit in `$EDITOR`                                                              |
| `m` then key     | Move note: `i` inbox, `p` project, `a` area, `r` resource, `A` archive (popup) |

**Related (3)**

| Key             | Action                                           |
| --------------- | ------------------------------------------------ |
| `↑/k` `↓/j`     | Move selection                                   |
| `enter / space` | Jump PARA to the related note                    |
| `*+`            | Add relation (fuzzy picker)                      |
| `d`             | Remove relation (confirm; outgoing entries only) |

**Info (4)**: `↑/k` `↓/j` scroll.

**Body (0)**

| Key         | Action                                                                                           |
| ----------- | ------------------------------------------------------------------------------------------------ |
| `↑/k` `↓/j` | Scroll                                                                                           |
| `ctrl+d/u`  | Half-page down / up                                                                              |
| `*v`        | Start line selection; `enter` extracts the lines into a new linked note (`f` fact, `p` proposal) |

**Triage mode**

| Key          | Action                                    |
| ------------ | ----------------------------------------- |
| `shift+K`    | Set kind                                  |
| `m` then key | Move note (same destination keys as PARA) |
| `*+`         | Add relation                              |
| `s` / `esc`  | Skip to next / leave triage               |

Mouse follows lazytask (click to focus or select, wheel to scroll).

## Yazi parity

PARA mirrors Yazi's file-manager behavior. This was written from memory of
Yazi's default keymap and theme; verify the details marked (verify) against
Yazi's docs before chunk 3d.

Markers: a colored bar at the left edge of each row, as in Yazi.

| State    | Color                 | Set by                          |
| -------- | --------------------- | ------------------------------- |
| Yanked   | green                 | `y`                             |
| Cut      | red                   | `x`                             |
| Selected | yellow (verify)       | `space`, `v`, `ctrl+a`          |

- Yank, cut and delete act on the selection when one exists, otherwise on the
  hovered item.
- The yank or cut buffer persists while you navigate directories and switch
  tabs, so you can cut in Projects and paste in Archives.
- Cutting clears the buffer after `p`. Whether a yank stays pasteable after
  `p` (verify): default is to clear it too.
- Yank and paste copies the file with its frontmatter unchanged. Cut and
  paste moves it.
- `p` never overwrites; on a name collision it reports an error. `P`
  overwrites after a confirm.
- `d` moves items to `.lazynote/trash/` in the notebook root (hidden), so `u`
  can undo it. `D` removes permanently and is not undoable. Both confirm.
- Hidden files (names starting with `.`, including `.lazynote/`) are hidden
  until `.` is pressed.
- All move, copy, rename and trash operations are undoable except `D`.
- Wikilinks resolve by filename, so moves do not break `related` entries
  unless the name collides. Rename does (open decision 8).

Move popup: pressing `m` opens a small popup listing the destinations and
their keys; the next key moves the note immediately with no confirm, and
`esc` cancels. Because the popup is quick, `mp`, `mA` and so on work as a
two-key chord for experienced users. Moves are undoable (`u`). Destinations
are the tab root folders; the note lands at the root of that tab (choosing a
subfolder is open decision 5). The keys are case-sensitive: `a` is area, `A` is archive.

Conflict notes:

- `a` is "new note" globally, as in lazytask, so Related uses `+` for adding.
- `h`/`l` navigate directories in PARA only; other panels use `j`/`k`.
- Related uses `d` for removing a relation (confirm) to stay consistent with
  PARA's delete.
- Inside the move popup `p` means project; in PARA outside the popup `p` is
  paste. The popup captures the next key, so there is no clash.
- Quick capture keys are global: `shift+F` fact, `shift+R` rumor, `i` idea.
  `m` is panel-scoped (PARA and triage).
- Inside the move popup `a` and `A` differ only by case. The popup shows
  both labels clearly, and moving to Archives is undoable.
- `c` filters and `shift+K` sets the kind, in both PARA and triage. Both
  clash with Yazi defaults (open decision 9).
- `a` creates in PARA (as in Yazi), so the Global `a` (new note) behaves the
  same there. Yazi's `D` is permanent delete in PARA, which no longer
  clashes since daily append moved to `t` and `shift+T`.
- In PARA, `enter` drills a directory or opens a note (focus Body), while
  `→`/`l` only drill directories; in Related, `enter` or `space` jumps to
  the note.
- `i` (quick idea) and `I` (import) differ only by case, like `a`/`A` in the
  move popup.

## Decisions

- Stack: Go 1.24, bubbletea, bubbles, lipgloss, testify, YAML config.
- Editing: open in `$EDITOR` (reuse the lazytask `editor` approach).
- Session state (selected notebook, tab, note, focused panel): SQLite via
  `modernc.org/sqlite` (pure Go, no cgo). Verify the dependency before adding.
- Agent instructions live in `AGENTS.md`, never `CLAUDE.md`.
- `ctrl+z` suspends to the shell using Bubble Tea's `tea.Suspend`, and the
  app redraws on `tea.ResumeMsg`. Terminal state must be restored on
  suspend; manual QA covers `ctrl+z` then `fg`.

- Notes store: markdown files under a notebook root containing `projects/`,
  `areas/`, `resources/`, `archives/`, hidden behind a `store.Store`
  interface.
- Frontmatter follows Obsidian's "properties" format: a YAML block delimited
  by `---` at the top of the file. Keys are free-form; `tags` and `aliases`
  are lists. Unknown keys must round-trip unchanged on write, with key order
  preserved (parse into a `yaml.Node`, not a fixed struct).
- Title: Obsidian has no title key, so the title is the filename without
  `.md`. Open: use a `title` property when present? (default: yes).
- Info shows all properties, so no schema is enforced.
- Relations: the `related` frontmatter property, a list of `[[wikilinks]]`
  (for example `related: ["[[Project Alpha]]", "[[Meeting 2026-10-01]]"]`).
  Obsidian reads it natively. Inline body links and shared tags are out of
  scope for now.
- Related panel is driven by the PARA selection. It shows two groups: the
  selected note's `related` entries (outgoing), and notes whose `related`
  lists the selected note (incoming, from a backlink index). Entries that do
  not resolve to a file are shown dimmed as unresolved. Selecting a row in
  Related and pressing `enter` jumps PARA to that note, switching tab and
  entering the note's directory when needed.
- Link resolution matches Obsidian: by filename without `.md`, case-insensitive,
  ignoring `|alias` and `#heading` suffixes. Duplicate names resolve to the
  shortest path.
- PARA directory browsing: each tab is rooted at its folder (`projects/`,
  `areas/`, `resources/`, `archives/`, `inbox/`) and cannot navigate above
  it. Directories list first, then notes, both sorted by name. Directory
  rows show an item count and no kind badge; the kind filter (`c`) hides
  non-matching notes but always shows directories. Each tab remembers its
  own current directory and cursor. The panel title shows the path relative
  to the tab root. With a directory selected, Info shows its name and item
  counts, and Body and Related are empty. `a` creates the note in the
  current directory, quick capture still goes to `inbox/` root, and `d` on
  a directory confirms and warns when it is not empty. `/` searches the
  current directory only.
- Backlink index is built at startup and kept in SQLite (`internal/state`),
  refreshed when file modification times change.

- Kind: every note has one kind, stored in the
  `kind` frontmatter property (a single string, lowercase):

  | Value      | Meaning                                                    |
  | ---------- | ---------------------------------------------------------- |
  | `fact`     | Confirmed information.                                     |
  | `rumor`    | Unconfirmed information.                                   |
  | `proposal` | Documented desired outcomes; may not reflect reality.      |
  | `fossil`   | Is or was true, but may have drifted due to time.          |
  | `idea`     | A loose thought that has not graduated to a proposal.      |
  | `draft`    | A forming idea that is incomplete and still being revised. |

  The set is closed and defined once in `internal/store` as a typed enum.
  A missing or unrecognized value is shown as `unset` and never
  rewritten silently. Matching on read is case-insensitive.

- Kind is visible in three places: a short badge on each PARA row
  (colored per class), a highlighted row at the top of Info, and a filter on
  the PARA list (`c` cycles all / fact / rumor / proposal / fossil / idea /
  draft / unset). The filter is saved in session state.
- Changing a kind in-app (`shift+K` in PARA and triage) opens a picker listing all six kinds and
  writes only the `kind` property, using the round-trip rules above.

## Intake

New material lands in an `inbox/` folder at the notebook root (outside the
four PARA folders) and is triaged into PARA later. Every intake note gets
frontmatter: `kind`, `source` (how it arrived), `captured` (ISO timestamp),
and `tags`. Inbox appears as a fifth PARA tab, shown only when non-empty.

Confirmed:

- Quick capture prompts, each a one-line prompt creating an inbox note:
  `shift+F` for `kind: fact`, `shift+R` for `kind: rumor`, `i` for
  `kind: idea`.
- Import: `I` prompts for a file path (or `lazynote import <file>`), then one
  key chooses `t` transcript or `n` note (the CLI uses `--type`).
  A transcript becomes a note with the text as body, `source: transcript`,
  and `kind: rumor` (raw and unconfirmed until reviewed); the original text
  is kept unmodified. An existing markdown note is copied into the inbox
  with its frontmatter preserved, `source: import` and `captured` added when
  absent, and `kind` left as is (or `unset`).

Also confirmed:

- Shell capture: `lazynote add "text" --kind idea`, and
  `pbpaste | lazynote add --source clipboard` or `lazynote add --clipboard`,
  so capture works from a hotkey or script without opening the TUI.
- URL clip: save a link with title and `source: <url>`, `kind: rumor`.
- Daily note append: `t` adds a timestamped line to today's
  `daily/YYYY-MM-DD.md`, `shift+T` to tomorrow's. The file is created when
  missing.
- Triage mode: step through inbox notes one at a time to set `kind`, pick a
  PARA destination, and add `related` links; moving a note is a file move.
- Extract from transcript: manually promote selected lines of a transcript
  into separate `fact` or `proposal` notes that `related`-link back to it.
  (Automatic extraction with an LLM is out of scope.)

## Open decisions (defaults assumed until answered)

1. (Resolved above: notes store.)
2. Creating a relation in-app. Default: `+` in the Related panel opens a
   fuzzy picker over all notes and appends the choice to the selected note's
   `related` property. Removing: `d` on a Related row (confirm).
3. Fossil drift. Default: `fossil` is set manually; no automatic aging and no
   `verified` date. A later chunk could add a `verified` property and flag
   old facts as fossil candidates.
4. Kind of new notes. Default: `unset` until the user
   chooses.
5. Move into a subfolder. Cut and paste covers it: `x`, navigate to any
   directory in any tab, `p`. Default: no `M` picker; `m` then key stays as
   the quick move to a tab root.
6. Delete target. Resolved: `d` trash (`.lazynote/trash/`, undoable), `D`
   permanent.
7. Yank/copy. Resolved: included (`y`, green marker).
8. Rename (`r`) and links. Renaming a note changes its wikilink target.
   Default: rewrite matching `related` entries in other notes (Obsidian
   behavior), shown in the confirm popup as "updates N notes".
9. Yazi key clashes: `c` (copy-path chords in Yazi, kind filter here), `K`
   (preview seek in Yazi, kind picker here), `D` (permanent delete in Yazi,
   daily append was here, now `t` and `shift+T`), `ctrl+r` (invert selection in Yazi, redo here), `o`
   (open in Yazi, `e` here). Default: keep lazynote's meaning for the rest, and omit
   copy-path chords and `ctrl+r` invert.

## Package layout

- `cmd/lazynote/` - root model, update, mouse, modal box, filter.
- `internal/ui/{panel,popup,helppopup,statusbar}` - shared widgets
  (port from lazytask).
- `internal/ui/{notebook,para,related,info,body}` - one package per panel.
- `internal/store/` - notes backend interface plus markdown implementation.
- `internal/state/` - SQLite session state.
- `internal/{editor,config,undo,version}` - as in lazytask.

## Chunks (one objective each, human checkpoint between chunks)

1. Scaffold: module, Makefile, VERSION, root model, five-panel grid, focus
   cycling, quit, `ctrl+z` suspend and resume.
2. `store.Store` interface plus markdown implementation; tests against a
   temp directory.
   2b. Kind enum, parsing from frontmatter, `unset` fallback;
   unit tests.
3. PARA panel: four tabs, `[`/`]` switching, note list with selection,
   `enter` to focus Body, kind badge per row.
   3b. Kind filter on the PARA list (`c`).
   3c. Directory browsing in PARA: Yazi-style navigation (`h`/`l`/arrows,
   `enter`, `gg`/`G`, half and full page), breadcrumb in the panel title,
   hidden-file toggle (`.`).
   3d. Yazi parity: selection (`space`, `v`, `ctrl+a`), yank/cut/paste with
   green/red/yellow markers, trash and permanent delete, create (`a`),
   rename (`r`), find (`/`, `n`/`N`) and filter (`f`).
4. Body panel: render the selected note.
5. Info panel: title plus all frontmatter properties, rendered generically
   (scalars, lists, nested values), in file order.
6. `internal/state`: SQLite persistence and restore on launch, including
   the current directory for each PARA tab.
7. Notebook panel: switch notebooks.
8. Related panel, read-only: `related` parsing, wikilink resolution,
   backlink index, panel follows PARA selection, `enter`/`space` to jump.
   8b. Related panel, editing: add (fuzzy picker) and remove relations, writing
   the `related` property without disturbing other frontmatter.
9. Actions: set kind (`shift+K` picker), new note, edit in `$EDITOR`, move between PARA tabs, cut/yank/paste, delete/trash, create, rename;
   confirmation popups; undo/redo.
   9b. Intake core: inbox folder and tab, quick fact (`shift+F`), quick rumor
   (`shift+R`), transcript and note import (`I` and `lazynote import`).
   9c. Quick idea (`i`, `kind: idea`).
   9d. Shell capture: `lazynote add` with `--kind`, `--source`, stdin and
   `--clipboard`.
   9e. Triage mode: step through inbox, set `kind`, move to a PARA folder, add
   `related` links.
   9f. Extract from transcript: promote selected lines into linked notes.
   9g. URL clip (title fetch needs a network call; confirm design first).
   9h. Daily note append.
10. Mouse support, help popup, config file.
11. Release tooling and CI (copy from lazytask).

Each chunk: unit tests for pure logic, a manual QA note, `go build ./...`
and `go test ./...` passing, one commit.
