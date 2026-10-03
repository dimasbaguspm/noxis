package entity

// Change is a proposed database change (SQL + rollback + notes), moving
// through submit → validate → approve → apply. Self-approval is allowed.
type Change struct {
	ID         string        `json:"id"`
	Connection string        `json:"connection" validate:"required"`
	Notes      string        `json:"notes" validate:"required"`
	Query      string        `json:"query" validate:"required"`
	Rollback   string        `json:"rollback" validate:"required"`
	Kind       string        `json:"kind"`   // dml | ddl
	Status     string        `json:"status"` // validated | unverified | failed | approved | applied | rejected
	Checks     []ChangeCheck `json:"checks,omitempty"`
	CreatedAt  int64         `json:"createdAt"`
	ApprovedAt int64         `json:"approvedAt,omitempty"`
	AppliedAt  int64         `json:"appliedAt,omitempty"`
	RejectedAt int64         `json:"rejectedAt,omitempty"`
	// what the batch touches (keyword-anchored, best effort)
	AffectedTables  []string `json:"affectedTables,omitempty"`
	AffectedColumns []string `json:"affectedColumns,omitempty"`
	// measured on the dry run (estimates) and on apply (actuals)
	EstDurationMs     int64 `json:"estDurationMs,omitempty"`
	EstAffected       int64 `json:"estAffected,omitempty"`
	AppliedAffected   int64 `json:"appliedAffected,omitempty"`
	AppliedDurationMs int64 `json:"appliedDurationMs,omitempty"`
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
