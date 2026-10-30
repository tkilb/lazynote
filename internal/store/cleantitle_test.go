package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Call Mum about the thing", "Call Mum about the thing"},
		{"  spaced   out   words  ", "spaced out words"},
		{`a/b\c:d*e?f"g<h>i|j`, "abcdefghij"},
		{"C# [notes] ^caret #tag", "C notes caret tag"},
		{"..hidden", "hidden"},
		{"trailing dots...", "trailing dots"},
		{"Héllo Wörld", "Héllo Wörld"},
		{"", ""},
		{"///", ""},
		{strings.Repeat("x", 200), strings.Repeat("x", 80)},
		{"tab\tand\nnewline", "tab and newline"},
		{"nul\x00byte", "nulbyte"},
		{"keeps case AND punctuation, ok!", "keeps case AND punctuation, ok!"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, CleanTitle(tt.in))
		})
	}
}
