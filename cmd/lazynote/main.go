// Command lazynote is a TUI note browser.
package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/config"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/version"
)

// openStore loads the config and opens the first available notebook. The
// Notebook panel (a later chunk) will replace this with a real selection.
func openStore() (store.Store, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	for _, nb := range cfg.Notebooks {
		if nb.Available {
			return store.NewMarkdown(nb.Path), nil
		}
	}
	return nil, errors.New("none of the configured notebooks is an existing directory")
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

	st, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazynote: %v\n", err)
		os.Exit(1)
	}

	// The alternate screen gives the program the full terminal buffer, so
	// resuming after ctrl+z or an external editor repaints cleanly. Focus
	// reporting lets the app refresh when the terminal regains focus (not
	// every terminal sends it; ctrl+l always works).
	p := tea.NewProgram(initialModel(st), tea.WithAltScreen(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running lazynote: %v\n", err)
		os.Exit(1)
	}
}
