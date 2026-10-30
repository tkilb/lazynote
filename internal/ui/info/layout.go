package info

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	// minCellWidth is the narrowest a column may get; the panel uses as many
	// columns as fit, up to maxColumns.
	minCellWidth = 24
	maxColumns   = 3
	// columnGap separates columns.
	columnGap = 2
)

// layout flows the rows into the panel width. Rows that fit on one line
// within a column share lines, side by side, to save vertical space; a row
// that does not fit (a long or multi-line value) takes the full width on
// its own lines, so nothing is cut off. The result scrolls vertically.
func (m Model) layout() Model {
	m.lines = nil
	if m.width > 0 {
		if m.std != nil {
			m.lines = m.std.lines(m.width)
		}
		m.lines = append(m.lines, flow(m.rows, m.width)...)
	}
	m.offset = m.clamp(m.offset)
	return m
}

func flow(rows []row, width int) []line {
	cols := min(max((width+columnGap)/(minCellWidth+columnGap), 1), maxColumns)
	cellWidth := (width - columnGap*(cols-1)) / cols

	var out []line
	var pending []string
	flush := func() {
		if len(pending) > 0 {
			out = append(out, line{text: strings.Join(pending, strings.Repeat(" ", columnGap))})
			pending = nil
		}
	}
	for _, r := range rows {
		if c, ok := cell(r, cellWidth); ok {
			pending = append(pending, c)
			if len(pending) == cols {
				flush()
			}
			continue
		}
		flush()
		out = append(out, fullRow(r, width)...)
	}
	flush()
	return out
}

// shortLabel is r's label cut to maxLabelWidth.
func shortLabel(r row) string {
	return ansi.Truncate(r.label, maxLabelWidth, "…")
}

// cell draws r as exactly width columns on one line, or reports false when
// its value is not a single line or label and value do not fit.
func cell(r row, width int) (string, bool) {
	if len(r.value) != 1 {
		return "", false
	}
	label, value := shortLabel(r), r.value[0]
	used := ansi.StringWidth(label) + labelGap + ansi.StringWidth(value)
	if used > width {
		return "", false
	}
	pad := strings.Repeat(" ", width-used)
	gap := strings.Repeat(" ", labelGap)
	if r.highlight != "" {
		return highlightStyle(r).Render(label + gap + value + pad), true
	}
	return labelStyle.Render(label) + gap + r.style.Render(value) + pad, true
}

// fullRow draws r across the whole width: the label on the first line and
// the wrapped value beside it, with continuation lines indented under the
// value.
func fullRow(r row, width int) []line {
	label := shortLabel(r)
	labelWidth := ansi.StringWidth(label)
	valueWidth := max(width-labelWidth-labelGap, 1)

	var wrapped []string
	for _, v := range r.value {
		wrapped = append(wrapped, strings.Split(ansi.Wrap(v, valueWidth, ""), "\n")...)
	}
	if len(wrapped) == 0 {
		wrapped = []string{""}
	}

	gap := strings.Repeat(" ", labelGap)
	indent := strings.Repeat(" ", labelWidth)
	out := make([]line, len(wrapped))
	for i, v := range wrapped {
		head := indent
		if i == 0 {
			head = label
		}
		if r.highlight != "" {
			text := head + gap + v
			text += strings.Repeat(" ", max(width-ansi.StringWidth(text), 0))
			out[i] = line{text: highlightStyle(r).Render(text)}
			continue
		}
		out[i] = line{text: labelStyle.Render(head) + gap + r.style.Render(v)}
	}
	return out
}

// highlightStyle draws a row on its colored background, as for the kind.
func highlightStyle(r row) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Background(r.highlight).Foreground(lipgloss.Color("0"))
}
