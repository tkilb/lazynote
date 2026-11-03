# lazynote - Plan

A TUI note browser in Go with Bubble Tea, modeled on `~/Git/lazytask`
(lazygit-style bordered panels, keyboard-first, `$EDITOR` for editing).

## Layout (from the paper mock)

```
┌[1] Stickies─────────────────────────────────┐┌Title───────────────────────────────┐
└──────────────────────────────────────────────┘└────────────────────────────────────┘
┌[2] PARA──────────────────────────────────────┐┌[0] Body────────────────────────────┐
│ Inbox  Projects  Areas  Resources  Archives  ││                                    │
│                                              ││                                    │
│                                              ││                                    │
└──────────────────────────────────────────────┘│                                    │
┌[3] Related───────────────────────────────────┐│                                    │
│                                              │└────────────────────────────────────┘
│                                              │┌[4] Info────────────────────────────┐
└──────────────────────────────────────────────┘└────────────────────────────────────┘
```

| Key | Panel    | Content                                                                                                                                                                                           |
| --- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `1` | Stickies | Notes you pinned with `s`, for quick jumps. One row of names until focused, then one row each (up to five). See chunk 12.                                                                          |
| `2` | PARA     | Tabs: Inbox (first; contains the `daily/` directory) / Projects / Areas / Resources / Archives. Each tab is a Yazi-style directory browser rooted at the tab's folder. Cycle tabs with `[` / `]`. |
| `3` | Related  | Notes related to the note selected in PARA; updates as the PARA cursor moves.                                                                                                                     |
| —   | Title    | Title of the selected note on one line, led by its kind badge. Display only: no number key, not in the focus cycle. |
| `0` | Body     | Read-only plain-text view of the selected note (frontmatter hidden, lines wrapped).                                                                                                               |
| `4` | Info     | Three standard rows (Kind, Source, Date; Tags; Aliases), then every other frontmatter property of the selected note (custom keys), scrolling. `uid` and `related` are hidden. Sits below Body. |

Focus: `0`-`4`, `Tab`, `Shift+Tab`. Tab goes Stickies, PARA, Related, Body, Info and
back; the Title panel is never focusable. Mouse support follows lazytask.

## Keybindings (consolidated; keys marked \* are new proposals)

Notation: a shifted letter is written as the capital letter (`K`, `F`, `T`),
which is what Bubble Tea reports; `shift+` is used only for non-letter keys
(`Shift+Tab`).

Conventions carried over from lazytask: lowercase acts on the selection,
uppercase is the bulk or heavier variant, destructive actions confirm with
`y`/`enter` and cancel with `n`/`esc`.

**Global**

| Key                 | Action                                        |
| ------------------- | --------------------------------------------- |
| `q` / `ctrl+c`      | Quit                                          |
| `ctrl+z`            | Suspend                                       |
| `0`-`4`             | Focus Body / Stickies / PARA / Related / Info |
| `s`                 | Pin or unpin the selected note as a sticky    |
| `B`                 | Choose the notebook to open (picker)          |
| `ctrl+f`            | Find a note by name anywhere (picker)         |
| `ctrl+/` or `F`     | Find text in any note (picker)                |
| `Tab` / `Shift+Tab` | Cycle panel focus                             |
| `?`                 | Keybindings popup (see "Help popup")          |
| `A`                 | Quick Add: form for a new inbox note          |
| `I`                 | Import transcript or note (path prompt)       |
| `U`                 | URL clip (a web note)                         |
| `o`                 | Open the web address in the note's source     |
| `b`                 | Open yesterday's daily note                   |
| `t`                 | Open today's daily note                       |
| `T`                 | Open tomorrow's daily note                    |
| `u` / `ctrl+r`      | Undo / redo                                   |
| `ctrl+l`            | Refresh from disk                             |

**PARA (2)** (Yazi parity; see "Yazi parity" below for markers and rules)

| Key                 | Action                                                                                    |
| ------------------- | ----------------------------------------------------------------------------------------- |
| `↑` `↓`         | Move selection; Related, Info and Body follow                                             |
| `←`                 | Go to the parent directory (no-op at the tab root)                                        |
| `→`                 | Enter the selected directory, or open a note (focus Body, 0)                              |
| `enter`             | Enter the selected directory, or open a note (focus Body, 0)                              |
| `gg` / `G`          | Jump to first / last entry                                                                |
| `ctrl+u` / `ctrl+d` | Page up / down (`pgup` / `pgdown` do the same; `ctrl+b` and `ctrl+f` are free)            |
| `space`             | Toggle selection on the hovered item and move down                                        |
| `v` / `V`           | Visual mode: select a range / unselect a range                                            |
| `ctrl+a`            | Select all in the current directory                                                       |
| `esc`               | Clear the selection                                                                       |
| `y`                 | Yank (copy) the selection, or the hovered item if none                                    |
| `x`                 | Cut the selection, or the hovered item if none                                            |
| `Y` / `X`           | Cancel the yank / cut                                                                     |
| `p`                 | Paste into the current directory (never overwrites)                                       |
| `P`                 | Paste, overwriting on name collision (replaced items go to trash; confirm)                |
| `d`                 | Delete the selection or hovered item to trash (confirm)                                   |
| `D`                 | Delete permanently (confirm)                                                              |
| `a`                 | Create: a name ending in `/` makes a directory, otherwise a note                          |
| `r`                 | Rename the hovered item                                                                   |
| `.`                 | Toggle hidden files                                                                       |
| `/`                 | Search names in the directory: jump to the first match (`esc` clears the search)          |
| `n` / `N`           | Next / previous search match, wrapping around                                             |
| `[` / `]`           | Previous / next tab (Inbox, Projects, Areas, Resources, Archives)                         |
| `K`                 | Set kind on the selected note (picker); clashes with Yazi `K` seek                        |
| `e`                 | Edit in `$EDITOR`                                                                         |
| `m` then key        | Move note: `i` inbox (`inbox/`), `p` project, `a` area, `r` resource, `A` archive (popup) |

