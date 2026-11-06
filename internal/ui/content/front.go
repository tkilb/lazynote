package content

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/tkilb/para/internal/textsafe"
)

// cleanFront makes the frontmatter lines safe to show: each is one line of
// printable text. An empty list stays empty.
func cleanFront(front []string) []string {
	if len(front) == 0 {
		return nil
	}
	out := make([]string, len(front))
	for i, l := range front {
		out[i] = textsafe.SingleLine(l)
	}
	return out
}

// frontKinds colors the frontmatter lines in the plain view: the --- lines
// like a code fence, the rest in the frontmatter color.
func frontKinds(front []string) []lineKind {
	kinds := make([]lineKind, len(front))
	for i, l := range front {
		kinds[i] = kindFront
		if strings.TrimSpace(l) == "---" {
			kinds[i] = kindFence
		}
	}
	return kinds
}

// renderFront draws the frontmatter lines as rendered rows, colored as in the
// plain view, wrapped to width, with a blank row after. It also returns the
// frontmatter line each row came from (the blank row belongs to the last).
func renderFront(front []string, width int) (rows []string, src []int) {
	if len(front) == 0 {
		return nil, nil
	}
	kinds := frontKinds(front)
	for i, line := range front {
		for _, row := range strings.Split(ansi.Wrap(line, width, ""), "\n") {
			rows = append(rows, colorRow(row, kinds[i]))
			src = append(src, i)
		}
	}
	return append(rows, ""), append(src, len(front)-1)
}
