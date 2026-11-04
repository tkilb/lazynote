package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name, in  string
		wantKind  Kind
		wantValue string
	}{
		{"empty means the clipboard", "", Clipboard, ""},
		{"white space means the clipboard", "  \t ", Clipboard, ""},
		{"https address", "https://example.com/a", Web, "https://example.com/a"},
		{"http address", "http://example.com", Web, "http://example.com"},
		{"scheme in capitals", "HTTPS://Example.com/A", Web, "HTTPS://Example.com/A"},
		{"address with padding", "  https://example.com/a  ", Web, "https://example.com/a"},
		{"a broken address is still a web import", "https://", Web, "https://"},
		{"home path", "~/Downloads/meeting.txt", File, "~/Downloads/meeting.txt"},
		{"absolute path", "/tmp/a b.txt", File, "/tmp/a b.txt"},
		{"padded path", "  notes.md ", File, "notes.md"},
		{"a bare name", "standup", File, "standup"},
		{"a name that starts with http", "httpd-notes.md", File, "httpd-notes.md"},
		{"another scheme is a file name", "ftp://host/file", File, "ftp://host/file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.in)
			assert.Equal(t, tt.wantKind, got.Kind)
			assert.Equal(t, tt.wantValue, got.Value)
		})
	}
}
