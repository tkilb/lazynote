package main

import (
	"fmt"

	"github.com/tkilb/lazynote/internal/ui/popup"
)

// yankBody copies the selected Body lines, or the cursor line, to the system
// clipboard and clears the selection. A failure is shown as an error and the
// selection is kept.
func (m model) yankBody() model {
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
