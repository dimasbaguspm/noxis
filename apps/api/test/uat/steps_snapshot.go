package uat

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
)

func (w *world) snapshotSteps(sc *godog.ScenarioContext) {
	sc.Step(`^the database "([^"]*)" is snapshotted as "([^"]*)"$`, w.snapshotDatabase)
	sc.Step(`^the "([^"]*)" snapshot of "([^"]*)" is restored$`, w.restoreDatabase)
}

// snapshotDatabase dumps the connection under test and keeps it in memory for the rest of the scenario; nothing is written to the repository.
func (w *world) snapshotDatabase(ctx context.Context, conn, key string) error {
	t := targets[w.targetKey]
	s, err := snapshotterFor(t)
	if err != nil {
		return err
	}
	data, err := s.dump(t)
	if err != nil {
		return err
	}
	if w.snaps == nil {
		w.snaps = map[string]dbSnapshot{}
	}
	w.snaps[key] = dbSnapshot{target: t, data: data}
	return nil
}

// restoreDatabase puts a named dump back into the target it came from.
func (w *world) restoreDatabase(ctx context.Context, key, conn string) error {
	snap, ok := w.snaps[key]
	if !ok {
		return fmt.Errorf("no snapshot %q was taken", key)
	}
	s, err := snapshotterFor(snap.target)
	if err != nil {
		return err
	}
	return s.load(snap.target, snap.data)
}
