package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/intake"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/kindstyle"
	"github.com/tkilb/lazynote/internal/ui/picker"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

const extractKindTitle = "Extract as"

// extractState is an extraction in progress: the note and lines it takes
// from, and the kind once chosen.
type extractState struct {
	origin, text string
}

// startExtract begins extracting the lines selected in Body: first the kind
// is chosen in a picker, then the title is asked for.
func (m model) startExtract() (model, tea.Cmd) {
	text := m.body.SelectedText()
	if strings.TrimSpace(text) == "" {
		return m.notify(popup.Info, "Only blank lines are selected. Pick some text first."), nil
	}
	var items []picker.Item
	for _, k := range store.Kinds() {
		items = append(items, picker.Item{
			Label: k.String(), Detail: k.Meaning(), Value: k.String(), Color: kindstyle.Color(k),
		})
	}
	p, cmd := picker.New(extractKindTitle, items)
	m.picker, m.pickerOp = &p, pickExtractKind
	m.extract = &extractState{origin: m.body.Rel(), text: text}
	return m, cmd
}

// askExtractTitle asks for the new note's title, starting with the first
// line of the selection, once its kind is chosen.
func (m model) askExtractTitle(kind store.Kind) (model, tea.Cmd) {
	first, _ := intake.SplitText(m.extract.text)
	title := fmt.Sprintf("Extract as %s: title", kind)
	return m.openRootInput(title, store.CleanTitle(first), func(m model, typed string) (model, tea.Cmd) {
		x := m.extract
		res, err := intake.Extract(m.store, time.Now(), kind, x.origin, typed, x.text)
		if err != nil {
			// Keep the selection, so the user can try again.
			return m.finishIntake(res, err), nil
		}
		m.extract = nil
		m.body = m.body.ClearSelection()
		return m.finishIntake(res, nil), nil
	})
}
