package constants

// What a probe left on the target. A failed probe is stored, not swallowed: the next reader
// should see that the last attempt failed and when.
const (
	ProbeOK     = "ok"
	ProbeFailed = "failed"
)