**Related (3)**

| Key             | Action                                           |
| --------------- | ------------------------------------------------ |
| `↑` `↓`     | Move selection                                   |
| `enter`         | Jump PARA to the related note                    |
| `*l`            | Add reference (fuzzy picker of notes)            |
| `d`             | Remove relation (confirm; References entries only)|

**Info (4)**: `↑` `↓` scroll.

**Body (0)**

| Key         | Action                                                                                       |
| ----------- | -------------------------------------------------------------------------------------------- |
| `↑` `↓`     | Move the cursor line (the view follows) |
| `ctrl+d/u`  | Page down / up (`pgdown` / `pgup` do the same) |
| `g` / `G`   | First / last line |
| `space`     | Pick the cursor line (cherry-pick) and move down |
| `v` / `V`   | Select / unselect a range as the cursor moves (the same key again closes it) |
| `ctrl+a`    | Select all |
| `y`         | Yank: copy the selection, or the cursor line, to the system clipboard |
| `enter`     | Extract the selection into a new note (kind picked in the picker) |
| `r`         | Rename the note shown, exactly as `r` does in PARA (focus stays in Body, and the cursor line is kept) |
| `e`         | Edit the note shown in `$EDITOR`, exactly as `e` does in PARA (focus stays in Body) |
| `esc`       | Close an open range; then clear the selection; then return focus to PARA (2) |
| `←`         | Return focus to PARA (2) (closing a range or clearing the selection first, like `esc`) |

Mouse follows lazytask (click to focus or select, wheel to scroll); the
details are in chunk 10.

## Yazi parity

PARA mirrors Yazi's file-manager behavior. This was written from memory of
Yazi's default keymap and theme; verify the details marked (verify) against
Yazi's docs before chunk 3d.

Markers: a colored bar at the left edge of each row, as in Yazi.

| State    | Color           | Set by                 |
| -------- | --------------- | ---------------------- |
| Yanked   | green           | `y`                    |
| Cut      | red             | `x`                    |
| Selected | yellow (verify) | `space`, `v`, `ctrl+a` |

- Selection belongs to a tab's current directory: leaving the directory
  clears it, so a bulk action never touches items you cannot see. A reload
  keeps the names that still exist. In visual mode (`v` select, `V` unselect;
  the same key again leaves it, the other key switches) the range between
  the anchor and the cursor follows the cursor. `esc` leaves visual mode
  keeping the selection; `esc` outside visual mode clears it. `space` is
  ignored in visual mode. The panel footer shows `VISUAL`/`UNSELECT`, the
  count (`3 sel`) and the position.
- Yank, cut and delete act on the selection when one exists, otherwise on the
  hovered item.
- The yank or cut buffer persists while you navigate directories and switch
  tabs, so you can cut in Projects and paste in Archives.
- Both yank and cut clear the buffer after `p` or `P` (decided; Yazi keeps a
  yank pasteable). A new `y` or `x` replaces the buffer and clears the
  selection. `Y` and `X` both empty the buffer. A yanked row shows a green
  bar and a cut row a red one; both outrank the yellow selection bar, and
  the footer shows `N yanked` or `N cut` on every tab.
- Yanking an item and pasting it into the directory it already lives in
  makes a duplicate named with the first free `_N` suffix before the
  extension (`a.md` becomes `a_1.md`, then `a_2.md`; a folder `p` becomes
  `p_1`), with no prompt, for both `p` and `P`.
- A paste is all or nothing: if any other name collides, `p` cancels the whole
  paste and names the count; if a step fails, the steps already done are
  undone. Cutting into the directory the item is already in does nothing.
  Sources that no longer exist are dropped and counted in the message.
  `P` asks first (a confirm popup) only when something would be replaced.
- Yank and paste copies the file, but every note in the copy (the one file, or
  each `.md` inside a copied folder) gets a fresh `uid` and its `title` set
  to its file name; other properties and the body are untouched, and other
  files are copied verbatim. A note whose frontmatter cannot be parsed is
  copied byte for byte and a warning popup says it kept its source's `uid`.
  Cut and paste moves it and keeps the `uid`.
- `p` never overwrites; on a name collision it reports an error. `P`
  overwrites after a confirm: each replaced file or directory goes to
  `.lazynote/trash/` first, so `u` restores it, and the overwrite and the
  paste undo together as one operation.
- `a` opens an input popup. A note name gets `.md`
  appended unless it already ends with it; a name ending in `/` makes a
  folder; any other `/`, empty names, `.` and `..` are rejected. A new note
  holds frontmatter with a fresh `uid` and its `title` (kind stays unset). The cursor
  moves to the new item, and `u` removes it again (to the trash).
