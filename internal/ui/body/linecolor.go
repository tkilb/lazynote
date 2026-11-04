package body

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// lineKind says how a source line of the plain view is colored, to match the
// rendered view (see gruvbox.go): a heading by its level, a code line, a
// fence, a quote or a rule. Everything else keeps the default color. The
// kind is worked out once per note, because a line inside a code block
// depends on the fence above it.
type lineKind int

const (
	kindText lineKind = iota
	kindH1
	kindH2
	kindH3
	kindH4
	kindH5
	kindH6
	kindCode  // a line inside a code block
	kindFence // the ``` or ~~~ line that opens or closes a code block
	kindQuote
	kindRule
)

var (
	headingRe = regexp.MustCompile(`^ {0,3}(#{1,6})(\s|$)`)
	fenceRe   = regexp.MustCompile("^ {0,3}(```|~~~)")
	quoteRe   = regexp.MustCompile(`^ {0,3}>`)
)

// lineStyles color each kind, in the gruvbox palette of the rendered view.
var lineStyles = map[lineKind]lipgloss.Style{
	kindH1:    headingLine(gbRed),
	kindH2:    headingLine(gbOrange),
	kindH3:    headingLine(gbYellow),
	kindH4:    headingLine(gbGreen),
	kindH5:    headingLine(gbBlue),
	kindH6:    headingLine(gbPurple),
	kindCode:  lipgloss.NewStyle().Foreground(lipgloss.Color(gbAqua)),
	kindFence: lipgloss.NewStyle().Foreground(lipgloss.Color(gbGrey1)),
	kindQuote: lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color(gbGrey1)),
	kindRule:  lipgloss.NewStyle().Foreground(lipgloss.Color(gbGrey0)),
}

func headingLine(color string) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
}

// classify returns the kind of each source line.
func classify(lines []string) []lineKind {
	kinds := make([]lineKind, len(lines))
	inCode := false
	for i, line := range lines {
		switch {
		case fenceRe.MatchString(line):
			kinds[i], inCode = kindFence, !inCode
		case inCode:
			kinds[i] = kindCode
		case headingRe.MatchString(line):
			kinds[i] = kindH1 + lineKind(len(headingRe.FindStringSubmatch(line)[1])-1)
		case isRule(line):
			kinds[i] = kindRule
		case quoteRe.MatchString(line):
			kinds[i] = kindQuote
		}
	}
	return kinds
}

// isRule reports whether line is a horizontal rule: three or more of one of
// - * _, with only spaces between.
func isRule(line string) bool {
	s := strings.TrimSpace(line)
	if len(s) < 3 || strings.IndexByte("-*_", s[0]) < 0 {
		return false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case s[0]:
			n++
		case ' ':
		default:
			return false
		}
	}
	return n >= 3
}

// colorRow draws one wrapped row of a source line of the given kind.
func colorRow(row string, kind lineKind) string {
	if style, ok := lineStyles[kind]; ok {
		return style.Render(row)
	}
	return row
}
