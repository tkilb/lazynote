package main

import (
	"fmt"
	"time"

	"github.com/tkilb/lazynote/internal/intake"
	"github.com/tkilb/lazynote/internal/ui/popup"
)

// openDaily jumps PARA to the daily note offset days from today (-1
// yesterday, 0 today, 1 tomorrow) and focuses PARA. A day with no note gets
// an empty one in inbox/daily, which undo removes.
func (m model) openDaily(offset int) model {
	res, err := intake.Daily(m.store, time.Now().AddDate(0, 0, offset))
	if err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf("could not open the daily note: %s.", err)))
	}
	if !res.Existed {
		m.history.Push(res.Action)
	}
	m.para = m.para.Reload()
	para, ok := m.para.Reveal(res.Rel)
	if !ok {
		return m.notify(popup.Warning, "That note is outside the PARA folders.")
	}
	m.para, m.focus = para, panelPARA
	return m.refreshLinks()
}
