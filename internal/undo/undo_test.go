package undo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recorder builds actions that append to a shared log.
type recorder struct{ log []string }

func (r *recorder) action(name string) Action {
	return Action{
		Description: name,
		Undo:        func() error { r.log = append(r.log, "undo "+name); return nil },
		Redo:        func() error { r.log = append(r.log, "redo "+name); return nil },
	}
}

func failing(name string, err error, r *recorder, failUndo, failRedo bool) Action {
	a := r.action(name)
	undo, redo := a.Undo, a.Redo
	a.Undo = func() error {
		if failUndo {
			return err
		}
		return undo()
	}
	a.Redo = func() error {
		if failRedo {
			return err
		}
		return redo()
	}
	return a
}

func TestStackEmpty(t *testing.T) {
	var s Stack
	assert.False(t, s.CanUndo())
	assert.False(t, s.CanRedo())

	_, err := s.Undo()
	assert.ErrorIs(t, err, ErrNothingToUndo)
	_, err = s.Redo()
	assert.ErrorIs(t, err, ErrNothingToRedo)
}

func TestStackUndoRedoOrder(t *testing.T) {
	var r recorder
	var s Stack
	s.Push(r.action("a"))
	s.Push(r.action("b"))
	s.Push(r.action("c"))

	for _, want := range []string{"c", "b", "a"} {
		got, err := s.Undo()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	assert.False(t, s.CanUndo())
	assert.True(t, s.CanRedo())

	for _, want := range []string{"a", "b", "c"} {
		got, err := s.Redo()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	assert.False(t, s.CanRedo())
	assert.Equal(t, []string{"undo c", "undo b", "undo a", "redo a", "redo b", "redo c"}, r.log)
}

func TestStackPushClearsRedo(t *testing.T) {
	var r recorder
	var s Stack
	s.Push(r.action("a"))
	s.Push(r.action("b"))
	_, err := s.Undo()
	require.NoError(t, err)
	require.True(t, s.CanRedo())

	s.Push(r.action("c"))
	assert.False(t, s.CanRedo(), "a new action discards the redo history")

	got, err := s.Undo()
	require.NoError(t, err)
	assert.Equal(t, "c", got)
	got, err = s.Undo()
	require.NoError(t, err)
	assert.Equal(t, "a", got, "b was discarded")
}

func TestStackFailedUndoKeepsPosition(t *testing.T) {
	var r recorder
	var s Stack
	boom := errors.New("changed")
	fails := true
	a := r.action("a")
	a.Undo = func() error {
		if fails {
			return boom
		}
		return nil
	}
	s.Push(a)

	desc, err := s.Undo()
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, "a", desc)
	assert.True(t, s.CanUndo(), "a refused undo stays on the stack")
	assert.False(t, s.CanRedo())

	fails = false
	_, err = s.Undo()
	require.NoError(t, err)
	assert.False(t, s.CanUndo())
}

func TestStackFailedRedoKeepsPosition(t *testing.T) {
	var r recorder
	var s Stack
	boom := errors.New("exists")
	s.Push(failing("a", boom, &r, false, true))
	_, err := s.Undo()
	require.NoError(t, err)

	_, err = s.Redo()
	assert.ErrorIs(t, err, boom)
	assert.True(t, s.CanRedo())
}

func TestGroupUndoReverseRedoForward(t *testing.T) {
	var r recorder
	g := Group("paste 3 items", r.action("a"), r.action("b"), r.action("c"))
	assert.Equal(t, "paste 3 items", g.Description)

	require.NoError(t, g.Undo())
	require.NoError(t, g.Redo())
	assert.Equal(t, []string{"undo c", "undo b", "undo a", "redo a", "redo b", "redo c"}, r.log)
}

func TestGroupUndoFailureRollsBack(t *testing.T) {
	var r recorder
	boom := errors.New("changed")
	g := Group("g", r.action("a"), failing("b", boom, &r, true, false), r.action("c"))

	err := g.Undo()
	assert.ErrorIs(t, err, boom)
	// c was undone before b failed, so it is redone to restore the state.
	assert.Equal(t, []string{"undo c", "redo c"}, r.log)
}

func TestGroupRedoFailureRollsBack(t *testing.T) {
	var r recorder
	boom := errors.New("exists")
	g := Group("g", r.action("a"), r.action("b"), failing("c", boom, &r, false, true))

	err := g.Redo()
	assert.ErrorIs(t, err, boom)
	// a and b were redone before c failed, so they are undone, last first.
	assert.Equal(t, []string{"redo a", "redo b", "undo b", "undo a"}, r.log)
}

func TestGroupRollbackFailureIsReported(t *testing.T) {
	var r recorder
	first, second := errors.New("first"), errors.New("second")
	stuck := r.action("c")
	stuck.Redo = func() error { return second }
	g := Group("g", failing("b", first, &r, true, false), stuck)

	err := g.Undo()
	assert.ErrorIs(t, err, first)
	assert.ErrorIs(t, err, second, "a failed rollback is reported with the cause")
}

func TestGroupEmpty(t *testing.T) {
	g := Group("nothing")
	assert.NoError(t, g.Undo())
	assert.NoError(t, g.Redo())
}
