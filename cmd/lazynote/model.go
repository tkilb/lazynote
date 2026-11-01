package main

import (
	"time"

	"github.com/atotto/clipboard"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/lazynote/internal/config"
	"github.com/tkilb/lazynote/internal/links"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/body"
	"github.com/tkilb/lazynote/internal/ui/info"
	"github.com/tkilb/lazynote/internal/ui/noteform"
	"github.com/tkilb/lazynote/internal/ui/panel"
	"github.com/tkilb/lazynote/internal/ui/para"
	"github.com/tkilb/lazynote/internal/ui/picker"
	"github.com/tkilb/lazynote/internal/ui/popup"
	"github.com/tkilb/lazynote/internal/ui/related"
	"github.com/tkilb/lazynote/internal/ui/statusbar"
	"github.com/tkilb/lazynote/internal/ui/title"
	"github.com/tkilb/lazynote/internal/undo"
)

// panelID identifies a panel. Its value is the number key that focuses it,
// so Body is 0 and Notebook is 1, matching the titles in the grid.
type panelID int

const (
	panelBody panelID = iota
	panelNotebook
	panelPARA
	panelRelated
	panelInfo
)

var panelNames = map[panelID]string{
	panelBody:     "Body",
	panelNotebook: "Notebook",
	panelPARA:     "PARA",
	panelRelated:  "Related",
	panelInfo:     "Info",
}

// focusCycle is the Tab/Shift+Tab order: the left column top to bottom,
// then the right column top to bottom. The Title panel is not in it.
var focusCycle = []panelID{panelNotebook, panelPARA, panelRelated, panelBody, panelInfo}

func (p panelID) next() panelID { return focusCycle[(p.cycleIndex()+1)%len(focusCycle)] }

func (p panelID) prev() panelID {
	return focusCycle[(p.cycleIndex()-1+len(focusCycle))%len(focusCycle)]
}

func (p panelID) cycleIndex() int {
	for i, c := range focusCycle {
		if c == p {
			return i
		}
	}
	return 0
}

// globalBindings are always shown in the status bar, after any bindings of
// the focused panel.
var globalBindings = []statusbar.Binding{
	{Key: "0-4", Label: "focus panel"},
	{Key: "tab", Label: "next panel"},
	{Key: "u", Label: "undo"},
	{Key: "?", Label: "help"},
	{Key: "ctrl+l", Label: "refresh"},
	{Key: "ctrl+z", Label: "suspend"},
	{Key: "q", Label: "quit"},
}

var paraBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "move"},
	{Key: "←/→", Label: "up/open"},
	{Key: "[ ]", Label: "tab"},
	{Key: "space", Label: "select"},
	{Key: "y/x/p", Label: "yank/cut/paste"},
	{Key: "d/D", Label: "trash/delete"},
	{Key: "a", Label: "new"},
	{Key: "r", Label: "rename"},
	{Key: "e", Label: "edit"},
	{Key: "/", Label: "search"},
	{Key: "n/N", Label: "next/prev match"},
	{Key: "v", Label: "visual"},
	{Key: ".", Label: "hidden"},
	{Key: "enter", Label: "open"},
}

var relatedBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "move"},
	{Key: "l", Label: "add reference"},
	{Key: "d", Label: "remove reference"},
	{Key: "enter", Label: "jump to note"},
	{Key: "esc/←", Label: "back to PARA"},
}

var infoBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "scroll"},
	{Key: "esc/←", Label: "back to PARA"},
}

var bodyBindings = []statusbar.Binding{
	{Key: "↑/↓", Label: "move"},
	{Key: "ctrl+d/u", Label: "half page"},
	{Key: "space", Label: "pick line"},
	{Key: "v", Label: "select lines"},
	{Key: "y", Label: "yank"},
	{Key: "r", Label: "rename"},
	{Key: "enter", Label: "extract"},
	{Key: "esc/←", Label: "back to PARA"},
}

