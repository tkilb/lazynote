package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/intake"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// captureKinds maps the global capture keys to the kind they create and the
// title of their prompt.
var captureKinds = map[string]struct {
	kind  store.Kind
	title string
}{
	"F": {store.KindFact, "Quick fact"},
	"R": {store.KindRumor, "Quick rumor"},
	"i": {store.KindIdea, "Quick idea"},
}

// openRootInput opens a one-line prompt whose text goes to accept.
func (m model) openRootInput(title string, accept func(model, string) (model, tea.Cmd)) (model, tea.Cmd) {
	in, cmd := m.openInput(&para.Prompt{Title: title})
	in.accept = accept
	m.input = in
	return m, cmd
}

// startCapture opens the quick capture prompt for the capture key.
func (m model) startCapture(key string) (model, tea.Cmd) {
	c := captureKinds[key]
	return m.openRootInput(c.title, func(m model, text string) (model, tea.Cmd) {
		res, err := intake.Capture(m.store, time.Now(), c.kind, intake.SourceQuick, text)
		return m.finishIntake(res, err, true), nil
	})
}

// startImport asks for a file path, then whether it is a transcript or a note.
func (m model) startImport() (model, tea.Cmd) {
	return m.openRootInput("Import file (path)", func(m model, text string) (model, tea.Cmd) {
		path := strings.TrimSpace(text)
		if path == "" {
			return m, nil
		}
		return m.openMenu("Import as", []popup.MenuItem{
			{Key: "t", Label: "Transcript (kept as written, kind rumor)"},
			{Key: "n", Label: "Note (markdown, properties kept)"},
		}, func(m model, key string) model {
			if key == "t" {
				res, err := intake.ImportTranscript(m.store, time.Now(), path)
				return m.finishIntake(res, err, false)
			}
			res, err := intake.ImportNote(m.store, time.Now(), path)
			return m.finishIntake(res, err, false)
		}), nil
	})
}

// finishIntake records the undo entry for a created note and refreshes the
// panels. Nothing is said when the new note shows in the current list; when
// it does not, a notice says where it went. Typing nothing is not an error.
func (m model) finishIntake(res intake.Result, err error, quiet bool) model {
	switch {
	case err != nil && quiet && err == intake.ErrEmpty:
		return m
	case err != nil:
		return m.notify(popup.Error, capitalize(fmt.Sprintf("could not add the note: %s.", err)))
	}
	m.history.Push(res.Action)
	m.para = m.para.Reload()
	m = m.refreshLinks()
	if res.Unchanged {
		m = m.notify(popup.Warning, "The note's properties could not be read, so it was imported unchanged and keeps its original uid.")
	}
	if !m.para.Shows(res.Rel) {
		m = m.notify(popup.Info, "Saved to the Inbox as "+strings.TrimSuffix(filepath.Base(res.Rel), ".md")+".")
	}
	return m
}
