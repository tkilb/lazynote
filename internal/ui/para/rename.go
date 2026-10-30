package para

import (
	"fmt"
	"strings"

	"github.com/tkilb/lazynote/internal/store"
)

// RenameRequest is a validated rename of the item at Rel to Name, the new
// file or folder name (a note's name includes .md).
type RenameRequest struct {
	Rel   string
	IsDir bool
	// Typed is what the user typed; Name is it cleaned.
	Typed, Name string
}

// askRename opens the rename prompt for the hovered item, starting with its
// current name (a note's without .md).
func (m Model) askRename() (Model, Effect) {
	e, ok := m.Selected()
	if !ok {
		return m, Effect{}
	}
	name := e.Name
	if !e.IsDir {
		name = strings.TrimSuffix(strings.TrimSuffix(name, ".md"), ".MD")
	}
	return m, Effect{Prompt: &Prompt{Title: "Rename", Label: "New name:", Initial: name, Op: OpRename}}
}

// rename validates the typed name and hands the rename to the root model. A
// name that is the same as the current one does nothing.
func (m Model) rename(text string) (Model, Effect) {
	e, ok := m.Selected()
	if !ok {
		return m, Effect{}
	}
	name, err := store.CleanRenamedName(text, e.IsDir)
	if err != nil {
		return m, Effect{Err: fmt.Errorf("rename: %w", err)}
	}
	if name == e.Name {
		return m, Effect{Info: "The name is unchanged."}
	}
	return m, Effect{Rename: &RenameRequest{Rel: e.Rel, IsDir: e.IsDir, Typed: text, Name: name}}
}
