package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/intake"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/noteform"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// captureKinds maps the global capture keys to the kind they create and the
// word that names it in the form.
var captureKinds = map[string]struct {
	kind  store.Kind
	label string // names the kind in the form: "Fact title"
}{
	"F": {store.KindFact, "Fact"},
	"R": {store.KindRumor, "Rumor"},
	"i": {store.KindIdea, "Idea"},
}

// openRootInput opens a one-line prompt whose text goes to accept.
func (m model) openRootInput(title string, accept func(model, string) (model, tea.Cmd)) (model, tea.Cmd) {
	in, cmd := m.openInput(&para.Prompt{Title: title})
	in.accept = accept
	m.input = in
	return m, cmd
}

// startCapture opens the quick capture form for the capture key.
func (m model) startCapture(key string) (model, tea.Cmd) {
	c := captureKinds[key]
	form, cmd := noteform.New(c.label, m.width, m.height)
	m.form = &form
	m.formSave = func(m model, title, body string) model {
		if strings.TrimSpace(title) == "" && strings.TrimSpace(body) == "" {
			return m // nothing typed
		}
		res, err := intake.Capture(m.store, time.Now(), c.kind, intake.SourceQuick, title, body)
		return m.finishIntake(res, err)
	}
	return m, cmd
}

// formKey handles a key while the capture form is open: esc cancels, enter in
// the title and ctrl+s anywhere save, and everything else edits the form.
func (m model) formKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	save := func() (tea.Model, tea.Cmd) {
		title, body, onSave := m.form.Title(), m.form.Body(), m.formSave
		m.form, m.formSave = nil, nil
		return onSave(m, title, body), nil
	}
	switch msg.String() {
	case "esc":
		m.form, m.formSave = nil, nil
		return m, nil
	case "ctrl+s":
		return save()
	case "enter":
		if m.form.FocusedField() == noteform.FieldTitle {
			return save()
		}
	}
	f, cmd := m.form.Update(msg)
	m.form = &f
	return m, cmd
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
				return m.finishIntake(res, err)
			}
			res, err := intake.ImportNote(m.store, time.Now(), path)
			return m.finishIntake(res, err)
		}), nil
	})
}

// finishIntake records the undo entry for a created note and refreshes the
// panels. Nothing is said when the new note shows in the current list; when
// it does not, a notice says where it went.
func (m model) finishIntake(res intake.Result, err error) model {
	if err != nil {
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
