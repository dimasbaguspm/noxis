package uat

import "testing"

func TestSnapshotterFor(t *testing.T) {
	for _, driver := range []string{"postgres", "mysql", "mariadb", "mongo", "redis", "valkey"} {
		if _, err := snapshotterFor(target{Driver: driver}); err != nil {
			t.Errorf("%s: %v", driver, err)
		}
	}
	if _, err := snapshotterFor(target{Driver: "cassandra"}); err == nil {
		t.Error("an unsupported driver has no snapshotter; want an error")
	}
}