type model struct {
	store   store.Store
	focus   panelID
	para    para.Model
	body    body.Model
	info    info.Model
	related related.Model
	title   title.Model
	// links indexes the related properties; nil until it could be built.
	links         *links.Index
	shown         shownNote
	width, height int
	// history is shared by pointer: it holds the session's undo/redo stack.
	history *undo.Stack
	// popups queues notifications; the front one blocks input until a key
	// dismisses it.
	popups popup.Model
	// pending is a yes/no question waiting for an answer.
	pending *para.Confirm
	// input is an open one-line text prompt.
	input *input
	// picker is the open fuzzy picker for adding a reference to the note at
	// pickerFor.
	picker    *picker.Model
	pickerFor string
	pickerOp  pickerOp
	// pickerKind is the kind the note had when the kind picker opened.
	pickerKind store.Kind
	// extract is the extraction in progress, if any.
	extract *extractState
	// writeClipboard copies text to the system clipboard (a yank).
	writeClipboard func(string) error
	// sources are the sources Quick Add offers, from the config.
	sources []string
	// form is the open quick-capture form; formSave handles its texts.
	form     *noteform.Model
	formSave func(model, noteform.Values) (model, tea.Cmd)
	// menu is the open choose-by-key menu, if any.
	menu *menu
	// question is a yes/no question owned by the root model, with what to do
	// on yes (the para panel's own questions use pending instead).
	question *question
	// session saves and restores the position between runs.
	session session
	// showingHelp is set while the help popup is open, scrolled by helpScroll.
	showingHelp bool
	helpScroll  int
	// saveErr is a failure to save on quit, reported after the screen closes.
	saveErr error
}

// shownNote identifies what the Body and Info panels last loaded. A changed
// modification time means the file changed on disk, and a changed item
// count means a folder listed differently (for example after toggling
// hidden files).
type shownNote struct {
	rel   string
	mod   time.Time
	items int
}

func initialModel(s store.Store) model {
	return model{
		store: s, focus: panelPARA, para: para.New(s),
		body: body.New(), info: info.New(), related: related.New(), title: title.New(), history: &undo.Stack{},
		sources: config.DefaultSources, writeClipboard: clipboard.WriteAll,
	}
}

// syncSelection makes Body and Info show what is under the PARA cursor,
// reading the note once, and only when the selection or the file changed.
// A folder shows its name and item count in Info and nothing in Body.
func (m model) syncSelection() model {
	entry, ok := m.para.Selected()
	if !ok {
		m.body, m.info, m.shown = m.body.Clear(), m.info.Clear(), shownNote{}
		m.related = m.related.Clear()
		m.title = m.title.Set("")
		return m
	}
	now := shownNote{rel: entry.Rel, mod: entry.ModTime, items: entry.ItemCount}
	if now == m.shown {
		return m
	}
	m.shown = now

	if entry.IsDir {
		m.title = m.title.Set("")
		m.body = m.body.Clear()
		m.related = m.related.Clear()
		m.info = m.info.ShowDir(entry.Rel, entry.Name, entry.ItemCount)
		return m
	}
	m.related = m.related.Show(entry.Rel, m.links.References(entry.Rel), m.links.ReferencedBy(entry.Rel))
	note, err := m.store.ReadNote(entry.Rel)
	if err != nil {
		m.title = m.title.Set(entry.Name)
		m.body = m.body.Fail(entry.Rel, err)
		m.info = m.info.ShowError(entry.Rel, err)
		return m
	}
	m.title = m.title.SetNote(note.Kind, note.Title)
	m.body = m.body.Load(entry.Rel, note.Body)
	m.info = m.info.ShowNote(entry.Rel, note)
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	return next.(model).syncSelection(), cmd
}

