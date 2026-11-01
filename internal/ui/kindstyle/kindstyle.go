// Package kindstyle gives each note kind its short badge text and color, so
// every panel shows a kind the same way.
package kindstyle

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/tkilb/lazynote/internal/store"
)

// BadgeWidth is the width of every badge; shorter ones are padded.
const BadgeWidth = 4

// UnsetBadge stands in for a note whose kind is unset.
const UnsetBadge = "----"

// unsetColor is a dim gray.
var unsetColor = lipgloss.Color("245")

var styles = map[store.Kind]struct {
	badge string
	color lipgloss.Color
}{
	store.KindFact:     {"fact", lipgloss.Color("2")},
	store.KindRumor:    {"rumr", lipgloss.Color("3")},
	store.KindProposal: {"prop", lipgloss.Color("6")},
	store.KindFossil:   {"foss", lipgloss.Color("8")},
	store.KindIdea:     {"idea", lipgloss.Color("4")},
	store.KindDraft:    {"draf", lipgloss.Color("5")},
	store.KindWeb:      {"web", lipgloss.Color("111")},
}

// Badge returns the four-character badge and color for k.
func Badge(k store.Kind) (string, lipgloss.Color) {
	if s, ok := styles[k]; ok {
		return fmt.Sprintf("%-*s", BadgeWidth, s.badge), s.color
	}
	return UnsetBadge, unsetColor
}

// Color returns the color for k.
func Color(k store.Kind) lipgloss.Color {
	_, c := Badge(k)
	return c
}
