package templates

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkilb/para/internal/store"
)

func write(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestListMergesSharedAndNotebookTemplates(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "shared.md", "x")
	write(t, dir, "both.md", "shared")
	write(t, dir, "daily.md", "x")
	write(t, dir, "notes.txt", "x")
	write(t, dir, ".hidden.md", "x")
	write(t, dir, "work/both.md", "work")
	write(t, dir, "work/decision-record.md", "x")
	write(t, dir, "work/daily.md", "x")
	write(t, dir, "personal/journal.md", "x")

	tests := []struct {
		name     string
		notebook string
		want     []string
	}{
		{"work", "work", []string{"both", "decision-record", "shared"}},
		{"personal is isolated from work", "personal", []string{"both", "journal", "shared"}},
		{"unknown notebook gets shared only", "other", []string{"both", "shared"}},
		{"blank notebook gets shared only", "", []string{"both", "shared"}},
		{"parent path is not a notebook", "..", []string{"both", "shared"}},
		{"separator is not a notebook", "a/b", []string{"both", "shared"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := List(dir, tt.notebook)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListWithMissingFolderIsEmpty(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "none"), "work")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	shared := write(t, dir, "both.md", "shared")
	own := write(t, dir, "work/both.md", "work")
	only := write(t, dir, "shared-only.md", "x")
	write(t, dir, "daily.md", "x")

	tests := []struct {
		name, notebook, tmpl, want string
		err                        error
	}{
		{"notebook overrides shared", "work", "both", own, nil},
		{"shared for another notebook", "personal", "both", shared, nil},
		{"shared only", "work", "shared-only", only, nil},
		{"missing", "work", "nope", "", ErrNotFound},
		{"daily is never offered", "work", "daily", "", ErrNotFound},
		{"traversal is not a name", "work", "../both", "", ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(dir, tt.notebook, tt.tmpl)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDailyPath(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string // relative to dir; "" for none
	}{
		{"none", nil, ""},
		{"old location", []string{"daily.md"}, "daily.md"},
		{"notebook folder", []string{"work/daily.md"}, "work/daily.md"},
		{"notebook folder wins", []string{"daily.md", "work/daily.md"}, "work/daily.md"},
		{"other notebook is ignored", []string{"personal/daily.md"}, ""},
		{"other notebook falls back to old location", []string{"daily.md", "personal/daily.md"}, "daily.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				write(t, dir, f, "x")
			}
			want := ""
			if tt.want != "" {
				want = filepath.Join(dir, tt.want)
			}
			assert.Equal(t, want, DailyPath(dir, "work"))
		})
	}
}

func TestDailyPathIgnoresADirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "work", "daily.md"), 0o755))
	assert.Empty(t, DailyPath(dir, "work"))
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Template
		err  error
	}{
		{"properties and body",
			"---\nkind: fact\nsource: meeting\ntags: [adr, \"team x\"]\n---\n\n# {{title}}\n\nBody\n\n\n",
			Template{Name: "t", Kind: store.KindFact, Source: "meeting", Tags: []string{"adr", "team x"}, Body: "# {{title}}\n\nBody"},
			nil},
		{"tags as one value", "---\ntags: solo\n---\nText\n",
			Template{Name: "t", Kind: store.KindUnset, Tags: []string{"solo"}, Body: "Text"}, nil},
		{"no properties", "Just text\n", Template{Name: "t", Kind: store.KindUnset, Body: "Just text"}, nil},
		{"properties only", "---\nkind: note\n---\n\n  \n", Template{}, ErrEmptyBody},
		{"empty file", "", Template{}, ErrEmptyBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse("t", []byte(tt.text))
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Body, got.Body)
			assert.Equal(t, tt.want.Source, got.Source)
			assert.Equal(t, tt.want.Tags, got.Tags)
			assert.Equal(t, tt.want.Kind.String(), got.Kind.String())
		})
	}
}

func TestParseRejectsBrokenProperties(t *testing.T) {
	_, err := Parse("t", []byte("---\nkind: [unclosed\n---\nBody\n"))
	require.Error(t, err)
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()
	big := write(t, dir, "big.md", "")
	require.NoError(t, os.Truncate(big, maxBytes+1))

	tests := []struct {
		name string
		path string
		err  error
	}{
		{"missing", filepath.Join(dir, "missing.md"), os.ErrNotExist},
		{"directory", dir, ErrNotRegular},
		{"too large", big, ErrTooLarge},
		{"empty body", write(t, dir, "e.md", "---\nkind: note\n---\n"), ErrEmptyBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load("x", tt.path)
			require.ErrorIs(t, err, tt.err)
		})
	}
	got, err := Load("ok", write(t, dir, "ok.md", "Hello\n"))
	require.NoError(t, err)
	assert.Equal(t, "Hello", got.Body)
}

func TestRender(t *testing.T) {
	day := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC) // a Wednesday
	tests := []struct {
		name, body, title, want string
	}{
		{"all three", "{{date}} {{weekday}} {{title}}", "Plan", "2026-11-04 Wednesday Plan"},
		{"repeated", "{{date}}/{{date}}", "", "2026-11-04/2026-11-04"},
		{"unknown stays", "{{who}} {{Date}} {{ date }}", "T", "{{who}} {{Date}} {{ date }}"},
		{"blank title stays", "# {{title}}", "", "# {{title}}"},
		{"no example-day heuristic", "Written 2020-01-01 (Monday)", "T", "Written 2020-01-01 (Monday)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Render(tt.body, Vars{Day: day, Title: tt.title}))
		})
	}
}