func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.form != nil {
			f := m.form.Resize(m.width, m.height)
			m.form = &f
		}
		g := computeGrid(m.width, m.height-1)
		_, rows := panel.InnerSize(g.leftWidth, g.paraHeight)
		m.para = m.para.Resize(rows)
		bw, bh := panel.InnerSize(g.rightWidth, g.bodyHeight)
		m.body = m.body.Resize(bw, bh)
		iw, ih := panel.InnerSize(g.rightWidth, g.infoHeight)
		m.info = m.info.Resize(iw, ih)
	case tea.FocusMsg:
		// Files may have changed while the terminal was in the background.
		// A pending question was worded for the rows on screen, so it waits.
		if m.pending == nil {
			m.para = m.para.Reload()
			m = m.refreshLinks()
		}
		return m, nil
	case editDoneMsg:
		return m.finishEdit(msg), tea.ClearScreen
	case tea.ResumeMsg:
		// Returning from ctrl+z: repaint everything on the alternate screen.
		return m, tea.ClearScreen
	case tea.KeyMsg:
		return m.updateKey(msg)
	default:
		// Cursor blink and similar messages belong to the open prompt.
		if m.picker != nil {
			p, cmd, _ := m.picker.Update(msg)
			m.picker = &p
			return m, cmd
		}
		if m.form != nil {
			f, cmd := m.form.Blink(msg)
			m.form = &f
			return m, cmd
		}
		if m.input != nil {
			var cmd tea.Cmd
			m.input.field, cmd = m.input.field.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key != "ctrl+c" {
		switch {
		case m.popups.Active():
			m.popups = m.popups.Dismiss()
			return m, nil
		case m.showingHelp:
			return m.updateHelp(msg), nil
		case m.form != nil:
			return m.formKey(msg)
		case m.menu != nil:
			return m.menuKey(key)
		case m.question != nil:
			return m.answerQuestion(key)
		case m.picker != nil:
			return m.pickerKey(msg)
		case m.pending != nil:
			return m.answer(key)
		case m.input != nil:
			return m.typeKey(msg)
		}
	}
	var cmd tea.Cmd
	switch key {
	case "q", "ctrl+c":
		m.saveErr = m.save()
		return m, tea.Quit
	case "?":
		m.showingHelp = true
	case "A":
		m, cmd = m.startQuickAdd()
	case "I":
		m, cmd = m.startImport()
	case "b":
		m = m.openDaily(-1)
	case "t":
		m = m.openDaily(0)
	case "T":
		m = m.openDaily(1)
	case "ctrl+z":
		return m.saveOrWarn(), tea.Suspend
	case "ctrl+l":
		m.para = m.para.Reload()
		m = m.refreshLinks()
		cmd = tea.ClearScreen
	case "u":
		m = m.runHistory("Undid", "Undo", m.history.Undo)
	case "ctrl+r":
		m = m.runHistory("Redid", "Redo", m.history.Redo)
	case "tab":
		m.focus = m.focus.next()
	case "shift+tab":
		m.focus = m.focus.prev()
	case "0", "1", "2", "3", "4":
		m.focus = panelID(key[0] - '0')
	case "esc", "left":
		switch m.focus {
		case panelBody:
			// Closing a range or clearing the selection comes before leaving
			// the panel.
			if m.body.Active() {
				m.body = m.body.Escape()
			} else {
				m.focus = panelPARA
			}
		case panelInfo, panelRelated:
			m.focus = panelPARA
		case panelPARA:
			m, cmd = m.paraKey(msg)
		}
	case "enter", "right":
		if m.focus == panelRelated {
			m, cmd = m.relatedKey(msg)
			break
		}
		if m.focus == panelBody && m.body.HasSelection() && key == "enter" {
			m, cmd = m.startExtract()
			break
		}
		if m.focus != panelPARA {
			break
		}
		if _, ok := m.para.SelectedNote(); ok {
			m.focus = panelBody
		} else {
			m, cmd = m.paraKey(msg)
		}
	default:
		if key == "y" && m.focus == panelBody {
			m = m.yankBody()
			break
		}
		if key == "r" && m.focus == panelBody {
			// Body shows the note under the PARA cursor, so renaming it is
			// PARA's rename.
			m, cmd = m.paraKey(msg)
			break
		}
		switch m.focus {
		case panelPARA:
			m, cmd = m.paraKey(msg)
		case panelRelated:
			m, cmd = m.relatedKey(msg)
		case panelBody:
			m.body = m.body.Update(msg)
		case panelInfo:
			m.info = m.info.Update(msg)
		}
	}
	return m, cmd
}

// paraKey sends a key to the PARA panel and applies what it asks for.
func (m model) paraKey(msg tea.KeyMsg) (model, tea.Cmd) {
	var eff para.Effect
	m.para, eff = m.para.Update(msg)
	return m.apply(eff)
}

func (m model) bindings() []statusbar.Binding {
	var local []statusbar.Binding
	switch m.focus {
	case panelPARA:
		local = paraBindings
	case panelRelated:
		local = relatedBindings
	case panelBody:
		local = bodyBindings
	case panelInfo:
		local = infoBindings
	}
	return append(append([]statusbar.Binding(nil), local...), globalBindings...)
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	g := computeGrid(m.width, m.height-1) // one row for the status bar
	render := func(id panelID, w, h int) string {
		switch id {
		case panelPARA:
			return m.para.View(w, h, m.focus == id)
		case panelRelated:
			return m.related.View(w, h, m.focus == id)
		case panelBody:
			return m.body.View(w, h, m.focus == id)
		case panelInfo:
			return m.info.View(w, h, m.focus == id)
		}
		title := string(rune('0'+id)) + " " + panelNames[id]
		return panel.Render(title, "", w, h, m.focus == id)
	}

	left := lipgloss.JoinVertical(lipgloss.Left,
		render(panelNotebook, g.leftWidth, g.notebookHeight),
		render(panelPARA, g.leftWidth, g.paraHeight),
		render(panelRelated, g.leftWidth, g.relatedHeight),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		m.title.View(g.rightWidth, g.titleHeight),
		render(panelBody, g.rightWidth, g.bodyHeight),
		render(panelInfo, g.rightWidth, g.infoHeight),
	)
	grid := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	view := lipgloss.JoinVertical(lipgloss.Left, grid, ansi.Truncate(m.statusLine(), m.width, "…"))
	return m.overlayModals(view)
}
