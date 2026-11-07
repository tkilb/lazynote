package noteform

import (
	"strings"

	"github.com/tkilb/para/internal/store"
)

// WithTemplates offers names in the Template chooser, after "none", and
// shows the chooser. No names leaves it hidden.
func (m Model) WithTemplates(names []string) Model {
	m.templates = append([]string(nil), names...)
	m.tmplIdx = 0
	return m.Resize(m.screenWidth, m.screenHeight)
}

// hasTemplates reports whether the Template chooser is shown.
func (m Model) hasTemplates() bool { return len(m.templates) > 0 }

// TemplateName is the chosen template's name, or "" for none.
func (m Model) TemplateName() string {
	if m.tmplIdx == 0 {
		return ""
	}
	return m.templates[m.tmplIdx-1]
}

// TemplateIndex is the chooser's position: 0 for none, then 1 for the first
// name. A host compares it before and after Update to see a change.
func (m Model) TemplateIndex() int { return m.tmplIdx }

// SetTemplateIndex moves the chooser to idx (see TemplateIndex).
func (m Model) SetTemplateIndex(idx int) Model {
	if idx >= 0 && idx <= len(m.templates) {
		m.tmplIdx = idx
	}
	return m
}

// BodyEdited reports whether the body holds text that a template did not put
// there, so replacing it would lose the user's typing.
func (m Model) BodyEdited() bool {
	body := m.body.Value()
	return body != "" && body != m.filled
}

// Title is the title as typed, for filling {{title}} into a template.
func (m Model) Title() string { return m.title.Value() }

// Apply puts a template's text in the body and its kind, source and tags in
// their fields, leaving a field the template does not set as it is.
func (m Model) Apply(f Fill) Model {
	m.body.SetValue(f.Body)
	m.body.CursorStart()
	m.filled = f.Body
	if f.Kind != store.KindUnset {
		for i, k := range m.kinds {
			if k == f.Kind {
				m.kindIdx = i
			}
		}
	}
	for i, s := range m.sources {
		if f.Source != "" && s == f.Source {
			m.srcIdx = i
		}
	}
	if len(f.Tags) > 0 {
		m.tags.SetValue(strings.Join(f.Tags, " "))
	}
	return m
}

// ClearFilled empties the body when a template put its text there and the
// user has not edited it since, as when the choice goes back to none.
func (m Model) ClearFilled() Model {
	if m.filled != "" && m.body.Value() == m.filled {
		m.body.SetValue("")
	}
	m.filled = ""
	return m
}
