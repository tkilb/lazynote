package gitsync

import (
	"sync"
	"time"
)

// Gate spaces out reads of the git state: after a read, another is allowed
// only once the interval has passed. A request that arrives too soon is not
// lost; it is held until the interval ends, and all such requests share one
// read, so a burst of changes costs one read per interval at most.
type Gate struct {
	mu       sync.Mutex
	interval time.Duration
	now      func() time.Time
	last     time.Time
	pending  bool
}

// NewGate returns a gate that allows a read every interval.
func NewGate(interval time.Duration) *Gate {
	return &Gate{interval: interval, now: time.Now}
}

// Mark records a read made outside the gate (one that was asked for directly)
// and cancels any held request, which that read has made redundant.
func (g *Gate) Mark() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.last, g.pending = g.now(), false
}

// Request asks for a read. run is true when the caller should read now. When
// it is false and wait is positive, the caller should call Due after wait; when
// both are zero a request is already held and nothing is needed.
func (g *Gate) Request() (run bool, wait time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending {
		return false, 0
	}
	elapsed := g.now().Sub(g.last)
	if g.last.IsZero() || elapsed >= g.interval {
		g.last = g.now()
		return true, 0
	}
	g.pending = true
	return false, g.interval - elapsed
}

// Due reports whether a held request is due, and so a read should be made now.
// It is false when a direct read (see Mark) already took the request's place.
func (g *Gate) Due() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.pending {
		return false
	}
	g.last, g.pending = g.now(), false
	return true
}
