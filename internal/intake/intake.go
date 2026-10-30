// Package intake creates the notes that arrive in the inbox: a line of
// quick capture, or a file imported as a transcript or as a note. Every
// intake note gets a uid, a title equal to its file name, a kind, a source,
// a date and a tags list. Nothing here reads or writes outside the notebook
// except one plain read of the file being imported.
package intake

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/undo"
)

const (
	// SourceAdhoc is the built-in default source of a note added by hand; the
	// config can change the list Quick Add offers.
	SourceAdhoc = "adhoc"
	// SourceTranscript is the source of an imported transcript.
	SourceTranscript = "transcript"
	// SourceExtract is the source of a note extracted from another note.
	SourceExtract = "extract"

	inboxDir     = "inbox"
	dayLayout    = "2006-01-02"
	noteExt      = ".md"
	maxNameTries = 1000
)

// Errors never echo a path, because they are shown in the interface.
var (
	ErrEmpty      = errors.New("a title is needed")
	ErrNotFound   = errors.New("file not found")
	ErrNotFile    = errors.New("not a regular file")
	ErrNotNote    = errors.New("not a markdown note")
	ErrTooLarge   = errors.New("file is too large")
	ErrUnreadable = errors.New("cannot read the file")
	ErrNoFreeName = errors.New("could not find a free name")
	ErrNoText     = errors.New("no text selected")
	ErrNoKind     = errors.New("a kind is needed")
)

// Result describes a note that was created.
type Result struct {
	// Rel is the new note's path relative to the notebook root.
	Rel string
	// Action undoes (and redoes) the creation.
	Action undo.Action
	// Unlinked is set when an extracted note could not link back to its
	// origin, because the origin's name has characters a link cannot hold.
	Unlinked bool
	// Unchanged is set when an imported note was saved exactly as it was,
	// because its properties could not be parsed: it keeps its original uid.
	Unchanged bool
}

// Name returns the file name of an intake note titled title: the cleaned
// title with .md, or "" when nothing usable is left.
func Name(title string) string {
	clean := store.CleanTitle(title)
	if clean == "" {
		return ""
	}
	return clean + noteExt
}

// Capture creates an inbox note named after its title, holding body, with the
// given kind (KindUnset for none), source and tags. A missing title is
// ErrEmpty.
func Capture(s store.Store, now time.Time, kind store.Kind, source, title, body string, tags []string) (Result, error) {
	name := Name(title)
	if name == "" {
		return Result{}, ErrEmpty
	}
	text := ""
	if body = strings.TrimSpace(body); body != "" {
		text = body + "\n"
	}
	return createUnique(s, name, func(final string) ([]byte, error) {
		return store.IntakeNote(final, kind, source, now.Format(dayLayout), text, tags)
	})
}

// ImportTranscript reads the file at path and creates an inbox note, named
// after the file, whose body is its text, unmodified, with source transcript
// and kind rumor (raw and unconfirmed until reviewed).
func ImportTranscript(s store.Store, now time.Time, path string) (Result, error) {
	text, err := readSource(path)
	if err != nil {
		return Result{}, err
	}
	base := filepath.Base(expandHome(path))
	name := Name(strings.TrimSuffix(base, filepath.Ext(base)))
	if name == "" {
		name = Name(SourceTranscript)
	}
	return createUnique(s, name, func(final string) ([]byte, error) {
		return store.IntakeNote(final, store.KindRumor, SourceTranscript, now.Format(dayLayout), string(text), nil)
	})
}

