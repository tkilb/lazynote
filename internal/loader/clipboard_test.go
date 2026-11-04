package loader

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clipboardOf(text string) func() (string, error) {
	return func() (string, error) { return text, nil }
}

func TestReadClipboard(t *testing.T) {
	got, err := ReadClipboard(clipboardOf("Anna: here is the thread\nSam: thanks"))
	require.NoError(t, err)
	assert.Equal(t, Clipboard, got.Kind)
	assert.Equal(t, "clipboard", got.Name)
	assert.Equal(t, "Anna: here is the thread\nSam: thanks", got.Text)
	assert.Empty(t, got.Origin)
}

func TestReadClipboardFailures(t *testing.T) {
	const secret = "SECRET-CLIPBOARD-TEXT"
	tests := []struct {
		name string
		read func() (string, error)
		want error
	}{
		{"cannot read", func() (string, error) { return secret, errors.New("pbpaste: " + secret) }, ErrClipboard},
		{"empty", clipboardOf(""), ErrNothingToImport},
		{"only white space", clipboardOf(" \n\t "), ErrNothingToImport},
		{"binary", clipboardOf("a\x00" + secret), ErrNotText},
		{"too large", clipboardOf(strings.Repeat("a", MaxBytes+1)), ErrTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadClipboard(tt.read)
			require.ErrorIs(t, err, tt.want)
			assert.NotContains(t, err.Error(), "SECRET", "errors never show clipboard text")
		})
	}
}