- `d` and `D` ask first with a confirm popup (`y`/`enter` confirms,
  `n`/`esc` cancels; both are drawn in red, and `D` also says it cannot be
  undone). For a
  folder the question gives the item count and how many items are not shown,
  counted from disk, so attachments and hidden files are included. A
  multi-item `d` is one undo entry and undoes the earlier items if a later
  one fails. `D` stops at the first failure and reports how many were
  removed.
- `d` moves items to `.lazynote/trash/` in the notebook root (hidden), so `u`
  can undo it. `D` removes permanently and is not undoable. Both confirm.
- Hidden files (names starting with `.`, including `.lazynote/`) are hidden
  until `.` is pressed.
- Every change lazynote makes to a note is undoable except `D` and the
  automatic daily retention purge (see the Undo decision).
- Wikilinks resolve by filename, so moves do not break `related` entries
  unless the name collides. Rename rewrites them (chunk 8c).

Move popup: pressing `m` opens a small popup listing the destinations and
their keys; the next key moves the note immediately with no confirm, and
`esc` cancels. Because the popup is quick, `mp`, `mA` and so on work as a
two-key chord for experienced users. Moves are undoable (`u`). Destinations
are the tab root folders; the note lands at the root of that tab (choosing a
subfolder is open decision 5). The keys are case-sensitive: `a` is area, `A` is archive. A note in
`inbox/daily/` named `YYYY-MM-DD.md` is routed to `archives/daily/` by `A`,
so the retention purge applies to it.

## Help popup

`?` opens a scrollable "Keybindings" popup ported from lazytask, with three
sections as in lazygit: **Local** (every key of the focused panel, a fuller
list than the status bar hints), **Global** (keys that work in any panel:
undo, redo, refresh, suspend, help, quit) and **Navigation** (`0`-`4`, `Tab`,
`Shift+Tab`). A panel with no keys of its own has no Local section. The up and down arrows
scroll, `esc` or `?` closes, other keys are ignored. It lists
only keys that are implemented, so each chunk that adds a key adds it to the
lists in `cmd/lazynote/help.go` (and the status bar hints in `model.go`).

Conflict notes:

- `a` creates a note or directory in PARA only (as in Yazi); it is not a
  global key. Related uses `l` for adding a reference; `space` is not used
  for it, and `+` is reserved for panel zoom.
- `←`/`→` navigate directories in PARA only; other panels use `↑`/`↓`.
- Movement uses the arrow keys only. `h`, `j`, `k` and `l` are not bound (the
  project owner uses arrows on a split keyboard), so they are free to be
  assigned to actions.
- Related uses `d` for removing a relation (confirm) to stay consistent with
  PARA's delete.
- Inside the move popup `p` means project; in PARA outside the popup `p` is
  paste. The popup captures the next key, so there is no clash.
- Quick Add (`A`) and import (`I`) are global. `m` is panel-scoped (PARA).
- Inside the move popup `a` and `A` differ only by case. The popup shows
  both labels clearly, and moving to Archives is undoable.
- `K` sets the kind in PARA and clashes with Yazi's `K` seek (open
  decision 9). `/` is a search that jumps and highlights, as in lazytask, never a filter; `n`/`N`
  step through its matches.
- Yazi's `D` is permanent delete in PARA, which does not clash since daily
  opening a daily note moved to `b`, `t` and `T`.
- In PARA, `enter` and `→` drill a directory or open a note (focus
  Body); `←` goes to the parent directory, and in Body `←` or `esc`
  returns focus to PARA. In Related, `enter` jumps to the note.
- Global `A` (Quick Add) is not the same as PARA's `a` (new note in the
  current folder, by name only); `A` is also a destination key inside the
  modal move popup, which captures the next key, so there is no clash.

## Decisions

- Notifications follow lazytask: errors, warnings and informational results
  appear as modal popups (`internal/ui/popup`, ported), queued and shown
  one at a time, each dismissed (and the key consumed) by any key, with a
  colored title: Info blue, Warning orange, Error red. Every popup (these,
  confirmations, input prompts) draws its title in the top border and its key
  hint in the bottom border, as the panels, the picker and the help popup do. Confirmations are
  popups too (orange; red for trash and delete), and text
  prompts are input popups. Routine successes (yank, cut, paste, create,
  trash, delete) show no popup because the list and markers already show
  them; undo and redo always report, including "Nothing to undo". The
  status bar shows only key hints.

- Stack: Go 1.25, bubbletea, bubbles, lipgloss, testify, YAML config.
- Clipboard (`lazynote add --clipboard`): `github.com/atotto/clipboard`, the
  same library lazytask already uses (indirectly, through Bubbles). It shells
  out to `pbpaste` on macOS and to `xclip`, `xsel` or `wl-paste` on Linux, so
  a missing tool must produce a clear error that names the tool. Checked:
  v0.1.4 (2021-02-24) is the latest release, a tiny stable library already
  compiled in through Bubbles' text inputs; `lazynote` now imports it
  directly, with no new download.
- Fuzzy matching (relation picker): `github.com/sahilm/fuzzy` v0.1.1 (added in chunk 8b).
  Approved by you. Checked before use: last release 2023-05-30 (no newer
  version), one 254-line file importing only `sort`, `unicode` and
  `unicode/utf8`, so it does no I/O and no network at run time. The download
  was a one-time network fetch you approved. The picker is
  `internal/ui/picker`.
