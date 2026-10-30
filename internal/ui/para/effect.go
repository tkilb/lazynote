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
}

// ConfirmOp names what happens when a Confirm is accepted.
type ConfirmOp int

const (
	// OpOverwritePaste pastes over items that already exist.
	OpOverwritePaste ConfirmOp = iota + 1
)

// Confirm is a yes/no question. The root model shows Prompt and, on yes,
// calls Model.ApplyConfirm with Op.
type Confirm struct {
	Prompt string
	Op     ConfirmOp
}
