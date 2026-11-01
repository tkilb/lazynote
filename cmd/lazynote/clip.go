package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/clip"
	"github.com/tkilb/lazynote/internal/intake"
	"github.com/tkilb/lazynote/internal/ui/clipform"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// clipTimeout bounds the whole title fetch, a little over the fetcher's own.
const clipTimeout = 10 * time.Second

// clipFetchedMsg arrives when the page title fetch ends. err is nil when
// title holds the page's title.
type clipFetchedMsg struct {
	url *url.URL
	// dir is the folder PARA was showing when the address was submitted; the
	// note is saved there.
	dir   string
	title string
	err   error
}

// startClip opens the clip form, with the address filled in from the
// clipboard when that holds one. Enter saves: with a title the note is saved
// at once; without one the page's own title is fetched in the background.
func (m model) startClip() (model, tea.Cmd) {
	initial, _ := clipboard.ReadAll()
	if _, err := clip.ParseURL(initial); err != nil {
		initial = ""
	}
	return m.openClipForm(clipform.Values{URL: strings.TrimSpace(initial)}, clipform.FieldURL)
}

// openClipForm shows the clip form with v filled in and the keyboard on f.
func (m model) openClipForm(v clipform.Values, f clipform.Field) (model, tea.Cmd) {
	form, _ := clipform.New("", m.width, m.height)
	form, cmd := form.WithValues(v, f)
	m.clipForm = &form
	return m, cmd
}

// clipFormKey handles a key while the clip form is open: esc cancels, enter
// saves, and everything else edits the form.
func (m model) clipFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.clipForm = nil
		return m, nil
	case "enter":
		v := m.clipForm.Values()
		m.clipForm = nil
		return m.saveClip(v)
	}
	f, cmd := m.clipForm.Update(msg)
	m.clipForm = &f
	return m, cmd
}

// saveClip acts on the form. A bad address says why and brings the form back
// with what was typed. A title saves the note at once, in the folder PARA is
// showing; without one the page is asked for its title.
func (m model) saveClip(v clipform.Values) (model, tea.Cmd) {
	if strings.TrimSpace(v.URL) == "" && strings.TrimSpace(v.Title) == "" {
		return m, nil // nothing typed
	}
	u, err := clip.ParseURL(v.URL)
	if err != nil {
		m = m.notify(popup.Warning, capitalize(err.Error())+".")
		return m.openClipForm(v, clipform.FieldURL)
	}
	if strings.TrimSpace(v.Title) == "" {
		m.clipping++
		return m, m.fetchTitle(u, m.para.Dir())
	}
	res, err := intake.Clip(m.store, time.Now(), m.para.Dir(), u, v.Title)
	m = m.finishIntake(res, err)
	if err != nil {
		return m.openClipForm(v, clipform.FieldTitle)
	}
	return m, nil
}

// fetchTitle returns the command that fetches the title of u.
func (m model) fetchTitle(u *url.URL, dir string) tea.Cmd {
	fetcher := m.fetcher
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), clipTimeout)
		defer cancel()
		title, err := fetcher.Title(ctx, u)
		return clipFetchedMsg{url: u, dir: dir, title: title, err: err}
	}
}

// finishClip saves the note. When the title could not be fetched the note is
// still saved, named after the address, with a warning saying why. When the
// note cannot be saved the prompt comes back with the address, so it is not
// lost.
func (m model) finishClip(msg clipFetchedMsg) (model, tea.Cmd) {
	m.clipping = max(m.clipping-1, 0)
	res, err := intake.Clip(m.store, time.Now(), msg.dir, msg.url, msg.title)
	m = m.finishIntake(res, err)
	if err != nil {
		return m.openClipForm(clipform.Values{URL: msg.url.String()}, clipform.FieldURL)
	}
	if msg.err != nil {
		m = m.notify(popup.Warning, fmt.Sprintf("%s, so the note is named after the address.", capitalize(msg.err.Error())))
	}
	return m, nil
}
