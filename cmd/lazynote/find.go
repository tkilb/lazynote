package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/findtext"
	"github.com/tkilb/lazynote/internal/ui/picker"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

const (
	findFileTitle = "Find note"
	findTextTitle = "Find text"
	// valueSep splits a note's path from a line number in a picker value;
	// a path cannot hold a NUL byte.
	valueSep = "\x00"
)

// noteName is a note's file name without .md.
func noteName(rel string) string {
	return strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
}

// openFileFinder lists every note in the notebook by its path without .md
// (such as "projects/deep/Rocket Plan"), sorted by path, so what is typed
// matches the folders as well as the name. It jumps PARA to the one chosen.
func (m model) openFileFinder() (model, tea.Cmd) {
	rels := slices.Clone(m.links.Rels())
	if len(rels) == 0 {
		return m.notify(popup.Info, "There are no notes to find."), nil
	}
	slices.SortFunc(rels, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	items := make([]picker.Item, len(rels))
	for i, rel := range rels {
		items[i] = picker.Item{Label: strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel)), Value: rel}
	}
	p, cmd := picker.New(findFileTitle, items)
	m.picker, m.pickerOp = &p, pickFile
	return m, cmd
}

// openTextFinder lists every non-blank line of every note, the note's name
// and line number dim at the right, and opens the note at the line chosen.
// The text is typed to narrow the list, as in the other pickers.
func (m model) openTextFinder() (model, tea.Cmd) {
	res := findtext.Lines(m.store, m.links.Rels())
	if len(res.Lines) == 0 {
		return m.notify(popup.Info, "There is no text to search."), nil
	}
	items := make([]picker.Item, len(res.Lines))
	for i, l := range res.Lines {
		items[i] = picker.Item{
			Label:  l.Text,
			Detail: fmt.Sprintf("%s:%d", noteName(l.Rel), l.Index+1),
			Value:  l.Rel + valueSep + strconv.Itoa(l.Index),
		}
	}
	title := findTextTitle
	if res.Truncated {
		title = fmt.Sprintf("%s (first %d lines)", findTextTitle, findtext.MaxLines)
	}
	p, cmd := picker.New(title, items)
	m.picker, m.pickerOp = &p, pickText
	return m, cmd
}

// chooseFoundNote opens the note chosen in a finder. A text hit also puts
// Content's cursor on the line and focuses Content; a note hit focuses PARA.
func (m model) chooseFoundNote(value string) model {
	rel, lineText, hasLine := strings.Cut(value, valueSep)
	line := -1
	if hasLine {
		if n, err := strconv.Atoi(lineText); err == nil {
			line = n
		}
	}
	para, ok := m.para.Reveal(rel)
	if !ok {
		return m.notify(popup.Warning, "That note is outside the PARA folders.")
	}
	m.para = para
	m = m.syncSelection()
	if line >= 0 {
		m.content, m.focus = m.content.SelectRow(line), panelContent
	} else {
		m.focus = panelPARA
	}
	return m
}
