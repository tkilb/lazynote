# lazynote - Agentic import plan

Temporary. This file records the agreed design for the agentic import and the
chunks to build it. Delete it in the last chunk. Do not re-decide what it
records; ask the user if something is unclear.

## Goal

Replace the old import (`I` with a typed path, and `lazynote import`) with an
agentic import. One source goes in; a folder of grouped concept notes comes out.
The owner cares about how the business works (facts, concepts, ideas) far more
than who does what. Action items are secondary and a separate task TUI tracks
them.

## Decisions (all confirmed by the user)

Sources: Zoom transcripts, Confluence dumps and Slack threads as `.txt` or `.md`
files or clipboard text, and web pages (scraped by lazynote).

Input (`I`): one box, no type choice.
- Empty then Enter: import the clipboard.
- Starts with `http`: fetch the page. Only `http` and `https`, never local or
  private addresses (reuse `internal/clip`). A login page or failed fetch stops
  with a clear warning and creates nothing; the user pastes or saves a file.
- Anything else: a path, with Tab completion, `~` expansion, and a bare name
  searched in `inbox/` first.
- A preview line under the box ("file, 42 KB", "clipboard, 3,100 words").
- `I` with the cursor on or inside an `import_` folder re-runs that folder
  immediately, with no dialog. New imports start from anywhere else.
- Files above the import limit (10 MB) are refused.

Backend: Claude Code CLI in print mode, no tools, Sonnet by default (not Haiku).
- Verified flags: `-p`, `--model`, `--output-format`, `--json-schema`, `--tools`,
  `--system-prompt`, `--no-session-persistence`. Re-check `claude --help`
  before using any other flag; never invent one.
- Command, model and timeout live in a `digest:` config block.
  `digest.enabled: true` is the user's consent; there is no confirmation prompt.
- The source is sent on standard input. It is untrusted data.

Prompts: two complete files in the user's config directory, `import-prompt.md`
(main pass) and `import-merge-prompt.md` (merge pass). The user owns the whole
prompt, including the output schema. lazynote writes defaults on first use. The
defaults say: cluster related concepts into one file each, keep facts and ideas
apart from actions, drop small talk and pleasantries, rank how things work above
who does what, suggest a few tags, name a short topic, and judge the `source`.

Output, written directly (best effort, no review step):
- Folder `inbox/import_YYYY-MM-DD_<topic>/`. If the agent gives no usable topic,
  or fails, use `import_YYYY-MM-DD_<file name, page title or "clipboard">`.
- `_source.md` first, always, before the agent runs (a copy of the file, the
  clipboard text or the page text). It is never changed after.
- One file per cluster of related concepts and ideas.
- `_actions.md` only when action items exist.
- Frontmatter on every file: `kind: import`, `source` from the configured
  sources (`meeting`, `confluence`, `slack`, `web`, `document`, `adhoc` ...; the
  agent judges it; a URL import records the URL), `date` (import day), `tags`
  (agent suggested), `related` links to `_source.md` and `_actions.md`.
- If `import` is missing from the user's kinds, write `kind: unset` and warn.
- Titles go through `store.CleanTitle`; agent text goes through
  `internal/textsafe`; agent output is never a path. The agent writes only inside
  the one `import_` folder and never touches PARA or any existing note.
- The original file is never modified.

Large sources: split on speaker or message boundaries, run each part, then a
second agent pass (the merge prompt) merges the clusters. If the merge fails,
keep the unmerged part files.

Re-run: delete every generated file in the folder (all but `_source.md`) and
rewrite them. No manifest or hashes; the user's edits in generated files are
replaced. Undo restores the previous files. One undo entry per import or re-run.

Failure (backend missing, timeout, bad JSON, empty result): warn, keep
`_source.md`. Errors never show source text.

Display: runs in the background. Status bar shows "Importing..." and the app
stays usable. On success it shows "Imported: <folder> (N files)" and clears
itself after 5 seconds. The cursor never moves.

Removed: the `lazynote import` command and the old plain import. The `U` link
clip stays as the quick bookmark (kind `link`, already renamed).

## Chunks (one commit each, human checkpoint between chunks)

Write tests first for engine code (table-driven, temp dirs, a fake backend; no
test calls a real model or the network). UI chunks ship a manual QA note.

1. Engine types in `internal/digest`: `Digester` interface, result types,
   schema validation, size limits, fake backend. Tested.
2. CLI backend and `digest:` config block, default prompt files. Tested with a
   fake command.
3. Source readers: file, clipboard, URL fetch, login-page detection. Tested.
4. Folder writer: naming and fallback, `_source.md`, frontmatter, kind fallback,
   `related` links, one undo entry. Tested.
5. Split and merge. Tested.
6. Re-run: replace generated files, undo restores. Tested.
7. TUI: the `I` dialog (completion, inbox search, preview), re-run on an
   `import_` folder, background run, status bar, 5 second success message. Manual
   QA note.
8. Remove the old import and `lazynote import`, update the README and the help
   popup, then delete this file.

Each chunk: `go build ./...` and `go test ./...` pass, one commit with a
`type: description` subject.
