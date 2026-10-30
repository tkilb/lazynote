package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/editor"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// editDoneMsg arrives when the editor exits; err is nil on success.
type editDoneMsg struct{ err error }

// edit opens the note at rel in the editor, handing the terminal over until
// it exits. A failure to prepare the command is reported as if the editor
// itself had failed.
func (m model) edit(rel string) tea.Cmd {
	path, err := m.store.Path(rel)
	if err != nil {
		return failEdit(err)
	}
	cmd, err := editor.Command(path, os.Environ())
	if err != nil {
		return failEdit(err)
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return editDoneMsg{err: err} })
}

func failEdit(err error) tea.Cmd {
	return func() tea.Msg { return editDoneMsg{err: err} }
}

// finishEdit refreshes from disk, since the editor may have changed the note,
// and reports a failure to run it.
func (m model) finishEdit(msg editDoneMsg) model {
	m.para = m.para.Reload()
	m = m.refreshLinks()
	if msg.err != nil {
		m = m.notify(popup.Error, fmt.Sprintf("Could not run the editor: %s.", msg.err))
	}
	return m
}
