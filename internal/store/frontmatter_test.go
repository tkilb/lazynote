package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantRaw  string
		wantBody string
		wantOK   bool
	}{
		{"basic", "---\na: 1\n---\nbody\n", "a: 1\n", "body\n", true},
		{"empty block", "---\n---\nbody", "", "body", true},
		{"no body", "---\na: 1\n---\n", "a: 1\n", "", true},
		{"closer at EOF without newline", "---\na: 1\n---", "a: 1\n", "", true},
		{"crlf", "---\r\na: 1\r\n---\r\nbody\r\n", "a: 1\r\n", "body\r\n", true},
		{"bom", "\xef\xbb\xbf---\na: 1\n---\nbody", "a: 1\n", "body", true},
		{"body keeps later delimiters", "---\na: 1\n---\nx\n---\ny\n", "a: 1\n", "x\n---\ny\n", true},
		{"no frontmatter", "# Title\ntext\n", "", "# Title\ntext\n", false},
		{"empty file", "", "", "", false},
		{"unclosed block is body", "---\na: 1\nbody\n", "", "---\na: 1\nbody\n", false},
		{"opener with trailing text", "--- x\na: 1\n---\n", "", "--- x\na: 1\n---\n", false},
		{"delimiter not first line", "\n---\na: 1\n---\n", "", "\n---\na: 1\n---\n", false},
		{"four dashes", "----\na: 1\n----\n", "", "----\na: 1\n----\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, body, ok := splitFrontmatter([]byte(tt.in))
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantRaw, string(raw))
			assert.Equal(t, tt.wantBody, string(body))
		})
	}
}

func TestFrontmatterRoundTrip(t *testing.T) {
	src := `zeta: last-alphabetically
uid: 5b0c3f7e-0000-4000-8000-000000000000
# a comment between keys
tags:
  - work
  - ideas
aliases: [Alpha, Beta]
custom:
  nested:
    deep: true
  list:
    - 1
    - two
kind: fact
related:
  - "[[Project Alpha]]"
empty:
date: 2026-10-29
`
	fm, err := ParseFrontmatter([]byte(src))
	require.NoError(t, err)

	out, err := fm.Marshal()
	require.NoError(t, err)

	// A second round trip is byte-stable, and the first keeps every key in
	// file order with its comment.
	again, err := ParseFrontmatter(out)
	require.NoError(t, err)
	out2, err := again.Marshal()
	require.NoError(t, err)
	assert.Equal(t, string(out), string(out2))

	got := string(out)
	order := []string{"zeta:", "uid:", "# a comment between keys", "tags:", "aliases:", "custom:", "kind:", "related:", "empty:", "date:"}
	last := -1
	for _, want := range order {
		idx := strings.Index(got[last+1:], want)
		require.GreaterOrEqual(t, idx, 0, "%q missing or out of order in:\n%s", want, got)
		last += idx + 1
	}
	assert.Contains(t, got, "deep: true")
	assert.Contains(t, got, "[[Project Alpha]]")
	assert.Contains(t, got, "2026-10-29")
}

func TestFrontmatterGet(t *testing.T) {
	fm, err := ParseFrontmatter([]byte("title: Hello\ntags: [a, b]\nn: 3\n"))
	require.NoError(t, err)

	assert.Equal(t, "Hello", fm.Get("title").Value)
	assert.Len(t, fm.Get("tags").Content, 2)
	assert.Equal(t, "3", fm.Get("n").Value)
	assert.Nil(t, fm.Get("missing"))
	assert.Nil(t, fm.Get("Title"), "keys are case-sensitive")
}

func TestParseFrontmatterEmpty(t *testing.T) {
	for _, in := range []string{"", "\n", "# only a comment\n"} {
		fm, err := ParseFrontmatter([]byte(in))
		require.NoError(t, err, "%q", in)
		assert.Nil(t, fm.Get("x"))
		out, err := fm.Marshal()
		require.NoError(t, err)
		assert.Empty(t, out)
	}
}

func TestParseFrontmatterErrors(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"list", "- a\n- b\n", ErrFrontmatterNotMapping},
		{"scalar", "just text\n", ErrFrontmatterNotMapping},
		{"malformed", "a: [unclosed\n", nil},
		{"bad indent", "a: 1\n  b: 2\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := ParseFrontmatter([]byte(tt.in))
			require.Error(t, err)
			assert.Nil(t, fm)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
