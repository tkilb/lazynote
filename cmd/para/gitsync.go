package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/para/internal/gitsync"
	"github.com/tkilb/para/internal/ui/statusbar"
)

// syncReadMsg carries the git state read for the notebook at path.
type syncReadMsg struct {
	path   string
	status gitsync.Status
	ok     bool
}

// syncDebounce is the least time between two reads of the git state that
// follow changes to the notes.
const syncDebounce = 60 * time.Second

// syncDueMsg arrives when a read held back by the debounce is due.
type syncDueMsg struct{}

// requestSync asks for a read after the notes changed: at once if the last
// read is older than syncDebounce, otherwise one read when it ends, however
// many changes come before then.
func (m model) requestSync() tea.Cmd {
	if m.syncGate == nil {
		return nil
	}
	run, wait := m.syncGate.Request()
	switch {
	case run:
		return m.readSync()
	case wait > 0:
		return tea.Tick(wait, func(time.Time) tea.Msg { return syncDueMsg{} })
	}
	return nil
}

// forceSync reads now, for what the user asked for (starting, switching
// notebook, refreshing, leaving lazygit), whatever the debounce says. It also
// drops the pending request that the change which came with it raised.
func (m model) forceSync() (model, tea.Cmd) {
	m.syncDirty = false
	if m.syncGate != nil {
		m.syncGate.Mark()
	}
	return m, m.readSync()
}

// finishDue reads when a held request is due.
func (m model) finishDue() tea.Cmd {
	if m.syncGate != nil && m.syncGate.Due() {
		return m.readSync()
	}
	return nil
}

// readSync reads the open notebook's git state off the UI thread. It uses
// local state only, so "behind" is as of the last fetch.
func (m model) readSync() tea.Cmd {
	if m.activeNotebook < 0 || m.activeNotebook >= len(m.notebooks) {
		return nil
	}
	path := m.notebooks[m.activeNotebook].Path
	return func() tea.Msg {
		st, ok := gitsync.Read(context.Background(), path)
		return syncReadMsg{path: path, status: st, ok: ok}
	}
}

// finishSync keeps the state read, unless the notebook changed meanwhile.
func (m model) finishSync(msg syncReadMsg) model {
	if m.activeNotebook < 0 || m.activeNotebook >= len(m.notebooks) || m.notebooks[m.activeNotebook].Path != msg.path {
		return m
	}
	m.gitSync = nil
	if msg.ok {
		st := msg.status
		m.gitSync = &st
	}
	return m
}

// syncState is the sync state for the status bar.
func (m model) syncState() *statusbar.Sync {
	if m.gitSync == nil {
		return nil
	}
	return &statusbar.Sync{Tracked: m.gitSync.Tracked, Ahead: m.gitSync.Ahead, Behind: m.gitSync.Behind, Uncommitted: m.gitSync.Uncommitted}
}
