package noteform

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// typeSelect handles a key in a chooser whose entries are labels. Entries
// before from are never matched by typing: the Kind and Template choosers
// pass 1 so their "nothing" entry (no kind, none) is skipped, and the Source
// chooser passes 0. Typing letters jumps to the first matching entry whose
// label starts with everything typed so far ("p" then "l" is plan); a letter
// that matches nothing is ignored. Backspace removes the last letter, and
// emptying it (or pressing backspace with nothing typed) goes back to
// labels[0]. Left and Right still cycle, and reset what was typed. It returns
// the new position and typed letters.
func typeSelect(msg tea.Msg, labels []string, idx int, typed string, from int) (int, string) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return idx, typed
	}
	switch {
	case key.Type == tea.KeyBackspace:
		if typed != "" {
			_, size := utf8.DecodeLastRuneInString(typed)
			typed = typed[:len(typed)-size]
		}
		if typed == "" {
			return 0, ""
		}
		return matchLabel(labels, typed, idx, from), typed
	case key.Type == tea.KeyRunes && !key.Alt:
		next := typed + strings.ToLower(string(key.Runes))
		if i := matchLabel(labels, next, -1, from); i >= 0 {
			return i, next
		}
		return idx, typed
	}
	if moved := cycle(msg, idx, len(labels)); moved != idx {
		return moved, ""
	}
	return idx, typed
}

// matchLabel returns the index of the first label from index from on that
// starts with prefix, ignoring case, or fallback when none does.
func matchLabel(labels []string, prefix string, fallback, from int) int {
	for i := from; i < len(labels); i++ {
		if strings.HasPrefix(strings.ToLower(labels[i]), prefix) {
			return i
		}
	}
	return fallback
}

// kindLabels are the kind chooser's entries, the unset kind first.
func (m Model) kindLabels() []string {
	labels := make([]string, len(m.kinds))
	for i, k := range m.kinds {
		labels[i] = k.String()
	}
	return labels
}

// templateLabels are the Template chooser's entries, "none" first.
func (m Model) templateLabels() []string {
	return append([]string{noTemplate}, m.templates...)
}
