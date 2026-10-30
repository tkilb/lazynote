// Package undo provides a small in-memory, multi-level undo/redo stack for
// reversible note changes. Ported from lazytask. It knows nothing about
// notes: callers supply Undo and Redo closures that do the actual reversal,
// so this package only tracks ordering, history truncation and failure.
package undo

import (
	"errors"
	"fmt"
)

// ErrNothingToUndo and ErrNothingToRedo report an empty side of the stack.
var (
	ErrNothingToUndo = errors.New("nothing to undo")
	ErrNothingToRedo = errors.New("nothing to redo")
)

// Action is one undoable, redoable user action. Undo reverses what the
// action did and Redo applies it again. Both may perform I/O and fail.
type Action struct {
	// Description is a short summary shown in feedback, e.g. "move a.md".
	Description string
	Undo        func() error
	Redo        func() error
}

// Group combines actions into one entry, so a multi-item operation undoes
// and redoes as a unit. Undo runs the members in reverse order and Redo in
// forward order. When a member fails, the members already run are reversed
// again, so a failed Undo or Redo leaves things as they were.
func Group(description string, actions ...Action) Action {
	return Action{
		Description: description,
		Undo: func() error {
			for i := len(actions) - 1; i >= 0; i-- {
				if err := actions[i].Undo(); err != nil {
					// Members after i were undone: redo them, front to back.
					var steps []func() error
					for _, a := range actions[i+1:] {
						steps = append(steps, a.Redo)
					}
					return rollback(err, steps)
				}
			}
			return nil
		},
		Redo: func() error {
			for i, a := range actions {
				if err := a.Redo(); err != nil {
					// Members before i were redone: undo them, back to front.
					var steps []func() error
					for j := i - 1; j >= 0; j-- {
						steps = append(steps, actions[j].Undo)
					}
					return rollback(err, steps)
				}
			}
			return nil
		},
	}
}

// rollback runs steps after cause and returns cause joined with any step
// that fails too.
func rollback(cause error, steps []func() error) error {
	errs := []error{cause}
	for _, step := range steps {
		if err := step(); err != nil {
			errs = append(errs, fmt.Errorf("rollback: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Stack is a multi-level undo/redo history. The zero value is an empty
// stack ready to use.
//
// Entries live in one slice; pos separates undoable history (entries[:pos])
// from the redoable future (entries[pos:]). Pushing a new action discards
// the future, because it no longer applies.
type Stack struct {
	entries []Action
	pos     int
}

// Push records a newly performed action as the next Undo candidate and
// discards any redo history.
func (s *Stack) Push(a Action) {
	s.entries = append(s.entries[:s.pos], a)
	s.pos++
}

// CanUndo reports whether there is an action to undo.
func (s *Stack) CanUndo() bool { return s.pos > 0 }

// CanRedo reports whether there is an undone action to redo.
func (s *Stack) CanRedo() bool { return s.pos < len(s.entries) }

// Undo runs the most recent action's Undo and returns its description. The
// stack only steps back when Undo succeeds, so a refused undo can be
// retried after the cause is fixed.
func (s *Stack) Undo() (string, error) {
	if !s.CanUndo() {
		return "", ErrNothingToUndo
	}
	a := s.entries[s.pos-1]
	if err := a.Undo(); err != nil {
		return a.Description, err
	}
	s.pos--
	return a.Description, nil
}

// Redo runs the next undone action's Redo and returns its description. The
// stack only steps forward when Redo succeeds.
func (s *Stack) Redo() (string, error) {
	if !s.CanRedo() {
		return "", ErrNothingToRedo
	}
	a := s.entries[s.pos]
	if err := a.Redo(); err != nil {
		return a.Description, err
	}
	s.pos++
	return a.Description, nil
}
