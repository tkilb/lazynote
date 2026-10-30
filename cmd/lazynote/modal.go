package main

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/popup"
	"github.com/tkilb/lazynote/internal/undo"
)

// input is an open text prompt.
type input struct {
	title, label string
	op           para.InputOp
	field        textinput.Model
}

// inputCharLimit bounds what the prompt accepts.
const inputCharLimit = 255

// apply carries out an Effect from a panel: it records the undo action,
// queues the notices as popups, and opens the question or prompt it asks
// for. The returned command blinks a prompt's cursor.
func (m model) apply(eff para.Effect) (model, tea.Cmd) {
	if eff.Action != nil {
		m.history.Push(*eff.Action)
	}
	if eff.Info != "" {
		m = m.notify(popup.Info, eff.Info)
	}
	if eff.Warn != "" {
		m = m.notify(popup.Warning, eff.Warn)
	}
	if eff.Err != nil {
		m = m.notify(popup.Error, capitalize(eff.Err.Error())+".")
	}

	var cmd tea.Cmd
	switch {
	case eff.Prompt != nil:
		m.input, cmd = m.openInput(eff.Prompt)
	case eff.Confirm != nil:
		m.pending = eff.Confirm
	}
	return m, cmd
}

// notify queues a popup notification.
func (m model) notify(sev popup.Severity, text string) model {
	m.popups = m.popups.Push(popup.Message{Severity: sev, Text: text})
	return m
}

// openInput starts a text prompt sized to the screen.
func (m model) openInput(p *para.Prompt) (*input, tea.Cmd) {
	field := textinput.New()
	field.Prompt = ""
	field.CharLimit = inputCharLimit
	field.Width = popup.FieldWidth(m.width)
	field.Focus()
	return &input{title: p.Title, label: p.Label, op: p.Op, field: field}, textinput.Blink
}

// typeKey feeds a key to the open prompt: enter accepts, esc cancels.
func (m model) typeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input = nil
		return m, nil
	case "enter":
		op, text := m.input.op, m.input.field.Value()
		m.input = nil
		var eff para.Effect
		m.para, eff = m.para.ApplyInput(op, text)
		return m.apply(eff)
	}
	var cmd tea.Cmd
	m.input.field, cmd = m.input.field.Update(msg)
	return m, cmd
}

// answer handles a key while a question is pending: y or enter accepts, n
// or esc declines, anything else is ignored.
func (m model) answer(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		op := m.pending.Op
		m.pending = nil
		var eff para.Effect
		m.para, eff = m.para.ApplyConfirm(op)
		return m.apply(eff)
	case "n", "esc":
		m.pending = nil
	}
	return m, nil
}

// runHistory runs an undo or redo, reports the result in a popup, and
// reloads the listing because files may have moved.
func (m model) runHistory(done, failed string, run func() (string, error)) model {
	desc, err := run()
	switch {
	case errors.Is(err, undo.ErrNothingToUndo), errors.Is(err, undo.ErrNothingToRedo):
		m = m.notify(popup.Info, capitalize(err.Error())+".")
	case err != nil:
		m = m.notify(popup.Error, failed+" failed ("+desc+"): "+err.Error()+".")
	default:
		m = m.notify(popup.Info, done+": "+desc)
	}
	m.para = m.para.Reload()
	return m
}

// overlayModals draws the open question or prompt over view, then the
// front notification on top of that.
func (m model) overlayModals(view string) string {
	switch {
	case m.input != nil:
		box := popup.InputBox(m.input.title, m.input.label, m.input.field.View(), m.width)
		view = popup.Overlay(view, box, m.width, m.height)
	case m.pending != nil && m.pending.Danger:
		view = popup.Overlay(view, popup.DangerConfirmBox(m.pending.Prompt, m.width), m.width, m.height)
	case m.pending != nil:
		view = popup.Overlay(view, popup.ConfirmBox(m.pending.Prompt, m.width), m.width, m.height)
	}
	if msg, ok := m.popups.Current(); ok {
		view = popup.Overlay(view, popup.Box(msg, m.width), m.width, m.height)
	}
	return view
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
