package main

import (
	"fmt"

	"github.com/tkilb/lazynote/internal/ui/popup"
)

// yankBody copies the selected Body lines, or the cursor line, to the system
// clipboard and clears the selection. In the Markdown reading view it only
// leaves that view. A failure is shown as an error and the
// selection is kept.
func (m model) yankBody() model {
	if m.body.Rendered() {
		// Yanking is picking lines, which the reading view cannot do: leave it
		// so the line to be copied shows, and let y be pressed again.
		m.body = m.body.LeaveRendered()
		return m
	}
	text, n := m.body.YankText()
	if n == 0 {
		return m
	}
	if err := m.writeClipboard(text); err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf(
			"could not copy to the clipboard: %s (it needs pbcopy on macOS, or xclip, xsel or wl-copy on Linux).", err)))
	}
	m.body = m.body.Yanked(n)
	return m
}
