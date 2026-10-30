# lazynote

A TUI note browser in Go with Bubble Tea, modeled on `~/Git/lazytask`.
`PLAN.md` is the source of truth for layout, keybindings, decisions and the
chunk list. Read it before changing behavior; do not re-decide what it records.

## Commands

- Go 1.25 (`go.mod` declares 1.25.0, which `modernc.org/sqlite` requires).
- Build: `make build` (writes the `lazynote` binary, which is git-ignored).
- Test: `make test` (`go test ./...`).
- Both `go build ./...` and `go test ./...` must pass before a commit, and
  after a commit that adds files, from a clean clone (a bad `.gitignore` once
  hid `cmd/lazynote/` from every commit).

## Testing policy

Tests cover engine packages only (`internal/store`, `internal/state`,
`internal/undo`, `internal/config`, `internal/intake`, the link index). Use table-driven tests
against temp directories; never touch a real notebook or the real state
database. UI packages (`cmd/lazynote`, `internal/ui/*`) have no automated
tests; each UI chunk ships a manual QA note instead. This deliberately
deviates from the organization 80% coverage policy (project owner decision).

## Workflow

- Work on a feature branch, never on `main`.
- One chunk per commit, with a human checkpoint between chunks.
- Agent instructions live in this file; never create a `CLAUDE.md`.
- Verify a dependency exists and is maintained before adding it to `go.mod`.
