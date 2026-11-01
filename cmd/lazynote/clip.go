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
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// clipTimeout bounds the whole title fetch, a little over the fetcher's own.
const clipTimeout = 10 * time.Second

// clipFetchedMsg arrives when the page title fetch ends. err is nil when
// title holds the page's title.
type clipFetchedMsg struct {
	url   *url.URL
	title string
	err   error
}

// startClip asks for a web address, starting with the clipboard's text when
// that is one, then fetches the page title in the background.
func (m model) startClip() (model, tea.Cmd) {
	initial, _ := clipboard.ReadAll()
	if _, err := clip.ParseURL(initial); err != nil {
		initial = ""
	}
	return m.askClipURL(strings.TrimSpace(initial))
}

// askClipURL opens the address prompt. A bad address says why and asks again,
// keeping what was typed.
func (m model) askClipURL(initial string) (model, tea.Cmd) {
	return m.openRootInput("Clip URL", initial, func(m model, text string) (model, tea.Cmd) {
		text = strings.TrimSpace(text)
		if text == "" {
			return m, nil
		}
		u, err := clip.ParseURL(text)
		if err != nil {
			m = m.notify(popup.Warning, capitalize(err.Error())+".")
			return m.askClipURL(text)
		}
		m.clipping++
		return m, m.fetchTitle(u)
	})
}

// fetchTitle returns the command that fetches the title of u.
func (m model) fetchTitle(u *url.URL) tea.Cmd {
	fetcher := m.fetcher
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), clipTimeout)
		defer cancel()
		title, err := fetcher.Title(ctx, u)
		return clipFetchedMsg{url: u, title: title, err: err}
	}
}

// finishClip saves the note. When the title could not be fetched the note is
// still saved, named after the address, with a warning saying why. When the
// note cannot be saved the prompt comes back with the address, so it is not
// lost.
func (m model) finishClip(msg clipFetchedMsg) (model, tea.Cmd) {
	m.clipping = max(m.clipping-1, 0)
	res, err := intake.Clip(m.store, time.Now(), msg.url, msg.title)
	m = m.finishIntake(res, err)
	if err != nil {
		return m.askClipURL(msg.url.String())
	}
	if msg.err != nil {
		m = m.notify(popup.Warning, fmt.Sprintf("%s, so the note is named after the address.", capitalize(msg.err.Error())))
	}
	return m, nil
}
