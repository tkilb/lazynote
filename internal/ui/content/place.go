package content

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// snippetRunes is how much of a line is used to find it in the other view.
const snippetRunes = 24

// minSnippetRunes is the least text worth searching for; shorter lines (a
// rule, a lone bracket) are placed by their position in the note instead.
const minSnippetRunes = 3

// stripMarks removes what Markdown source and glamour's output differ by:
// emphasis marks, bullets, quote bars and task boxes.
var stripMarks = strings.NewReplacer("*", "", "_", "", "`", "", "~", "", "•", "", "│", "",
	"[✓]", "", "[ ]", "", "[x]", "", "[X]", "", "\t", " ")

// snippet is the start of s with marks, leading heading, quote and list
// signs, and runs of spaces removed, or "" when too little is left to search
// for.
func snippet(s string) string {
	s = strings.Join(strings.Fields(stripMarks.Replace(s)), " ")
	s = strings.TrimLeft(s, "#>-+ ")
	if utf8.RuneCountInString(s) < minSnippetRunes {
		return ""
	}
	if r := []rune(s); len(r) > snippetRunes {
		s = string(r[:snippetRunes])
	}
	return s
}

// guess places index i of from items at the same share of to items.
func guess(i, from, to int) int {
	if from <= 0 || to <= 0 {
		return 0
	}
	return min(max(i*to/from, 0), to-1)
}

// nearest returns, of the lines in lines whose text contains snip, the one
// closest to index est, or est when none does.
func nearest(snip string, lines []string, est int) int {
	best := -1
	for i, line := range lines {
		if strings.Contains(strings.Join(strings.Fields(stripMarks.Replace(line)), " "), snip) {
			if best < 0 || abs(i-est) < abs(best-est) {
				best = i
			}
		}
	}
	if best < 0 {
		return est
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// srcForRow finds the source line that rendered row r came from: for a
// frontmatter row, its line; for the text, the line holding the row's text,
// nearest to where the row sits in the text, else that position. Rendering
// rewrites lines, so for the text this is a good guess, not a map.
func (m Model) srcForRow(r int) int {
	nRows, nFront := len(m.frontRows), len(m.front)
	if r >= 0 && r < nRows {
		return m.frontSrc[r]
	}
	rows, lines := m.rendered[min(nRows, len(m.rendered)):], m.srcLines[min(nFront, len(m.srcLines)):]
	r -= nRows
	est := guess(r, len(rows), len(lines))
	if r < 0 || r >= len(rows) {
		return nFront + est
	}
	snip := snippet(ansi.Strip(rows[r]))
	if snip == "" {
		return nFront + est
	}
	return nFront + nearest(snip, lines, est)
}

// rowForSrc is srcForRow the other way: the rendered row that shows source
// line src.
func (m Model) rowForSrc(src int) int {
	nRows, nFront := len(m.frontRows), len(m.front)
	if src < nFront {
		for i, s := range m.frontSrc {
			if s == max(src, 0) {
				return i
			}
		}
		return 0
	}
	rows, lines := m.rendered[min(nRows, len(m.rendered)):], m.srcLines[nFront:]
	src -= nFront
	est := guess(src, len(lines), len(rows))
	if src >= len(lines) {
		return nRows + est
	}
	snip := snippet(lines[src])
	if snip == "" {
		return nRows + est
	}
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = ansi.Strip(r)
	}
	return nRows + nearest(snip, plain, est)
}
