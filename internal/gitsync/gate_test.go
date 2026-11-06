package gitsync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeGate returns a gate on a clock the test moves.
func fakeGate(interval time.Duration) (*Gate, *time.Time) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	g := NewGate(interval)
	g.now = func() time.Time { return now }
	return g, &now
}

func TestGateRunsTheFirstRequestAtOnce(t *testing.T) {
	g, _ := fakeGate(time.Minute)
	run, wait := g.Request()
	assert.True(t, run)
	assert.Zero(t, wait)
}

func TestGateHoldsAnEarlyRequestUntilTheIntervalEnds(t *testing.T) {
	g, now := fakeGate(time.Minute)
	g.Request()

	*now = now.Add(20 * time.Second)
	run, wait := g.Request()
	assert.False(t, run)
	assert.Equal(t, 40*time.Second, wait)

	*now = now.Add(5 * time.Second)
	run, wait = g.Request()
	assert.False(t, run, "a second early request shares the held one")
	assert.Zero(t, wait)

	*now = now.Add(35 * time.Second)
	assert.True(t, g.Due())
	assert.False(t, g.Due(), "a held request is answered once")
}

func TestGateRunsAgainOnceTheIntervalHasPassed(t *testing.T) {
	g, now := fakeGate(time.Minute)
	g.Request()
	*now = now.Add(time.Minute)
	run, _ := g.Request()
	assert.True(t, run)
	run, wait := g.Request()
	assert.False(t, run, "the interval starts again")
	assert.Equal(t, time.Minute, wait)
}

func TestGateMarkReplacesAHeldRequest(t *testing.T) {
	g, now := fakeGate(time.Minute)
	g.Request()
	*now = now.Add(10 * time.Second)
	g.Request() // held

	*now = now.Add(10 * time.Second)
	g.Mark() // a direct read at 20s
	assert.False(t, g.Due(), "the direct read took the held request's place")

	*now = now.Add(30 * time.Second)
	run, wait := g.Request()
	assert.False(t, run, "the interval counts from the direct read")
	assert.Equal(t, 30*time.Second, wait)
}

func TestGateDueWithoutARequestDoesNothing(t *testing.T) {
	g, _ := fakeGate(time.Minute)
	assert.False(t, g.Due())
}
