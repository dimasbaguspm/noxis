// Package keyed is one mutex per name. The API is a single process, and the state file is written by one
// connection, but two applies against the same target are still two statements racing for the same DDL
// lock: the second one loses at the engine and the journal records a failure that was avoidable. A
// per-target lock turns that into a queue, and a queue is what an operator expects.
package keyed

import "sync"

// Group holds one mutex per name, created on first use. The zero value is ready to use, and a nil Group
// is a no-op: a use case built without one still works, it just does not serialise.
type Group struct {
	mu    sync.Mutex
	byKey map[string]*sync.Mutex
}

// Lock takes the lock for name and returns the function that releases it.
func (g *Group) Lock(name string) func() {
	if g == nil {
		return func() {}
	}
	g.mu.Lock()
	if g.byKey == nil {
		g.byKey = make(map[string]*sync.Mutex)
	}
	lock, ok := g.byKey[name]
	if !ok {
		lock = &sync.Mutex{}
		g.byKey[name] = lock
	}
	g.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}