- Editing: open in `$EDITOR` (reuse the lazytask `editor` approach).
- Session state (selected notebook, tab, note, focused panel): SQLite via
  `modernc.org/sqlite` (pure Go, no cgo). Verify the dependency before adding.
  The database lives at `$XDG_STATE_HOME/lazynote/state.db`, falling back to
  `~/.local/state/lazynote/state.db`; `LAZYNOTE_STATE_DIR` overrides the
  directory and every test sets it to a temp directory. The link index in it
  is a rebuildable cache.
- Agent instructions live in `AGENTS.md`, never `CLAUDE.md`.
- `ctrl+z` suspends to the shell using Bubble Tea's `tea.Suspend`, and the
  app redraws on `tea.ResumeMsg`. Terminal state must be restored on
  suspend; manual QA covers `ctrl+z` then `fg`.

- Undo: `u` and `ctrl+r` cover every change lazynote makes to a note, except
  permanent deletion (`D`) and the daily retention purge.
  - Created notes (quick capture, `a`, import, extract): undo moves the new
    note to trash.
  - Move, copy, paste, rename, trash and overwrite: undo reverses them.
  - Kind changes and relation edits: undo restores the previous frontmatter.
  - Daily append: undo removes the appended line.
  - One user action is one undo entry, so a multi-item paste or a rename with
    its link rewrites undoes together.
  - History lives in memory for the session only, and a new action clears the
    redo stack.
  - If a file changed on disk after the action (an external edit), undo
    refuses for that file and reports it, instead of overwriting the edit.
- Config: `~/.config/lazynote/config.yaml`. Notebooks are an array of root
  paths:

  ```yaml
  notebooks:
    - ~/Notes/work
    - ~/Notes/personal
  retention:
    daily: 30d
  ui:
    mouse: true
  sources:
    - adhoc
    - meeting
    - conversation
  kinds:
    - name: fact
      color: "#2ea043"
    - name: recipe
      color: "#ff8800"
      meaning: Something to cook
      badge: cook
  ```

  `sources` (optional) lists the sources Quick Add offers; the first is the
  default. When it is absent or empty the built-in list is used: adhoc,
  meeting, conversation, email, web, document (`config.DefaultSources`).
  Each entry is trimmed and must be non-empty, free of control characters, at
  most 40 characters and distinct from the others ignoring case; a bad entry
  is a config error that names it.

  `kinds` (optional) adds kinds to the built-in seven or changes how they
  look (see the Kind decision and chunk 13).

  The first entry is the default notebook. The display name is the
  directory's base name. `~` is expanded. Each path must be an existing
  directory; a missing path is listed as unavailable and cannot be selected,
  and an empty or absent `notebooks` list shows an error with this example
  and exits. `B` lists the entries in a picker and `enter` switches
  between them. Session state remembers the last notebook used and keeps
  per-notebook state (tab, directory, cursor, link index) separate.
- Notes store: markdown files under a notebook root containing `projects/`,
  `areas/`, `resources/`, `archives/` (with `archives/daily/` for digested dailies), `inbox/` (with
  `inbox/daily/` inside it) and the hidden
  `.lazynote/` (trash), behind a `store.Store` interface.
  `.lazynote/` lives inside the notebook, so sync tools or git may pick it up.
  Lazynote does not edit files outside notes (no automatic `.gitignore`
  changes); the README documents that you should exclude `.lazynote/` from
  sync and version control. Obsidian already hides dot-folders.
- Missing folders: a notebook may lack any of the tab roots (`inbox/`,
  `projects/`, `areas/`, `resources/`, `archives/`) or the nested `daily/`
  folders. A missing root shows as an empty tab, never an error, and browsing
  creates nothing. A folder is created on the first write that needs it (for
  example the first paste, quick capture, `a` or daily append), through the
  path-safety function below. `.lazynote/` is created the same way, on the
  first trash.
- Path safety: every file operation (create, rename, paste, move, trash,
  delete, import, `lazynote add`, retention purge) goes through one function
  in `internal/store` that resolves the target path and confirms it stays
  inside the notebook root. It rejects `..` segments, absolute paths, empty
  names, NUL bytes and symlinks that resolve outside the root, and compares
  paths on a directory boundary, not a bare string prefix. In the `a` and
  `r` prompts, the only `/` allowed is a trailing one on `a` to make a
  directory. Import may read a file from anywhere, but only as a regular
  file read, and always writes inside the root. Rejections show a short
  error in the status bar without echoing full paths. The function has
  table-driven tests for each rejection case, including symlink escapes.
- External changes: files may change under lazynote (Obsidian, sync, an
  editor). It refreshes from disk after `$EDITOR` returns, when the terminal
  regains focus (Bubble Tea focus reporting, `tea.WithReportFocus`; not every
  terminal sends it), and on `ctrl+l`. There is no live file watcher. A
  refresh reloads the current directory listing, the selected note (Body and
  Info) and the link index for files whose modification time changed. The
  cursor stays on the same file name; if that file is gone, it stays at the
  same row. If the directory you are in has vanished, the panel falls back
  to the nearest ancestor that still exists (dropping its selection and
  search). Other tabs reload when next shown. A refresh waits while a
  confirmation is open, so the question always matches the rows it was
  asked about. A cut or yank entry whose file has vanished is dropped and
  reported when pasting.
