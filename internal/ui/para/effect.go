package para

import "github.com/tkilb/lazynote/internal/undo"

// Effect is what a key press asks the root model to do besides updating the
// panel: record an undo action, show a message, or ask for confirmation.
type Effect struct {
	// Action, when set, is pushed onto the undo history.
	Action *undo.Action
	// Info is a result message; Err, when set, is shown as an error instead.
	Info string
	Err  error
	// Confirm, when set, asks the user before anything changes.
	Confirm *Confirm
	// Prompt, when set, asks the user to type a line of text.
	Prompt *Prompt
}

// ConfirmOp names what happens when a Confirm is accepted.
type ConfirmOp int

const (
	// OpOverwritePaste pastes over items that already exist.
	OpOverwritePaste ConfirmOp = iota + 1
	// OpTrash moves the targets to the trash.
	OpTrash
	// OpDelete permanently deletes the targets.
	OpDelete
)

// Confirm is a yes/no question. The root model shows Prompt and, on yes,
// calls Model.ApplyConfirm with Op.
type Confirm struct {
	Prompt string
	Op     ConfirmOp
	// Danger marks an action that cannot be undone.
	Danger bool
}

// InputOp names what happens with the text a Prompt collects.
type InputOp int

const (
	// OpCreate creates a note, or a folder with a trailing slash.
	OpCreate InputOp = iota + 1
)

// Prompt asks for one line of text. The root model shows Label with an
// input field and, on enter, calls Model.ApplyInput with Op and the text.
type Prompt struct {
	Label string
	Op    InputOp
}
