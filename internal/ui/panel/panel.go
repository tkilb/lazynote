// Package panel provides the shared lazygit-style bordered-panel styling
// used across the panel grid: a rounded border and bold title, both
// switching to a highlight color when the panel is focused. Ported from
// lazytask.
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

	// SelectedRowBackground is the background color for the selected row in a
	// list panel, matching lazygit's default theme.selectedLineBgColor.
	SelectedRowBackground = lipgloss.Color("4")
	// activeTabColor highlights the selected tab in a FrameTabs title, kept
	// distinct from FocusedColor so it stays visible in a focused panel.
	activeTabColor = lipgloss.Color("4")

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
//
// An optional footer (e.g. a "3/10" scroll position) is embedded
// right-aligned in the bottom border line. Only the first is used.
func Frame(title, body string, width, height int, focused bool, footer ...string) string {
	lineStyle := lipgloss.NewStyle().Foreground(borderColor(focused))
	top := topBorder(title, width, focused, lineStyle)
	return frameBody(top, firstFooter(footer), body, width, height, lineStyle)
}

func firstFooter(footer []string) string {
	if len(footer) == 0 {
		return ""
	}
	return footer[0]
}

// Tab describes one segment of a multi-tab panel title (see FrameTabs).
type Tab struct {
	Label  string
	Active bool
}

// FrameTabs is like Frame, but embeds a row of tabs in the title instead of
// a single label (e.g. "╭─[2]-Inbox - Projects─╮"), styling the active tab
// distinctly from the others.
func FrameTabs(number rune, tabs []Tab, body string, width, height int, focused bool, footer ...string) string {
	lineStyle := lipgloss.NewStyle().Foreground(borderColor(focused))
	top := topBorderTabs(number, tabs, width, focused, lineStyle)
	return frameBody(top, firstFooter(footer), body, width, height, lineStyle)
}

// frameBody renders the border/content assembly shared by Frame and
// FrameTabs, given an already-built top border line.
func frameBody(top, footer, body string, width, height int, lineStyle lipgloss.Style) string {
	// Height/Width here size only the content area; the border characters
	// are added separately above/below/around it.
	content := lipgloss.NewStyle().Width(width).Height(height).Render(body)

	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n")
	side := lineStyle.Render("│")
	for _, line := range strings.Split(content, "\n") {
		b.WriteString(side)
		b.WriteString(line)
		b.WriteString(side)
		b.WriteString("\n")
	}
	b.WriteString(bottomBorder(footer, width, lineStyle))
	return b.String()
}

// ScrollWindow returns the [start, end) slice bounds of a scrollable list of
// totalItems entries, sized so exactly visibleRows are shown, such that
// cursor always falls within [start, end). It is a pure function of its
// arguments, so callers need not persist a scroll offset.
func ScrollWindow(cursor, totalItems, visibleRows int) (start, end int) {
	visibleRows = max(visibleRows, 1)
	if totalItems <= visibleRows {
		return 0, totalItems
	}
	start = min(max(cursor-visibleRows+1, 0), totalItems-visibleRows)
	return start, start + visibleRows
}

// topBorder builds the top border line with the title embedded, e.g.
// "╭──[1]-Notebook────────╮". A title that does not fit within width is
// truncated to avoid overflowing the panel.
func topBorder(title string, width int, focused bool, lineStyle lipgloss.Style) string {
	const leadDashes = 2
	label := titleLabel(title)
	if len(label) > width-leadDashes {
		label = label[:max(width-leadDashes, 0)]
	}
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

// bottomBorder builds the bottom border line, optionally embedding footer
// right-aligned near the corner (e.g. "╰──────────3/10─╯"). A footer that
// does not fit within width is dropped rather than truncated.
func bottomBorder(footer string, width int, lineStyle lipgloss.Style) string {
	if footer == "" || len(footer) > width {
		return lineStyle.Render("╰" + strings.Repeat("─", width) + "╯")
	}

	const trailDashes = 1
	fill := max(width-trailDashes-len(footer), 0)
	return lineStyle.Render("╰"+strings.Repeat("─", fill)) +
		lipgloss.NewStyle().Render(footer) +
		lineStyle.Render(strings.Repeat("─", trailDashes)+"╯")
}

// tabSeparator joins the labels in a FrameTabs title.
const tabSeparator = " - "

// topBorderTabs builds the top border line for FrameTabs, with the active
// tab styled distinctly. It uses the first of these layouts that fits within
// width: every full label; inactive tabs shrunk to their first letter; the
// active tab alone. If none fits, it falls back to a truncated plain label.
func topBorderTabs(number rune, tabs []Tab, width int, focused bool, lineStyle lipgloss.Style) string {
	prefix := "[" + string(number) + "]-"

	const leadDashes = 2
	budget := width - leadDashes - len(prefix)
	var shown []Tab
	for _, layout := range tabLayouts(tabs) {
		if tabsWidth(layout) <= budget {
			shown = layout
			break
		}
	}
	if shown == nil {
		return topBorder(prefix+tabsPlain(tabs), width, focused, lineStyle)
	}

	inactive := lipgloss.NewStyle()
	active := lipgloss.NewStyle().Bold(true).Foreground(activeTabColor)

	var b strings.Builder
	b.WriteString(lineStyle.Render("╭" + strings.Repeat("─", leadDashes)))
	b.WriteString(TitleStyle(focused).Render(prefix))
	for i, t := range shown {
		if i > 0 {
			b.WriteString(inactive.Render(tabSeparator))
		}
		style := inactive
		if t.Active {
			style = active
		}
		b.WriteString(style.Render(t.Label))
	}
	fill := max(budget-tabsWidth(shown), 0)
	b.WriteString(lineStyle.Render(strings.Repeat("─", fill) + "╮"))
	return b.String()
}

// tabLayouts lists the candidate layouts of tabs, widest first.
func tabLayouts(tabs []Tab) [][]Tab {
	compact := make([]Tab, len(tabs))
	var activeOnly []Tab
	for i, t := range tabs {
		compact[i] = t
		if !t.Active && t.Label != "" {
			compact[i].Label = t.Label[:1]
		}
		if t.Active {
			activeOnly = append(activeOnly, t)
		}
	}
	return [][]Tab{tabs, compact, activeOnly}
}

func tabsWidth(tabs []Tab) int {
	total := len(tabSeparator) * max(len(tabs)-1, 0)
	for _, t := range tabs {
		total += len(t.Label)
	}
	return total
}

func tabsPlain(tabs []Tab) string {
	labels := make([]string, len(tabs))
	for i, t := range tabs {
		labels[i] = t.Label
	}
	return strings.Join(labels, tabSeparator)
}
