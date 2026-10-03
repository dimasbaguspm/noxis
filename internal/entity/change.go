package entity

// Change is a proposed database change: the SQL to run plus the SQL that
// undoes it, submitted with notes and validated before anyone approves it.
// The flow is submit → validate → approve → apply; self-approval is allowed
// (single-operator tool).
type Change struct {
	ID         string        `json:"id"`
	Connection string        `json:"connection" validate:"required"`
	Notes      string        `json:"notes" validate:"required"`
	Query      string        `json:"query" validate:"required"`
	Rollback   string        `json:"rollback" validate:"required"`
	Status     string        `json:"status"` // validated | failed | approved | applied
	Checks     []ChangeCheck `json:"checks,omitempty"`
	CreatedAt  int64         `json:"createdAt"`
	ApprovedAt int64         `json:"approvedAt,omitempty"`
	AppliedAt  int64         `json:"appliedAt,omitempty"`
}

// ChangeCheck is one validation outcome (the query and its rollback are each
// dry-run in a transaction that rolls back — nothing persists).
type ChangeCheck struct {
	Target string `json:"target"` // query | rollback
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Validate checks the required fields.
func (c Change) Validate() error { return validate.Struct(c) }
