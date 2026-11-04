package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tkilb/lazynote/internal/clip"
)

// ResolveFile turns what the user typed into the path to read. A bare name,
// with no folder part and not starting with "~", is looked for in inboxDir
// first: the exact file name, else the one file whose name without its
// extension matches ignoring case, so "standup" finds "standup.md". Two
// files with the same stem are not guessed between. Anything else, or a bare
// name not in the inbox, comes back as typed (trimmed) for ReadFile to read.
func ResolveFile(input, inboxDir string) string {
	name := strings.TrimSpace(input)
	if name == "" || inboxDir == "" || !isBare(name) {
		return name
	}
	if info, err := os.Stat(filepath.Join(inboxDir, name)); err == nil && info.Mode().IsRegular() {
		return filepath.Join(inboxDir, name)
	}
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		return name
	}
	var found []string
	for _, e := range entries {
		stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if !strings.EqualFold(stem, name) {
			continue
		}
		if info, err := os.Stat(filepath.Join(inboxDir, e.Name())); err == nil && info.Mode().IsRegular() {
			found = append(found, e.Name())
		}
	}
	if len(found) != 1 {
		return name
	}
	return filepath.Join(inboxDir, found[0])
}

// isBare reports a name with no folder part that is not a home path.
func isBare(name string) bool {
	return !strings.ContainsRune(name, filepath.Separator) && !strings.ContainsRune(name, '/') && !strings.HasPrefix(name, "~")
}

// Complete completes the path the user is typing, as Tab does in a shell. It
// returns the text with the start shared by every match filled in (a single
// folder gets a trailing "/"), and the matching names, folders with a "/",
// sorted ignoring case. A bare name, or nothing, completes against inboxDir;
// a path completes in the folder it names, with "~" expanded only to look and
// kept as typed in the result. Hidden names match only when the part typed
// starts with a dot. Nothing is read but folder listings.
func Complete(input, inboxDir string) (string, []string) {
	var dir, typedDir, prefix string
	switch {
	case input == "~":
		return completeIn(expandHome("~"), "~/", "")
	case isBare(input):
		return completeIn(inboxDir, "", input)
	}
	i := strings.LastIndex(input, "/")
	dir, typedDir, prefix = expandHome(input[:i+1]), input[:i+1], input[i+1:]
	if i < 0 {
		dir, typedDir, prefix = ".", "", input
	}
	return completeIn(dir, typedDir, prefix)
}

// completeIn lists dir for names starting with prefix and builds the result
// for the typed folder part.
func completeIn(dir, typedDir, prefix string) (string, []string) {
	if dir == "" {
		return typedDir + prefix, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return typedDir + prefix, nil
	}
	showHidden := strings.HasPrefix(prefix, ".")
	var names []string
	for _, e := range entries {
		name := e.Name()
		if (!showHidden && strings.HasPrefix(name, ".")) || !hasPrefixFold(name, prefix) {
			continue
		}
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return typedDir + prefix, nil
	}
	sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
	if len(names) == 1 {
		return typedDir + names[0], names
	}
	return typedDir + commonPrefix(names, len([]rune(prefix))), names
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// commonPrefix is the start shared by all names ignoring case, in the case of
// the first name, and never shorter than the minRunes the user already typed.
func commonPrefix(names []string, minRunes int) string {
	first := []rune(names[0])
	n := len(first)
	for _, name := range names[1:] {
		other := []rune(name)
		k := 0
		for k < n && k < len(other) && strings.EqualFold(string(first[k]), string(other[k])) {
			k++
		}
		n = k
	}
	return string(first[:max(n, min(minRunes, len(first)))])
}

// Describe is the one-line preview of what an import of input would read:
// the clipboard (with its word count), a web page (its host) or a file (its
// size, or why it cannot be read). It reads the clipboard, and looks at a file
// only with a stat. It never shows any content, path or address beyond the
// host.
func Describe(input, inboxDir string, readClipboard func() (string, error)) string {
	in := Classify(input)
	switch in.Kind {
	case Clipboard:
		return describeClipboard(readClipboard)
	case Web:
		u, err := clip.ParseURL(in.Value)
		if err != nil {
			return "not a valid web address"
		}
		return "web page, " + u.Hostname()
	}
	return describeFile(ResolveFile(in.Value, inboxDir))
}

func describeClipboard(read func() (string, error)) string {
	text, err := read()
	switch {
	case err != nil:
		return "cannot read the clipboard"
	case len(text) > MaxBytes:
		return "the clipboard is too large (the limit is 10 MB)"
	}
	switch n := len(strings.Fields(text)); n {
	case 0:
		return "the clipboard is empty"
	case 1:
		return "clipboard, 1 word"
	default:
		return fmt.Sprintf("clipboard, %d words", n)
	}
}

func describeFile(path string) string {
	info, err := os.Stat(expandHome(path))
	switch {
	case err != nil:
		return "file not found"
	case info.IsDir():
		return "a folder, not a file"
	case !info.Mode().IsRegular():
		return "not a regular file"
	case info.Size() > MaxBytes:
		return "file is too large (the limit is 10 MB)"
	}
	return "file, " + humanSize(info.Size())
}

func humanSize(n int64) string {
	const kb, mb = 1 << 10, 1 << 20
	switch {
	case n == 1:
		return "1 byte"
	case n < kb:
		return fmt.Sprintf("%d bytes", n)
	case n < mb:
		return fmt.Sprintf("%d KB", max((n+kb/2)/kb, 1))
	}
	return fmt.Sprintf("%.1f MB", float64(n)/mb)
}
