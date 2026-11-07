// Package templates finds, reads and fills the note templates in
// ~/.config/para/templates. A notebook uses the subfolder named after the
// notebook's folder (notebook ~/Notes/work uses templates/work), and the .md
// files directly in templates/ are shared by every notebook; a notebook's own
// template overrides a shared one of the same name. The daily template is
// special: it is never offered, only looked up by DailyPath.
package templates

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tkilb/para/internal/store"
)

const (
	ext = ".md"

	// DailyName is the template that makes daily notes. It is excluded from
	// List and Resolve.
	DailyName = "daily"

	// maxBytes bounds a template file, as a note is bounded.
	maxBytes = 1 << 20

	tagsKey   = "tags"
	sourceKey = "source"
)

var (
	// ErrNotFound reports a template name that the notebook does not offer.
	ErrNotFound = errors.New("template not found")
	// ErrEmptyBody reports a template with nothing but properties.
	ErrEmptyBody = errors.New("template has no body")
	// ErrTooLarge reports a template file over the size limit.
	ErrTooLarge = errors.New("template is too large")
	// ErrNotRegular reports a template that is not a regular file.
	ErrNotRegular = errors.New("template is not a regular file")
)

// Template is a template file split into the values it pre-fills and its
// body.
type Template struct {
	Name string
	// Kind, Source and Tags come from the properties; they are zero when the
	// template does not set them.
	Kind   store.Kind
	Source string
	Tags   []string
	// Body is the text after the properties, with leading blank lines and
	// trailing whitespace removed.
	Body string
}

// notebookDir is the subfolder of dir for notebook, or "" when the name
// cannot be a folder name.
func notebookDir(dir, notebook string) string {
	switch notebook {
	case "", ".", "..", string(filepath.Separator):
		return ""
	}
	if strings.ContainsRune(notebook, filepath.Separator) {
		return ""
	}
	return filepath.Join(dir, notebook)
}

// scan adds the regular .md files directly in folder to found, replacing
// same-named entries. A missing folder adds nothing.
func scan(folder string, found map[string]string) error {
	entries, err := os.ReadDir(folder)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read templates folder: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		stem, isMD := strings.CutSuffix(name, ext)
		if !isMD || stem == "" || strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(folder, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			found[stem] = path
		}
	}
	return nil
}

// index maps template names to paths for notebook: shared first, then the
// notebook's own over them.
func index(dir, notebook string) (map[string]string, error) {
	found := map[string]string{}
	if err := scan(dir, found); err != nil {
		return nil, err
	}
	if nb := notebookDir(dir, notebook); nb != "" {
		if err := scan(nb, found); err != nil {
			return nil, err
		}
	}
	delete(found, DailyName)
	return found, nil
}

// List returns the sorted names of the templates notebook offers.
func List(dir, notebook string) ([]string, error) {
	found, err := index(dir, notebook)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(found))
	for n := range found {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// Resolve returns the path of the named template for notebook, or
// ErrNotFound.
func Resolve(dir, notebook, name string) (string, error) {
	found, err := index(dir, notebook)
	if err != nil {
		return "", err
	}
	path, ok := found[name]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return path, nil
}

// DailyPath returns the daily template for notebook: templates/<notebook>/
// daily.md, else templates/daily.md, else "" for none.
func DailyPath(dir, notebook string) string {
	candidates := []string{filepath.Join(dir, DailyName+ext)}
	if nb := notebookDir(dir, notebook); nb != "" {
		candidates = append([]string{filepath.Join(nb, DailyName+ext)}, candidates...)
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() {
			return p
		}
	}
	return ""
}

// Load reads and parses the template at path. A template without a body is
// ErrEmptyBody.
func Load(name, path string) (Template, error) {
	data, err := readLimited(path)
	if err != nil {
		return Template{}, err
	}
	return Parse(name, data)
}

// readLimited reads a regular file of at most maxBytes.
func readLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotRegular
	}
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

// Parse splits data into the values the properties pre-fill and the body.
// Properties that cannot be parsed are an error, so a broken template is
// reported instead of half used.
func Parse(name string, data []byte) (Template, error) {
	n := store.ParseNote(name+ext, data)
	if n.FrontmatterErr != nil {
		return Template{}, fmt.Errorf("template properties cannot be read: %w", n.FrontmatterErr)
	}
	body := strings.TrimRight(strings.TrimLeft(n.Body, "\r\n"), " \t\r\n")
	if strings.TrimSpace(body) == "" {
		return Template{}, ErrEmptyBody
	}
	return Template{
		Name: name, Kind: n.Kind, Body: body,
		Source: scalar(n.Frontmatter, sourceKey), Tags: list(n.Frontmatter, tagsKey),
	}, nil
}

// scalar is the text of property key, or "".
func scalar(fm *store.Frontmatter, key string) string {
	if v := fm.Get(key); v != nil {
		return strings.TrimSpace(v.Value)
	}
	return ""
}

// list is the items of property key, whether written as a list or as one
// value, without blanks.
func list(fm *store.Frontmatter, key string) []string {
	v := fm.Get(key)
	if v == nil {
		return nil
	}
	var out []string
	items := v.Content
	if len(items) == 0 {
		items = append(items, v)
	}
	for _, it := range items {
		if s := strings.TrimSpace(it.Value); s != "" {
			out = append(out, s)
		}
	}
	return out
}
