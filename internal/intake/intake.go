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
	"unicode"

	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/undo"
)

const (
	// SourceQuick is the source of a note typed into a quick capture prompt.
	SourceQuick = "quick"
	// SourceTranscript is the source of an imported transcript.
	SourceTranscript = "transcript"

	inboxDir      = "inbox"
	stampLayout   = "2006-01-02-150405"
	dayLayout     = "2006-01-02"
	noteExt       = ".md"
	slugWords     = 6
	slugMaxRunes  = 40
	maxNameTries  = 1000
	slugDropChars = `/\:*?"<>|#[]^`
)

// Errors never echo a path, because they are shown in the interface.
var (
	ErrEmpty      = errors.New("nothing to capture")
	ErrNotFound   = errors.New("file not found")
	ErrNotFile    = errors.New("not a regular file")
	ErrNotNote    = errors.New("not a markdown note")
	ErrTooLarge   = errors.New("file is too large")
	ErrUnreadable = errors.New("cannot read the file")
	ErrNoFreeName = errors.New("could not find a free name")
)

// Result describes a note that was created.
type Result struct {
	// Rel is the new note's path relative to the notebook root.
	Rel string
	// Action undoes (and redoes) the creation.
	Action undo.Action
	// Unchanged is set when an imported note was saved exactly as it was,
	// because its properties could not be parsed: it keeps its original uid.
	Unchanged bool
}

// Slug returns the first few words of text for a file name: lowercase,
// without characters that are unsafe in a file name or a link, spaces
// collapsed, at most slugMaxRunes long. It may be empty.
func Slug(text string) string {
	var words []string
	for _, w := range strings.Fields(text) {
		if w = cleanWord(w); w != "" {
			words = append(words, w)
		}
		if len(words) == slugWords {
			break
		}
	}
	slug := strings.Trim(strings.Join(words, " "), ". ")
	if r := []rune(slug); len(r) > slugMaxRunes {
		slug = strings.TrimRight(string(r[:slugMaxRunes]), ". ")
	}
	return slug
}

func cleanWord(w string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(slugDropChars, r) {
			return -1
		}
		return unicode.ToLower(r)
	}, w)
}

// Name returns the file name of an intake note made at now from text:
// "YYYY-MM-DD-HHMMSS slug.md", or without the slug when there is none.
func Name(now time.Time, text string) string {
	name := now.Format(stampLayout)
	if slug := Slug(text); slug != "" {
		name += " " + slug
	}
	return name + noteExt
}

// Capture creates an inbox note holding text as its body, with the given kind
// and source.
func Capture(s store.Store, now time.Time, kind store.Kind, source, text string) (Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Result{}, ErrEmpty
	}
	name := Name(now, text)
	data, err := store.IntakeNote(name, kind, source, now.Format(dayLayout), text+"\n")
	if err != nil {
		return Result{}, err
	}
	return createUnique(s, name, func(final string) ([]byte, error) {
		return rename(data, name, final)
	})
}

// ImportTranscript reads the file at path and creates an inbox note whose
// body is its text, unmodified, with source transcript and kind rumor (raw
// and unconfirmed until reviewed).
func ImportTranscript(s store.Store, now time.Time, path string) (Result, error) {
	text, err := readSource(path)
	if err != nil {
		return Result{}, err
	}
	base := filepath.Base(expandHome(path))
	name := Name(now, strings.TrimSuffix(base, filepath.Ext(base)))
	data, err := store.IntakeNote(name, store.KindRumor, SourceTranscript, now.Format(dayLayout), string(text))
	if err != nil {
		return Result{}, err
	}
	return createUnique(s, name, func(final string) ([]byte, error) {
		return rename(data, name, final)
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

// rename sets the title of the note data to final's name when a collision
// gave it a different name than the one it was made for.
func rename(data []byte, made, final string) ([]byte, error) {
	if made == final {
		return data, nil
	}
	return store.SetTitle(data, strings.TrimSuffix(final, noteExt))
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
