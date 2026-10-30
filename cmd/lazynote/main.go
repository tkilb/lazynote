// Command lazynote is a TUI note browser.
package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/config"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/popup"
	"github.com/tkilb/lazynote/internal/version"
)

// openStore loads the config and opens the first available notebook,
// returning it, its root path and the config's Quick Add sources. The
// Notebook panel (a later chunk) will replace this with a real selection.
func openStore() (store.Store, string, []string, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, "", nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", nil, err
	}
	for _, nb := range cfg.Notebooks {
		if nb.Available {
			return store.NewMarkdown(nb.Path), nb.Path, cfg.Sources, nil
		}
	}
	return nil, "", nil, errors.New("none of the configured notebooks is an existing directory")
}

// isVersionArg reports whether args (os.Args[1:]) ask for the version
// rather than the TUI.
func isVersionArg(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "version", "--version", "-v":
		return true
	}
	return false
}

func main() {
	if isVersionArg(os.Args[1:]) {
		fmt.Printf("lazynote %s\n", version.Version)
		return
	}

	st, root, sources, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazynote: %v\n", err)
		os.Exit(1)
	}
	sess, sessErr := openSession(root)
	if sess.db != nil {
		defer sess.db.Close()
	}
	start := initialModel(st)
	start.sources = sources
	if sessErr != nil {
		start = start.notify(popup.Warning, fmt.Sprintf("Your position will not be saved: %s.", sessErr))
	}
	start = start.withSession(sess).refreshLinks()

	// The alternate screen gives the program the full terminal buffer, so
	// resuming after ctrl+z or an external editor repaints cleanly. Focus
	// reporting lets the app refresh when the terminal regains focus (not
	// every terminal sends it; ctrl+l always works).
	p := tea.NewProgram(start, tea.WithAltScreen(), tea.WithReportFocus())
	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running lazynote: %v\n", err)
		os.Exit(1)
	}
	if m, ok := final.(model); ok && m.saveErr != nil {
		fmt.Fprintf(os.Stderr, "lazynote: could not save your position: %v\n", m.saveErr)
	}
}
