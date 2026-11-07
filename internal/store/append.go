package store

import (
	"bytes"
	"strings"

	"github.com/tkilb/para/internal/undo"
)

// Append adds text to the end of the note at rel, after one blank line (none
// when the note is empty). The existing bytes, properties included, are not
// changed or reordered. Undo restores the exact previous bytes and refuses
// (ErrChanged) when the note was changed meanwhile. Text that is only
// whitespace is ErrNothingToAppend.
func (m *Markdown) Append(rel, text string) (undo.Action, error) {
	text = strings.Trim(text, "\r\n")
	if strings.TrimSpace(text) == "" {
		return undo.Action{}, ErrNothingToAppend
	}
	path, info, before, err := m.readForEdit(rel)
	if err != nil {
		return undo.Action{}, err
	}
	after := appendedNote(before, text)
	if len(after) > maxNoteBytes {
		return undo.Action{}, ErrTooLarge
	}
	return rewriteNote("append to", path, info, before, after)
}

// appendedNote returns data followed by a blank line and text, ending in one
// line break. A note of zero bytes just becomes text.
func appendedNote(data []byte, text string) []byte {
	out := bytes.Clone(data)
	if len(out) == 0 {
		return []byte(text + "\n")
	}
	if !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	out = append(out, '\n')
	return append(out, text+"\n"...)
}
