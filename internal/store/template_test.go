package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleDaily = "---\nuid: 18e0ff11\ntitle: daily\nkind: \"\"\nsource: daily\ndate: 2026-11-02\ntags: [daily]\npeople: []\nrelated: []\n---\n\n# Daily Note — 2026-11-02 (Monday)\n\n## Goals\n\n- [ ] -\n\nOn Mondays we meet. Monday standup.\n"

var wed = time.Date(2026, 11, 4, 9, 30, 0, 0, time.Local)

func TestTemplatedNoteWithASampleDay(t *testing.T) {
	out, err := TemplatedNote([]byte(sampleDaily), "2026-11-04.md", "daily", wed)
	require.NoError(t, err)
	n := ParseNote("2026-11-04.md", out)
	require.NoError(t, n.FrontmatterErr)

	assert.NotEqual(t, "18e0ff11", n.Frontmatter.Get("uid").Value, "a note made from a template gets its own uid")
	assert.NotEmpty(t, n.Frontmatter.Get("uid").Value)
	assert.Equal(t, "2026-11-04", n.Frontmatter.Get("title").Value, "the title is the file name")
	assert.Equal(t, "2026-11-04", n.Frontmatter.Get("date").Value)
	assert.Equal(t, "daily", n.Frontmatter.Get("source").Value)
	assert.Len(t, n.Frontmatter.Get("tags").Content, 1, "the template's other properties are kept")
	assert.Contains(t, n.Body, "# Daily Note — 2026-11-04 (Wednesday)", "the sample date and weekday become the day's")
	assert.Contains(t, n.Body, "- [ ] -")
	assert.Contains(t, n.Body, "On Mondays we meet. Monday standup.", "only the weekday of the sample date, as a whole word, in parentheses after it, changes")
	assert.Equal(t, append(append([]string(nil), standardKeys...), "related"), keysOf(t, out))
}

func TestTemplatedNoteWithPlaceholders(t *testing.T) {
	tmpl := "---\ndate: 2020-01-01\n---\n# {{date}} ({{weekday}})\n\nWritten 2020-01-01 stays.\n"
	out, err := TemplatedNote([]byte(tmpl), "2026-11-04.md", "daily", wed)
	require.NoError(t, err)
	n := ParseNote("x.md", out)
	assert.Equal(t, "# 2026-11-04 (Wednesday)\n\nWritten 2020-01-01 stays.\n", n.Body, "with placeholders, nothing else is replaced")
	assert.Equal(t, "2026-11-04", n.Frontmatter.Get("date").Value)
}

func TestTemplatedNoteWithoutProperties(t *testing.T) {
	out, err := TemplatedNote([]byte("# Notes\n\n- a\n"), "2026-11-04.md", "daily", wed)
	require.NoError(t, err)
	n := ParseNote("x.md", out)
	assert.Equal(t, "# Notes\n\n- a\n", n.Body)
	assert.Equal(t, "2026-11-04", n.Frontmatter.Get("date").Value)
	assert.Equal(t, "daily", n.Frontmatter.Get("source").Value, "a missing source is the given one")
	assert.NotEmpty(t, n.Frontmatter.Get("uid").Value)
}

func TestTemplatedNoteKeepsAnExistingSource(t *testing.T) {
	out, err := TemplatedNote([]byte("---\nsource: meeting\n---\nbody"), "d.md", "daily", wed)
	require.NoError(t, err)
	assert.Equal(t, "meeting", ParseNote("d.md", out).Frontmatter.Get("source").Value)
}

func TestTemplatedNoteRefusesUnreadableProperties(t *testing.T) {
	_, err := TemplatedNote([]byte("---\nkey: [oops\n---\nbody"), "d.md", "daily", wed)
	assert.ErrorIs(t, err, ErrFrontmatterInvalid)
}

func TestTemplatedNoteLeavesTheBodyAloneWithNoSampleDay(t *testing.T) {
	out, err := TemplatedNote([]byte("---\ntags: [x]\n---\nMonday 2026-11-02 stays\n"), "d.md", "daily", wed)
	require.NoError(t, err)
	assert.Equal(t, "Monday 2026-11-02 stays\n", ParseNote("d.md", out).Body)
}
