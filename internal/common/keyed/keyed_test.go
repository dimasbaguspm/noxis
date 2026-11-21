package keyed

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// One name, one holder: the second caller waits for the first, which is what makes two applies against
// the same target a queue instead of a race.
func TestLockSerialisesOneName(t *testing.T) {
	var group Group
	var inside atomic.Int32
	var overlaps atomic.Int32

	unlock := group.Lock("target-a")
	var workers sync.WaitGroup
	workers.Add(4)
	for range 4 {
		go func() {
			defer workers.Done()
			release := group.Lock("target-a")
			defer release()
			if inside.Add(1) > 1 {
				overlaps.Add(1)
			}
			inside.Add(-1)
		}()
	}

	// Nobody gets in while the first holder is here.
	time.Sleep(20 * time.Millisecond)
	if got := inside.Load(); got != 0 {
		t.Fatalf("%d goroutines were inside while the lock was held", got)
	}
	unlock()
	workers.Wait()

	if overlaps.Load() != 0 {
		t.Fatalf("%d overlapping holders, want none", overlaps.Load())
	}
}

// Different names are different locks: two servers under one API must not queue behind each other.
func TestLockIsPerName(t *testing.T) {
	var group Group
	unlock := group.Lock("target-a")

	done := make(chan struct{})
	go func() {
		release := group.Lock("target-b")
		release()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("a second name waited for the first: the group is one lock, not one per name")
	}
	unlock()
}

// A use case built without a group still runs: the zero value is usable and a nil group is a no-op.
func TestNilGroupIsUsable(t *testing.T) {
	t.Parallel()
	var group *Group
	unlock := group.Lock("anything")
	unlock()
}
