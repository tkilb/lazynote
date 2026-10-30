package textsafe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "hello world", "hello world"},
		{"newlines kept", "a\nb\n\nc", "a\nb\n\nc"},
		{"tab becomes spaces", "a\tb", "a    b"},
		{"carriage returns dropped", "a\r\nb\rc", "a\nbc"},
		{"escape sequence", "x\x1b[2Jy", "x[2Jy"},
		{"osc title with bel", "x\x1b]0;pwned\x07y", "x]0;pwnedy"},
		{"c1 csi", "x\u009b31my", "x31my"},
		{"nul and del", "a\x00b\x7fc", "abc"},
		{"unicode kept", "héllo — 日本語 🙂", "héllo — 日本語 🙂"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Sanitize(tt.in))
		})
	}
}

func TestSanitizeLeavesNoControlBytes(t *testing.T) {
	var all []byte
	for b := 0; b < 256; b++ {
		all = append(all, byte(b))
	}
	for _, r := range Sanitize(string(all)) {
		assert.True(t, r == '\n' || r >= ' ' && r != 0x7f && (r < 0x80 || r > 0x9f), "control rune %U survived", r)
	}
}

func TestSingleLine(t *testing.T) {
	assert.Equal(t, "a b c", SingleLine("a\nb\r\nc"))
	assert.Equal(t, "no escape", SingleLine("no\x1b escape"))
}