- Frontmatter follows Obsidian's "properties" format: a YAML block delimited
  by `---` at the top of the file. Keys are free-form; `tags` and `aliases`
  are lists. Unknown keys must round-trip unchanged on write, with key order
  preserved (parse into a `yaml.Node`, not a fixed struct).
- Title: the title is always the file name without `.md` (any case of the
  extension), and the `title` frontmatter property must always equal it.
  The Title panel shows the file name, never the property. Every note
  lazynote writes carries `title` set to its file name: blank notes from `a`
  and every other creation path, and a copy that gets a new name (such as
  the `_1` duplicate) has its `title` rewritten, other properties and the
  body untouched (a note with no frontmatter gets a block holding the
  title; one whose frontmatter cannot be parsed is left exactly as it is).
  Rename (chunk 8c) updates it too, in the same undo entry. lazynote never
  rewrites a note on read just to fix its title, so a note changed outside
  the app (for example renamed in Obsidian) can disagree until lazynote next
  writes it; Info then shows the differing `title` property, and hides it
  when it equals the file name. Editing in `$EDITOR` is the exception that
  goes the other way: when the editor returns and the `title` property was
  changed, the file is renamed to match it (see "Title follows the editor").
- Uid: every note carries a stable `uid` property, a random UUIDv4
  generated with the standard library (`crypto/rand`, no new dependency). It
  never changes, including across rename and move, and the backlink index
  keys on it. Info does not display it. A note without a `uid` gets one the
  first time lazynote writes it for any reason (create, import, rename, kind change, relation edit);
  lazynote never rewrites an untouched note just to add one.
- Info shows the kind and all properties except `uid`, so no
  schema is enforced. The title has its own panel above Body.
- Relations: the `related` frontmatter property, a list of `[[wikilinks]]`
  (for example `related: ["[[Project Alpha]]", "[[Meeting 2026-10-01]]"]`).
  Obsidian reads it natively. Inline body links and shared tags are out of
  scope for now.
- Related panel is driven by the PARA selection. It shows two groups: the
  selected note's `related` entries (**References**), and notes whose `related`
  lists the selected note (**Referenced by**, from a backlink index). These two
  names are used everywhere, in the UI, the code (`Index.References`,
  `Index.ReferencedBy`) and the docs. Entries that do
  not resolve to a file are shown dimmed as unresolved; they are never
  created from Related, and `enter` on one does nothing except report it as
  unresolved in the status bar. Body always shows the current PARA
  selection. Selecting a row in
  Related and pressing `enter` jumps PARA to that note, switching tab and
  entering the note's directory when needed.
- Link resolution matches Obsidian: by filename without `.md`, case-insensitive,
  ignoring `|alias` and `#heading` suffixes. Duplicate names resolve to the
  shortest path.
