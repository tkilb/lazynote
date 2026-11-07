package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/para/internal/templates"
	"github.com/tkilb/para/internal/ui/noteform"
	"github.com/tkilb/para/internal/ui/picker"
	"github.com/tkilb/para/internal/ui/popup"
)

const appendTemplateTitle = "Append template"

// templateNames lists the templates the open notebook offers, never daily.
func (m model) templateNames() ([]string, error) {
	return templates.List(m.templatesDir, m.notebookName())
}

// loadTemplate reads the named template of the open notebook.
func (m model) loadTemplate(name string) (templates.Template, error) {
	path, err := templates.Resolve(m.templatesDir, m.notebookName(), name)
	if err != nil {
		return templates.Template{}, err
	}
	return templates.Load(name, path)
}

// templateFailed tells the user a template could not be used.
func (m model) templateFailed(name string, err error) model {
	return m.notify(popup.Warning, capitalize(fmt.Sprintf("the template %q could not be used (%s).", name, err)))
}

// startAppendTemplate opens the template picker for the selected note.
func (m model) startAppendTemplate() (model, tea.Cmd) {
	entry, ok := m.para.SelectedNote()
	if !ok {
		return m.notify(popup.Info, "Select a note to append a template to."), nil
	}
	names, err := m.templateNames()
	if err != nil {
		return m.notify(popup.Warning, capitalize(fmt.Sprintf("the templates could not be listed (%s).", err))), nil
	}
	if len(names) == 0 {
		return m.notify(popup.Info, "This notebook has no templates. Add .md files to "+
			shortPath(filepath.Join(m.templatesDir, m.notebookName()))+" (or "+shortPath(m.templatesDir)+" for all notebooks)."), nil
	}
	items := make([]picker.Item, len(names))
	for i, n := range names {
		items[i] = picker.Item{Label: n, Value: n}
	}
	p, cmd := picker.New(appendTemplateTitle, items)
	m.picker, m.pickerFor, m.pickerOp = &p, entry.Rel, pickAppendTemplate
	return m, cmd
}

// appendTemplate adds the rendered body of the named template to the end of
// the note at rel, as one undo entry. The note's own properties and text stay
// as they are.
func (m model) appendTemplate(rel, name string) model {
	t, err := m.loadTemplate(name)
	if err != nil {
		return m.templateFailed(name, err)
	}
	title := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	text := templates.Render(t.Body, templates.Vars{Day: time.Now(), Title: title})
	act, err := m.store.Append(rel, text)
	if err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf("appending %q failed: %s.", name, err)))
	}
	m.history.Push(act)
	m.para = m.para.Reload()
	return m.refreshLinks()
}

// newQuickAddForm makes the Add Note form with the Template chooser when the
// notebook has templates. A failure to list them leaves the chooser out and
// is returned for the caller to show.
func (m model) newQuickAddForm() (noteform.Model, tea.Cmd, error) {
	form, cmd := noteform.New("Add Note", m.sources, m.width, m.height)
	names, err := m.templateNames()
	if err != nil {
		return form, cmd, err
	}
	return form.WithTemplates(names), cmd, nil
}

// templateChanged reacts to a new choice in the form's Template chooser, whose
// previous position was prev. Choosing a template over a body the user typed
// is confirmed first, with the choice put back meanwhile.
func (m model) templateChanged(f noteform.Model, prev int) model {
	if !f.BodyEdited() || f.TemplateName() == "" {
		return m.applyTemplate(f, prev)
	}
	chosen := f.TemplateIndex()
	back := f.SetTemplateIndex(prev)
	m.form = &back
	return m.ask("Replace the body you typed with the template?", func(m model) model {
		return m.applyTemplate(m.form.SetTemplateIndex(chosen), prev)
	})
}

// applyTemplate fills the form from the template it has chosen: the body and
// the kind, source and tags the template sets. Choosing none removes a body
// a template put in. A template that cannot be used is reported and the
// choice goes back to prev.
func (m model) applyTemplate(f noteform.Model, prev int) model {
	name := f.TemplateName()
	if name == "" {
		f = f.ClearFilled()
	} else if t, err := m.loadTemplate(name); err != nil {
		f = f.SetTemplateIndex(prev)
		m = m.templateFailed(name, err)
	} else {
		body := templates.Render(t.Body, templates.Vars{Day: time.Now(), Title: strings.TrimSpace(f.Title())})
		f = f.Apply(noteform.Fill{Body: body, Kind: t.Kind, Source: t.Source, Tags: t.Tags})
	}
	m.form = &f
	return m
}
