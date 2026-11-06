// Package statusbar renders a lazygit-style bottom help bar showing the
// keybindings active for whatever panel/mode currently has focus. It has
// no knowledge of notes or any specific panel — callers pass in the
// list of bindings relevant to their current mode.
package statusbar

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// keyStyle uses ANSI blue (SGR 34), matching lazygit's default
	// options-bar color (theme.OptionsTextColor: "blue").
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("4"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// bookStyle draws the open notebook's name, as lazytask draws its
	// taskbook.
	bookStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6"))

	aheadStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e78a4e"))
	behindStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	inSyncStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dirtyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cleanStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	separator = "  "
)

// Binding describes a single keybinding entry shown in the status bar, e.g.
// {Key: "a", Label: "add"}.
type Binding struct {
	Key   string
	Label string
}

// Render lays out bindings as a single lazygit-style line, e.g.
// "a add  d done  x delete  q quit". Each entry pairs a highlighted key
// with a dimmed label. Returns an empty string if bindings is empty.
func Render(bindings []Binding) string {
	if len(bindings) == 0 {
		return ""
	}

	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, keyStyle.Render(b.Key)+" "+labelStyle.Render(b.Label))
	}
	return strings.Join(parts, separator)
}

// Sync is the open notebook's git state: the number of uncommitted files and,
// with an upstream (Tracked), the commits to push (Ahead) and to pull (Behind).
type Sync struct {
	Tracked       bool
	Ahead, Behind int
	Uncommitted   int
}

// renderSync draws the sync state right after the notebook name, with no space
// before it, as parts joined by commas: the uncommitted files "629" (red),
// then the commits to push "2↑" (orange) and to pull "3↓" (blue), only the
// sides that need syncing, or "✓" (green) when a tracked branch needs none.
// A branch without an upstream that is fully committed shows a yellow "✓", as
// there is nothing to compare with. Such as "(629,2↑3↓)" or "(✓)". Nothing is
// drawn for a nil sync.
func renderSync(s *Sync) string {
	if s == nil {
		return ""
	}
	var parts []string
	if s.Uncommitted > 0 {
		parts = append(parts, dirtyStyle.Bold(true).Render(strconv.Itoa(s.Uncommitted)))
	}
	if s.Tracked {
		var ud string
		if s.Ahead > 0 {
			ud += aheadStyle.Bold(true).Render(strconv.Itoa(s.Ahead) + "↑")
		}
		if s.Behind > 0 {
			ud += behindStyle.Bold(true).Render(strconv.Itoa(s.Behind) + "↓")
		}
		if ud == "" {
			ud = inSyncStyle.Bold(true).Render("✓")
		}
		parts = append(parts, ud)
	}
	if len(parts) == 0 { // committed, with no upstream to compare with
		parts = append(parts, cleanStyle.Bold(true).Render("✓"))
	}
	return bookStyle.Render("(") + strings.Join(parts, bookStyle.Render(",")) + bookStyle.Render(")")
}

// RenderWithBook renders the open notebook's name, its git sync state when
// sync is not nil, then a space and the key bindings.
func RenderWithBook(book string, sync *Sync, bindings []Binding) string {
	res := Render(bindings)
	if book == "" {
		return res
	}
	rendered := bookStyle.Render(book) + renderSync(sync)
	if res == "" {
		return rendered
	}
	return rendered + " " + res
}