- PARA directory browsing: each tab is rooted at its folder (`projects/`,
  `areas/`, `resources/`, `archives/`, `inbox/`) and cannot navigate above
  it. Directories list first, then notes, both sorted by name. Directory
  rows show an item count and no kind badge. Each tab remembers its
  own current directory and cursor. Inside a folder, the first row of the
  panel is the path of the folders below the tab root, `alpha/notes/`
  (with a trailing slash; the tab's name is in the title), in the folder
  color (cyan, as folder rows are): the one you are in bold, those above
  plain, and the slashes dim. At the tab root there is no path row and the list gets the full height. The
  path is cut from the left when too long (`…ha/notes/2026`) so the
  current folder always shows. The row takes one list row while it is there
  (a page is one row shorter, and a click on it only focuses the panel). The
  bottom border keeps the search text (`search: text`) and the position.
  (Tried to make drilling
  obvious; it is one commit, easy to revert.) With a directory selected, Info shows its name and item
  counts, and Body and Related are empty. `a` creates the note in the
  current directory, quick capture still goes to the `inbox/` root, and `d` on
  a directory confirms and warns when it is not empty. `/` searches the
  current directory, as lazytask's `/` does: the list is never filtered. `/`
  opens an inline prompt at the bottom of the screen in place of the key
  hints; `enter` jumps to the nearest row at or after the cursor whose name
  (shown without `.md`; directories included) contains the text, ignoring
  case, wrapping around, and leaves the query active so `n` and `N` step to
  the next and previous match (wrapping; no-ops without a query). The
  matching text is highlighted, teal on the cursor row and yellow on other
  rows, and the footer shows `search: text`. An empty `enter` or a query
  with no match is a silent no-op, and `esc` in the prompt clears the
  query. The search clears when you leave the directory (it survives tab
  switches). Outside the prompt `esc` undoes the innermost thing first:
  visual mode, then the search, then the selection.
- File scope: listings show directories and `.md` files only. Attachments
  (images, PDFs, and so on) are not listed but are left untouched. Names
  starting with `.` stay hidden until `.` is pressed, as in Yazi, but
  `.obsidian/` is always excluded from listings, the link index and
  retention. Because attachments are invisible, the "not empty" check before
  deleting or trashing a directory counts every entry on disk, not just the
  listed ones, and the confirm popup says how many unlisted files it holds.
  Cutting and pasting a directory moves everything inside it.
- Info shows, for the note under the cursor, three standard rows that sit in
  the same place for every note, drawn even when a field is empty: row 1 holds
  `Kind` (highlighted in the badge color, `unset` when missing or
  unrecognized), `Source` and `Date` as three equal columns; row 2 is `Tags`;
  row 3 is `Aliases`. Each is one line and a long value is cut with an
  ellipsis. `Date` is shown as `YYYY-MM-DD` with any time of day dropped
  (`store.DateOnly`); a value that is not a date is shown as written. Below
  them come all other properties in file order, flowing into columns (as
  many as fit, 24 columns wide at least, up to three): a row whose label and
  value fit on one line within a column sits beside others, and a row that
  does not fit (a long or multi-line value) takes the full width on its own
  lines, wrapped. A scalar shows its text; a list of scalars is one
  comma-separated line (`tags  work, ideas`); other lists show one `- ` item
  per line; a mapping shows indented `key: value` lines. Depth and length are
  bounded (400 lines, 8 levels), and all text is sanitized like Body. Hidden:
  `uid`, `related` (the Related panel shows it), `title` when it equals the
  file name, and `kind` when its value is recognized (an unrecognized value
  stays visible as a property). Invalid frontmatter shows a red `Properties`
  row after the standard rows. For a folder, Info shows its name and the
  number of entries it lists; a note that cannot be read shows the error.
  `↑` `↓` scroll, with a position footer when it overflows, and `esc` or
  `←` returns to PARA. The right column, top to bottom, is the Title panel
  (3 rows), Body, and Info, which is fixed at 3 content rows (5 with the
  border) and scrolls by line; Body gets the rest.
- Body shows the note under the PARA cursor as plain text and follows it as
  the cursor moves; with a directory selected it is empty, and a note that
  cannot be read shows the error (for example "note is too large"). The
  frontmatter is hidden, blank lines around the text are dropped, lines wrap
  at word boundaries (a word longer than the panel is split), tabs become
  four spaces, and a position footer (`first line/total`) shows once the
  note is longer than the panel. Body has a cursor line, highlighted while the
  panel has focus, that moves with `↑` `↓` by source line (a wrapped line is
  one line) and with `ctrl+d`/`ctrl+u` or `pgdown`/`pgup` by a page and keeps your place: the view follows it, the
  footer shows `cursor line/total lines`, and the cursor of each note is
  remembered for the session, so coming back to a note puts you where you
  were. A note is read again only when the selection or its modification
  time changes; the cursor survives a refresh of the same note. Lines are
  selected as in PARA: `space` picks single lines, `v`/`V` select or unselect a
  range, `ctrl+a` selects all, selected lines are highlighted and the footer
  shows `VISUAL`, `UNSELECT`, `N sel` and the number of groups of adjacent
  lines. `y` yanks and `enter` extracts (see chunk 9d). Note text is untrusted: every control
  character, including escape and the C1 range, is removed before drawing,
  so a note can never send terminal escape sequences.
- Backlink index is built at startup and kept in SQLite (`internal/state`),
  refreshed when file modification times change.

- Kind: every note has one kind, stored in the
  `kind` frontmatter property (a single string, lowercase):

  | Value      | Meaning                                                    |
  | ---------- | ---------------------------------------------------------- |
  | `fact`     | Accurate information.                                      |
  | `rumor`    | Unconfirmed information.                                   |
  | `proposal` | Documented desired outcomes; may not reflect reality.      |
  | `fossil`   | Is or was true, but may have drifted due to time.          |
  | `idea`     | A loose thought that has not graduated to a proposal.      |
  | `draft`    | A forming idea that is incomplete and still being revised. |
  | `web`      | A web page saved by its address, which is the `source` property (`o` opens it). Not offered when extracting lines. |

  The built-in set is defined once in `internal/store` as a typed enum, and the
  config can add kinds to it and restyle the built-in ones (chunk 13).
  A missing or unrecognized value is shown as `unset` and never
  rewritten silently. Matching on read is case-insensitive.

- Kind is visible in three places: a short badge on each PARA row (colored per
  class), the same badge as a colored pill before the title in the Title
  panel, and a highlighted row at the top of Info.
- Changing a kind in-app (`K` in PARA) opens a picker listing all seven kinds and
  writes only the `kind` property, using the round-trip rules above.

## Intake

New material lands in `inbox/`, shown as the Inbox tab (outside the four
PARA folders), and is sorted into PARA later from the Inbox tab. Every intake note gets
frontmatter: `kind`, `source` (how it arrived), `date` (the day it arrived, `YYYY-MM-DD`),
and `tags`. Inbox is the first tab and is always shown. Daily notes are not
a tab: they live in `inbox/daily/`, which appears as an ordinary directory,
listed first like any directory, next to the intake notes.

Confirmed:

- Quick Add (`A`): one form with Title, multi-line Body, Kind and Tags boxes
  (see the 9b notes) creating an inbox note. The file name is the title, cleaned (unsafe characters
  removed), plus `.md`, with no timestamp (the day is the `date` property);
  a numeric suffix resolves a collision. The body field is the note body;
  the title is the file name, as for every note. The same naming applies to
  `lazynote add`, extracted notes and URL clips (named from their own
  title) and to imported transcripts (named from the file).
- Import: `I` prompts for a file path (or `lazynote import <file>`), then one
  key chooses `t` transcript or `n` note (the CLI uses `--type`).
  A transcript becomes a note with the text as body, `source: transcript`,
  and `kind: rumor` (raw and unconfirmed until reviewed); the original text
  is kept unmodified. An existing markdown note is copied into the inbox
  with its frontmatter preserved, `source: import` and `date` added when
  absent, and `kind` left as is (or `unset`).

Also confirmed:

- Shell capture: `lazynote add "Title" "body" --kind idea`, and
  `pbpaste | lazynote add --source clipboard` or `lazynote add --clipboard`,
  so capture works from a hotkey or script without opening the TUI.
- URL clip (`U`, chunk 9f): save a link as a web note, named after the
  page title, with `kind: web` and `source: <url>`; the body is empty. It is
  saved in the folder PARA is showing. `o` opens the address in the browser. See chunk 9f for the rules.
- Daily retention: when a notebook is first opened in a session (at startup for the
  active one, on first switch for others), lazynote
  permanently deletes files in `archives/daily/` named `YYYY-MM-DD.md` whose date is
  `retention.daily` or more before today. The period comes from config
  (default `30d`; the format is a positive whole number followed by `d`
  for days, and anything else is a config error). It never touches other folders or
  other filenames, resolves paths within the notebook root, and reports the
  count in the status bar. Permanent deletion is not undoable.
- Daily notes: `b`, `t` and `T` open yesterday's, today's and tomorrow's
  daily note (they do not append). PARA jumps to `inbox/daily/YYYY-MM-DD.md`
  and takes focus. Nothing is typed into the note: edit it with `e`. When the
  day has no note in `inbox/daily` or `archives/daily`, an empty one is
  created in `inbox/daily` (uid, title, date, source `daily`, no kind); undo
  removes it.
- Sorting the inbox: there is no separate triage mode. Work through the
  Inbox tab with the ordinary PARA keys: `K` to set a kind, `m` then a
  key to file a note, the add-relation key in Related, and Body line
  selection (`v`, `enter`) to extract. If a queue or mode is needed later, it
  will be added then.
- Daily notes in `inbox/daily/` are digested, not filed: extract the useful
  lines into new notes, then archive the daily with `m` then `A`. On a daily
  note `K` does not apply and `m` accepts only `A`. A daily older than
  `retention.daily` that has not been archived is marked overdue in the
  Inbox tab (a distinct row color). It is never deleted automatically; only
  archived dailies are purged.
- Daily notes are working scratch space. Once digested, `m` then `A` moves
  the daily note to `archives/daily/` (undoable, no confirm). Archived dailies are temporary: they are deleted
  automatically once they reach `retention.daily` (see Daily retention).
- Extract: manually promote selected lines of a note into separate notes,
  choosing each note's kind in the picker. Extracted notes are created in
  `inbox/` and show up in the Inbox tab. From a transcript they `related`-link
  back to it. From a daily note they must not link back, because the daily
  note will eventually be deleted; they record `source: daily YYYY-MM-DD` as plain
  text instead. Manual extraction is chunk 9d. Agentic extraction from a
  transcript is chunk 14.

## Open decisions (defaults assumed until answered)

Numbers are kept stable; resolved items were removed.

3. Fossil drift. Default: `fossil` is set manually; no automatic aging and no
   `verified` date. A later chunk could add a `verified` property and flag
   old facts as fossil candidates.
4. Kind of new notes. Default: `unset` until the user
   chooses.
5. Move into a subfolder. Cut and paste covers it: `x`, navigate to any
   directory in any tab, `p`. Default: no `M` picker; `m` then key stays as
   the quick move to a tab root.
9. Yazi key clashes. Current clashes and their resolution: `K` (preview seek
   in Yazi, kind picker here), `ctrl+r` (invert selection in Yazi, redo here)
   and `o` (open in Yazi, `e` edits here). Default: keep lazynote's meaning
   and omit the Yazi behavior.
11. Trash retention. Items trashed with `d` sit in `.lazynote/trash/` and
    accumulate. Default: no automatic purge yet.
12. Digester backend and privacy (chunk 14). Default: run the Claude Code
    CLI in print mode; the transcript leaves the machine only for the
    configured backend, after a one-time confirmation that names it. Open:
    the Claude API directly (needs a new dependency and key handling), or a
    local model. Whether a `todo` kind should be added to the built-ins.

## Future enhancements

- Zoom the focused panel to full screen, as lazygit does, on `+`. `+` is
  reserved for this and must not be given another meaning.
- Markdown rendering in Body with `charmbracelet/glamour` (headings, lists,
  code blocks, emphasis). Needs a new dependency, so verify it is current and
  maintained and get approval first. Keep the plain-text view available as a
  toggle, and make sure line selection for extraction (`v`) still maps to
  source lines when the rendered view is on.

## Package layout

- `cmd/lazynote/` - root model, update, mouse, modal box, search.
- `internal/ui/{panel,popup,helppopup,statusbar}` - shared widgets
  (port from lazytask).
- `internal/ui/{para,related,info,body,stickies}` - one package per panel.
- `internal/store/` - notes backend interface plus markdown implementation.
- `internal/state/` - SQLite session state.
- `internal/{editor,config,undo,version}` - as in lazytask.

## Chunks (one objective each, human checkpoint between chunks)

Done and removed from this list to save context; the behavior they added is
recorded in the Decisions and Intake sections, the code and its tests, and
`README.md`. The numbers stay stable because other sections cite them:

- 1 scaffold, 2 config and path safety (2a store read, 2b kinds), 3 PARA panel
  (3c to 3i browsing, selection, undo, yank/cut/paste, trash/create, search,
  refresh), 4 Body, 5 Info and Title (5b unique uid on copy, 5c metadata rows,
  5d edit in `$EDITOR`), 6 session state, 7 notebooks, 8 Related and the link
  index (8a to 8c), 9 move and kind (9b intake core, 9c shell capture,
  9d extract, 9e daily rules, 9f URL clip, 9g open dailies, 9h retention,
  9i find), 10 mouse, 11 release tooling and README, 12 stickies,
  13 configurable kinds.

Remaining:

14. Transcript digester (agentic import). Replaces the dumb text import for
    transcripts; the plain import stays as the fallback. Not started.
    - Goal: `I` then `t` (and `lazynote import FILE --type transcript`)
      hands the transcript to an LLM agent that returns a structured digest,
      instead of only saving the raw text as a `rumor` note.
    - Output, matching the hand-made summaries in the inbox: a summary note
      with sections Facts, Rumors / Unconfirmed, Todos and Ideas, plus the
      untouched transcript. Each item becomes a candidate note, so the human
      never has to retype it.
    - Flow: (1) save the raw transcript as today (`source: transcript`,
      `kind: rumor`, text unmodified), so nothing is lost if the agent fails;
      (2) run the agent in the background (status bar "Digesting
      transcript..."; the app stays usable); (3) show a review list of
      proposed notes (kind, title, body, tick to keep); (4) on accept, create
      the kept notes in `inbox/` through `intake.Extract`-style creation,
      each `related`-linked to the transcript, one undo entry for the batch;
      (5) write the summary into the transcript note above the transcript
      under a `# Summary` heading, as the manual process does. Nothing is
      written without the review step.
    - Kind mapping: Facts -> `fact`, Rumors -> `rumor`, Todos -> `proposal`
      (until a `todo` kind exists in config), Ideas -> `idea`. Items the agent
      is unsure about default to `rumor`. Kinds come from the configured list.
    - Engine in a new `internal/digest`: a `Digester` interface
      (`Digest(ctx, transcript) (Digest, error)`), a `Digest` value type
      (summary, attendees, items with kind, title, body, owner and source
      quote), validation of the agent's output, and a backend. Tests use a
      fake `Digester`; no test calls a real model or the network.
    - Backend (open decision 12): default is to run the Claude Code CLI in
      print mode with a fixed prompt and a JSON schema, passed the transcript
      on standard input. No new Go dependency, and no API key handled by
      lazynote. Command, model and timeout come from a `digest:` config block.
    - Safety: the transcript is untrusted data, so the prompt tells the agent
      to treat it as content, not instructions. Output is parsed as JSON only,
      checked against the schema, cut to size limits (items, title length,
      body length), run through `internal/textsafe`, and never executed or
      used as a path. Titles go through `store.CleanTitle`. The agent gets no
      file or shell tools. The transcript is sent only to the configured
      backend, and only after the first-use confirmation (decision 12).
      Errors never show transcript text.
    - Limits: files above the import limit (10 MB) are refused as today; a
      transcript over the model limit is split into chunks on speaker-turn
      boundaries and the partial digests are merged and de-duplicated.
    - Failure: backend missing, timeout, bad JSON or an empty digest shows a
      warning and leaves the raw note in place; the user can retry with a key
      on the transcript note.
    - Shell: `lazynote import FILE --type transcript --digest` prints the
      proposed items as JSON on standard output and writes nothing but the
      raw note, so a script can review them. `--no-digest` forces the old
      behavior.
    - Config: `digest.enabled` (default false until the backend is chosen),
      `digest.command`, `digest.timeout`.
    - Sub-chunks, one commit each: 14a engine types, validation and fake
      backend, tested; 14b CLI backend and `digest:` config, tested; 14c
      chunking and merge, tested; 14d TUI: background run, status bar and
      review list; 14e summary write-back and `--digest` flag; 14f README and
      manual QA note.

Each chunk: tests for any engine code it adds, a manual QA note for any UI it
adds, `go build ./...` and `go test ./...` passing, one commit.

## Testing

Tests cover the engine only; the UI is verified by manual QA.

- Engine packages (tested): `internal/findtext`, `internal/store` (path safety, move, copy,
  trash, overwrite, create, rename with link rewriting, frontmatter
  round-trip, kind parsing, retention purge), `internal/state`,
  `internal/undo`, `internal/config`, and the link index. Use table-driven
  unit tests, plus integration tests against a temp notebook directory and a
  temp state directory. Tests must never touch a real notebook or the real
  state database.
- UI packages (`cmd/lazynote`, `internal/ui/*`): no automated tests. Each UI
  chunk ships a manual QA note that says exactly what to run and what to look
  for.
- No coverage percentage and no end-to-end suite.
- This deliberately deviates from the organization policy (80% coverage plus
  integration and end-to-end tests); the project owner chose it for a
  personal project.
