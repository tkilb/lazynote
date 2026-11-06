package main

import (
	"fmt"

	"github.com/tkilb/para/internal/ui/popup"
)

// yankContent copies the selected Content lines, or the cursor line, to the system
// clipboard and clears the selection. In the Markdown reading view it only
// leaves that view. A failure is shown as an error and the
// selection is kept.
func (m model) yankContent() model {
	if m.content.Rendered() {
		// Yanking is picking lines, which the reading view cannot do: leave it
		// so the line to be copied shows, and let y be pressed again.
		m.content = m.content.LeaveRendered()
		return m
	}
	text, n := m.content.YankText()
	if n == 0 {
		return m
	}
	if err := m.writeClipboard(text); err != nil {
		return m.notify(popup.Error, capitalize(fmt.Sprintf(
			"could not copy to the clipboard: %s (it needs pbcopy on macOS, or xclip, xsel or wl-copy on Linux).", err)))
	}
	m.content = m.content.Yanked(n)
	return m
}
