// Package panel provides the shared lazygit-style bordered-panel styling
// used across the panel grid: a rounded border and bold title, both
// switching to a highlight color when the panel is focused. Ported from
// lazytask; tab titles and scroll helpers arrive with the PARA panel.
package panel

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// UnfocusedColor is the border/title color for a panel that does not
	// currently have focus. The empty color leaves the terminal foreground
	// in place, matching lazygit's "default" theme color.
	UnfocusedColor = lipgloss.Color("")
	// FocusedColor is the border/title color for the focused panel,
	// matching lazygit's default theme.activeBorderColor.
	FocusedColor = lipgloss.Color("2")

	// minWidth/minHeight guard against nonsensical (zero or negative) panel
	// sizes, e.g. before the first tea.WindowSizeMsg arrives.
	minWidth  = 10
	minHeight = 1
)

// TitleStyle returns the bold title style for a panel, colored according to
// whether it currently has focus.
func TitleStyle(focused bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(borderColor(focused))
}

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return FocusedColor
	}
	return UnfocusedColor
}

// Render draws a titled, bordered panel with the given body text, sized to
// occupy outerWidth x outerHeight (the panel's total footprint, border
// included). The title is embedded in the top border line (lazygit-style,
// e.g. "╭─[1]-Notebook─────╮") so the full inner height is available to body.
func Render(title, body string, outerWidth, outerHeight int, focused bool) string {
	width, height := InnerSize(outerWidth, outerHeight)
	return Frame(title, body, width, height, focused)
}

// InnerSize converts a panel's total outer footprint (border included) into
// the content width/height available inside the border, clamped to sane
// minimums.
func InnerSize(outerWidth, outerHeight int) (width, height int) {
	width = max(outerWidth-2, minWidth)
	height = max(outerHeight-2, minHeight)
	return width, height
}

// Frame wraps body (already sized to width columns and up to height rows)
// in a rounded border of exactly width+2 x height+2, with title embedded in
// the top border line rather than as a separate content row.
func Frame(title, body string, width, height int, focused bool) string {
	lineStyle := lipgloss.NewStyle().Foreground(borderColor(focused))

	// Height/Width here size only the content area; the border characters
	// are added separately above/below/around it.
	content := lipgloss.NewStyle().Width(width).Height(height).Render(body)

	var b strings.Builder
	b.WriteString(topBorder(title, width, focused, lineStyle))
	b.WriteString("\n")
	side := lineStyle.Render("│")
	for _, line := range strings.Split(content, "\n") {
		b.WriteString(side)
		b.WriteString(line)
		b.WriteString(side)
		b.WriteString("\n")
	}
	b.WriteString(lineStyle.Render("╰" + strings.Repeat("─", width) + "╯"))
	return b.String()
}

// topBorder builds the top border line with the title embedded, e.g.
// "╭──[1]-Notebook────────╮". A title that does not fit within width is
// truncated to avoid overflowing the panel.
func topBorder(title string, width int, focused bool, lineStyle lipgloss.Style) string {
	label := titleLabel(title)
	if len(label) > width {
		label = label[:width]
	}

	const leadDashes = 2
	fill := max(width-leadDashes-len(label), 0)

	var b strings.Builder
	b.WriteString(lineStyle.Render("╭" + strings.Repeat("─", leadDashes)))
	b.WriteString(TitleStyle(focused).Render(label))
	b.WriteString(lineStyle.Render(strings.Repeat("─", fill) + "╮"))
	return b.String()
}

// titleLabel formats a panel title for the border line. Titles of the form
// "<digit> <name>" (the number-key hint, e.g. "1 Notebook") become
// "[1]-Notebook"; any other title is used as-is.
func titleLabel(title string) string {
	if len(title) > 2 && title[1] == ' ' && title[0] >= '0' && title[0] <= '9' {
		return "[" + string(title[0]) + "]-" + title[2:]
	}
	return title
}
