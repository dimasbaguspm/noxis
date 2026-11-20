package constants

import "time"

// ApplyBackgroundRows is the size past which apply stops holding the request open. Below it the
// statement is fast enough to answer in the call; above it the caller gets 202 and polls, because a
// table copy of a million rows outlives every sensible client timeout.
const ApplyBackgroundRows = 100_000

// ApplyProgressInterval is how often the running statement is asked what it is doing. Frequent
// enough to see a copy move, rare enough that the watcher never costs more than the statement.
const ApplyProgressInterval = 2 * time.Second
