// Command lazynote is a TUI note browser.
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/tkilb/lazynote/internal/cli"
	"github.com/tkilb/lazynote/internal/config"
	"github.com/tkilb/lazynote/internal/store"
	"github.com/tkilb/lazynote/internal/ui/popup"
	"github.com/tkilb/lazynote/internal/version"
)

// startup is what the config gives the interface and the shell commands.
type startup struct {
	store         store.Store
	root          string
	sources       []string
	retentionDays int
}

// openStore loads the config and opens the first available notebook. The
// Notebook panel (a later chunk) will replace this with a real selection.
func openStore() (startup, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return startup{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return startup{}, err
	}
	for _, nb := range cfg.Notebooks {
		if nb.Available {
			return startup{
				store: store.NewMarkdown(nb.Path), root: nb.Path,
				sources: cfg.Sources, retentionDays: cfg.Retention.DailyDays,
			}, nil
		}
	}
	return startup{}, errors.New("none of the configured notebooks is an existing directory")
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
	open := func() (store.Store, []string, error) {
		st, err := openStore()
		return st.store, st.sources, err
	}
	if handled, code := cli.Run(os.Args[1:], cli.SystemEnv(open, clipboard.ReadAll)); handled {
		os.Exit(code)
	}

	boot, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lazynote: %v\n", err)
		os.Exit(1)
	}
	sess, sessErr := openSession(boot.root)
	if sess.db != nil {
		defer sess.db.Close()
	}
	start := initialModel(boot.store)
	start.sources = boot.sources
	start.para = start.para.WithRetention(boot.retentionDays, time.Now)
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
