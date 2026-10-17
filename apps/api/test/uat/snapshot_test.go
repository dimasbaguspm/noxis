package uat

import "testing"

func TestSnapshotterFor(t *testing.T) {
	for _, driver := range []string{"postgres", "mariadb"} {
		if _, err := snapshotterFor(target{Driver: driver}); err != nil {
			t.Errorf("%s: %v", driver, err)
		}
	}
	if _, err := snapshotterFor(target{Driver: "redis"}); err == nil {
		t.Error("redis has no snapshotter; want an error")
	}
}
