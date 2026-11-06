// Package cli implements para's shell commands, which create inbox notes
// without opening the interface:
//
//	para add [TITLE] [BODY] [--kind K] [--source S] [--tags a,b] [--clipboard]
//
// Everything is optional but the title, which may also come from the first
// line of piped or clipboard text. The commands print the new note's path on
// standard output and report problems on standard error, never echoing a file
// path.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tkilb/para/internal/config"
	"github.com/tkilb/para/internal/intake"
	"github.com/tkilb/para/internal/store"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// Env is everything the commands need from the outside, so tests can supply
// their own.
type Env struct {
	// Stdin is read only when it is piped (StdinTTY false) and no body was
	// given.
	Stdin    io.Reader
	StdinTTY bool
	Stdout   io.Writer
	Stderr   io.Writer
	// Now stamps the note's date.
	Now time.Time
	// Open opens the notebook and returns it with the configured sources, the
	// first of which is the default source.
	Open func() (store.Store, []string, error)
	// Clipboard returns the clipboard's text.
	Clipboard func() (string, error)
}

const addUsageFormat = `Usage: para add [TITLE] [BODY] [flags]

Adds a note to the inbox and prints its path. TITLE names the note. BODY, or
text piped in, or the clipboard, is its body; with no TITLE the first line of
that text is the title. Put -- before a title that starts with a dash.

Flags:
  --kind K       %s or unset (default unset)
  --source S     any word, such as meeting (default: the first configured source)
  --tags a,b     tag names separated by commas or spaces
  --clipboard    take the text from the clipboard
`

// addUsage is the usage text of add, naming the kinds that are configured.
func addUsage() string {
	return fmt.Sprintf(addUsageFormat, kindNames())
}

// kindNames lists the configured kinds for a message, such as "fact, rumor".
func kindNames() string {
	kinds := store.Kinds()
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.String()
	}
	return strings.Join(names, ", ")
}

// Run runs the command named by args[0], reporting whether it was one of
// ours. Anything else is left to the caller.
func Run(args []string, env Env) (handled bool, code int) {
	if len(args) == 0 {
		return false, exitOK
	}
	switch args[0] {
	case "add":
		return true, runAdd(args[1:], env)
	}
	return false, exitOK
}

// parse reads flags and positional arguments in any order into fs and
// returns the positionals. Everything after a bare "--" is positional.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var tail []string
	if i := indexOf(args, "--"); i >= 0 {
		args, tail = args[:i], args[i+1:]
	}
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
	return append(pos, tail...), nil
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// newFlagSet makes a flag set that prints the given usage instead of the
// flag package's own.
func newFlagSet(name, usage string, env Env) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(env.Stderr, usage) }
	return fs
}

// flagError prints a flag-parsing error and returns the exit code: help is
// not an error.
func flagError(err error, usage string, env Env) int {
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(env.Stderr, usage)
		return exitOK
	}
	fmt.Fprintf(env.Stderr, "para: %v\n\n%s", err, usage)
	return exitUsage
}

func fail(env Env, format string, args ...any) int {
	fmt.Fprintf(env.Stderr, "para: "+format+"\n", args...)
	return exitError
}

func runAdd(args []string, env Env) int {
	fs := newFlagSet("add", addUsage(), env)
	kindFlag := fs.String("kind", "", "")
	sourceFlag := fs.String("source", "", "")
	tagsFlag := fs.String("tags", "", "")
	clipboard := fs.Bool("clipboard", false, "")
	pos, err := parse(fs, args)
	if err != nil {
		return flagError(err, addUsage(), env)
	}
	if len(pos) > 2 {
		return fail(env, "too many arguments: give a title and at most one body")
	}

	kind, err := parseKind(*kindFlag)
	if err != nil {
		return fail(env, "%v", err)
	}
	source := ""
	if flagSet(fs, "source") {
		if source, err = config.ValidateSource(*sourceFlag); err != nil {
			return fail(env, "--source %v", err)
		}
	}
	tags, err := intake.ParseTags(*tagsFlag)
	if err != nil {
		return fail(env, "--tags: %v", err)
	}

	title, body, code := addText(pos, *clipboard, env)
	if code != exitOK {
		return code
	}

	st, sources, err := env.Open()
	if err != nil {
		return fail(env, "%v", err)
	}
	if source == "" {
		source = intake.SourceAdhoc
		if len(sources) > 0 {
			source = sources[0]
		}
	}
	res, err := intake.Capture(st, env.Now, kind, source, title, body, tags)
	if err != nil {
		return fail(env, "could not add the note: %v", err)
	}
	fmt.Fprintln(env.Stdout, filepath.ToSlash(res.Rel))
	return exitOK
}

func flagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

// parseKind reads --kind: empty and "unset" mean no kind, anything else must
// be one of the six kinds, in any case.
func parseKind(s string) (store.Kind, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == string(store.KindUnset) {
		return store.KindUnset, nil
	}
	if k := store.ParseKind(s); k != store.KindUnset {
		return k, nil
	}
	return store.KindUnset, fmt.Errorf("unknown kind %q: use one of %s or unset", s, kindNames())
}

// addText works out the title and body of the note from the positional
// arguments, and from the clipboard or piped text when no body was given.
func addText(pos []string, clipboard bool, env Env) (title, body string, code int) {
	if len(pos) == 2 {
		return pos[0], pos[1], exitOK
	}
	text, haveText, err := externalText(clipboard, env)
	if err != nil {
		return "", "", fail(env, "%v", err)
	}
	switch {
	case len(pos) == 1:
		return pos[0], text, exitOK
	case !haveText:
		return "", "", fail(env, "a title is needed: give one, pipe text in, or use --clipboard")
	}
	title, body = intake.SplitText(text)
	if title == "" {
		return "", "", fail(env, "a title is needed: the text has nothing in it")
	}
	return title, body, exitOK
}

// externalText returns the clipboard's text when asked, else piped text when
// stdin is not a terminal. haveText says whether either source applies.
func externalText(clipboard bool, env Env) (text string, haveText bool, err error) {
	if clipboard {
		text, err = env.Clipboard()
		if err != nil {
			return "", false, fmt.Errorf("could not read the clipboard: %v (it needs pbpaste on macOS, or xclip, xsel or wl-paste on Linux)", err)
		}
		if strings.TrimSpace(text) == "" {
			return "", false, errors.New("the clipboard is empty")
		}
		return text, true, nil
	}
	if env.StdinTTY || env.Stdin == nil {
		return "", false, nil
	}
	data, err := io.ReadAll(io.LimitReader(env.Stdin, store.MaxNoteBytes+1))
	switch {
	case err != nil:
		return "", false, errors.New("could not read the piped text")
	case len(data) > store.MaxNoteBytes:
		return "", false, errors.New("the piped text is too large")
	}
	return string(data), true, nil
}

// SystemEnv is the Env for the real terminal: it reads os.Stdin when piped
// and the system clipboard.
func SystemEnv(open func() (store.Store, []string, error), clipboard func() (string, error)) Env {
	tty := true
	if info, err := os.Stdin.Stat(); err == nil {
		tty = info.Mode()&os.ModeCharDevice != 0
	}
	return Env{
		Stdin: os.Stdin, StdinTTY: tty, Stdout: os.Stdout, Stderr: os.Stderr,
		Now: time.Now(), Open: open, Clipboard: clipboard,
	}
}