// ImportNote reads the markdown file at path and copies it into the inbox
// under its own name, keeping its properties, adding source (import) and
// date when missing, and giving it a fresh uid and a title equal to its new
// name. A file whose properties cannot be parsed is saved as it is
// (Result.Unchanged).
func ImportNote(s store.Store, now time.Time, path string) (Result, error) {
	data, err := readSource(path)
	if err != nil {
		return Result{}, err
	}
	base := filepath.Base(expandHome(path))
	if !strings.EqualFold(filepath.Ext(base), noteExt) || strings.EqualFold(base, noteExt) {
		return Result{}, ErrNotNote
	}
	var unchanged bool
	res, err := createUnique(s, base, func(final string) ([]byte, error) {
		out, err := store.ImportedNote(data, final, now.Format(dayLayout))
		if errors.Is(err, store.ErrFrontmatterInvalid) {
			unchanged = true
			return data, nil
		}
		return out, err
	})
	res.Unchanged = unchanged
	return res, err
}

// createUnique creates an inbox note named name, or "name 2", "name 3" and
// so on when that is taken. contents makes the bytes for the name finally
// used.
func createUnique(s store.Store, name string, contents func(final string) ([]byte, error)) (Result, error) {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	for n := 1; n <= maxNameTries; n++ {
		final := name
		if n > 1 {
			final = fmt.Sprintf("%s %d%s", stem, n, ext)
		}
		data, err := contents(final)
		if err != nil {
			return Result{}, err
		}
		rel := filepath.Join(inboxDir, final)
		act, err := s.Create(rel, data)
		if errors.Is(err, store.ErrExists) {
			continue
		}
		if err != nil {
			return Result{}, err
		}
		return Result{Rel: rel, Action: act}, nil
	}
	return Result{}, ErrNoFreeName
}

// expandHome replaces a leading ~ with the home directory.
func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

// readSource reads a regular file from anywhere, up to store.MaxNoteBytes.
func readSource(path string) ([]byte, error) {
	path = expandHome(path)
	if path == "" || strings.ContainsRune(path, 0) {
		return nil, ErrNotFound
	}
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNotFound
	case err != nil:
		return nil, ErrUnreadable
	case !info.Mode().IsRegular():
		return nil, ErrNotFile
	case info.Size() > store.MaxNoteBytes:
		return nil, ErrTooLarge
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, ErrUnreadable
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, store.MaxNoteBytes+1))
	switch {
	case err != nil:
		return nil, ErrUnreadable
	case len(data) > store.MaxNoteBytes:
		return nil, ErrTooLarge
	}
	return data, nil
}

// SplitText splits text, such as pasted from the clipboard, into a title and
// a body: the title is its first non-blank line, trimmed, and the body is
// what follows it with leading blank lines dropped. A single line is all
// title.
func SplitText(text string) (title, body string) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rest := strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
		return strings.TrimSpace(line), rest
	}
	return "", ""
}

// Extract creates an inbox note holding text taken from the note at origin,
// with the chosen kind. It links back to the origin with a related property,
// except when the origin is a dated daily note: that will be deleted, so the
// new note records source "daily YYYY-MM-DD" as plain text instead. The origin
// is never changed.
func Extract(s store.Store, now time.Time, kind store.Kind, origin, title, text string) (Result, error) {
	name := Name(title)
	text = strings.TrimSpace(text)
	switch {
	case kind == store.KindUnset:
		return Result{}, ErrNoKind
	case name == "":
		return Result{}, ErrEmpty
	case text == "":
		return Result{}, ErrNoText
	}

	source := SourceExtract
	var related []string
	var unlinked bool
	if date, daily := store.DailyDate(origin); daily {
		source = "daily " + date
	} else if link := originLink(origin); link != "" {
		related = []string{link}
	} else {
		unlinked = true
	}
	res, err := createUnique(s, name, func(final string) ([]byte, error) {
		return store.ExtractedNote(final, kind, source, now.Format(dayLayout), text+"\n", related)
	})
	res.Unlinked = unlinked
	return res, err
}

// originLink is the wikilink to the note at rel, or "" when its name has
// characters a link cannot hold.
func originLink(rel string) string {
	base := filepath.Base(rel)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || strings.ContainsAny(stem, "|#[]^") {
		return ""
	}
	return "[[" + stem + "]]"
}
