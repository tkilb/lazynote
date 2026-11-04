package info

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"go.yaml.in/yaml/v3"

	"github.com/tkilb/lazynote/internal/store"
)

const (
	sourceKey = "source"
	dateKey   = "date"
	tagsKey   = "tags"
	peopleKey = "people"
	// relatedKey is hidden: the Related panel shows it.
	relatedKey = "related"

	// stdColumns is how many cells the first standard row holds.
	stdColumns = 3
	// labelColumn aligns the values of the single-cell standard rows.
	labelColumn = len("People")
	// inlineBudget bounds the lines read from one standard value.
	inlineBudget = 8
)

// standard is the fixed head of a note's Info: the rows that sit in the same
// place for every note, whatever it holds.
type standard struct {
	kind                       store.Kind
	kindColor                  lipgloss.Color
	source, date, tags, people string
}

// isStandardKey reports whether key is drawn in the standard rows (or
// hidden) rather than among the other properties.
func isStandardKey(key string) bool {
	switch key {
	case uidKey, relatedKey, sourceKey, dateKey, tagsKey, peopleKey:
		return true
	}
	return false
}

// newStandard reads the standard values out of fm, which may be nil.
func newStandard(kind store.Kind, color lipgloss.Color, fm *store.Frontmatter) standard {
	s := standard{kind: kind, kindColor: color}
	if fm == nil {
		return s
	}
	s.source = inline(fm.Get(sourceKey))
	s.date = dateText(fm.Get(dateKey))
	s.tags = inline(fm.Get(tagsKey))
	s.people = inline(fm.Get(peopleKey))
	return s
}

// inline draws a value on one line, items joined with commas.
func inline(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	budget := inlineBudget
	return strings.Join(valueLines(n, 0, &budget), ", ")
}

// dateText draws a date as YYYY-MM-DD, without any time of day. A value that
// is not a date is shown as written.
func dateText(n *yaml.Node) string {
	text := inline(n)
	if day, ok := store.DateOnly(text); ok {
		return day
	}
	return text
}

// lines draws the three standard rows for width columns: Kind, Source and
// Date side by side; then Tags; then People, each across the whole width. Each is exactly one line; a long value is
// cut with an ellipsis.
func (s standard) lines(width int) []line {
	cell := max((width-columnGap*(stdColumns-1))/stdColumns, 1)
	gap := strings.Repeat(" ", columnGap)
	join := func(cells ...string) line { return line{text: strings.Join(cells, gap)} }
	return []line{
		join(s.kindCell(cell), pair("Source", s.source, 0, cell), pair("Date", s.date, 0, cell)),
		{text: pair("Tags", s.tags, labelColumn, width)},
		{text: pair("People", s.people, labelColumn, width)},
	}
}

// kindCell draws the highlighted Kind cell.
func (s standard) kindCell(width int) string {
	r := row{highlight: s.kindColor}
	text := ansi.Truncate("Kind"+strings.Repeat(" ", labelGap)+s.kind.String(), width, "…")
	text += strings.Repeat(" ", max(width-ansi.StringWidth(text), 0))
	return highlightStyle(r).Render(text)
}

// pair draws "label  value" padded to exactly width columns, with the value
// starting after labelWidth columns of label (0 means the label's own width).
func pair(label, value string, labelWidth, width int) string {
	labelWidth = max(labelWidth, ansi.StringWidth(label))
	head := label + strings.Repeat(" ", labelWidth-ansi.StringWidth(label)+labelGap)
	room := max(width-ansi.StringWidth(head), 0)
	value = ansi.Truncate(value, room, "…")
	pad := strings.Repeat(" ", max(width-ansi.StringWidth(head)-ansi.StringWidth(value), 0))
	return labelStyle.Render(head) + plain.Render(value) + pad
}
