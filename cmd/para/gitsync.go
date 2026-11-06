package main

import (
	"context"

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
