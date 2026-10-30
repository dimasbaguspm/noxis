package entity

// Change is a proposed database change (SQL + rollback + notes), moving through submit → validate → approve → apply.
type Change struct {
	ID          string        `json:"id"`
	Connection  string        `json:"connection" validate:"required"`
	// Driver is derived at read time from the connection that serves the change; it is never stored.
	Driver      string        `json:"driver,omitempty"`
	Notes       string        `json:"notes" validate:"required"`
	Query       string        `json:"query" validate:"required"`
	Rollback    string        `json:"rollback" validate:"required"`
	Kind        string        `json:"kind"`   // dml | ddl
	Status      string        `json:"status"` // validated | unverified | failed | approved | applying | applied | rejected
	Checks      []ChangeCheck `json:"checks,omitempty"`
	CreatedAt   int64         `json:"createdAt"`
	CreatedBy   string        `json:"createdBy,omitempty"`
	ApprovedAt  int64         `json:"approvedAt,omitempty"`
	ApprovedBy  string        `json:"approvedBy,omitempty"`
	ApplyingAt  int64         `json:"applyingAt,omitempty"`
	AppliedAt   int64         `json:"appliedAt,omitempty"`
	AppliedBy   string        `json:"appliedBy,omitempty"`
	RejectedAt  int64         `json:"rejectedAt,omitempty"`
	RejectedBy  string        `json:"rejectedBy,omitempty"`
	ScheduledBy string        `json:"scheduledBy,omitempty"`
	// what the batch touches (keyword-anchored, best effort)
	AffectedTables  []string `json:"affectedTables,omitempty"`
	AffectedColumns []string `json:"affectedColumns,omitempty"`
	// measured on the dry run (estimates) and on apply (actuals)
	EstDurationMs     int64  `json:"estDurationMs,omitempty"`
	EstAffected       int64  `json:"estAffected,omitempty"`
	AppliedAffected   int64  `json:"appliedAffected,omitempty"`
	AppliedDurationMs int64  `json:"appliedDurationMs,omitempty"`
	ScratchSchema     string `json:"scratchSchema,omitempty"`
	// ScheduledAt is the unix time the change should run; 0 means execute immediately (on manual apply, as before).
	ScheduledAt int64 `json:"scheduledAt,omitempty"`
}

// ChangeCheck is one validation outcome.
type ChangeCheck struct {
	Target string `json:"target"` // query | rollback
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Validate checks the required fields.
func (c Change) Validate() error { return validate.Struct(c) }
