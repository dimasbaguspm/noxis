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
	Status     string        `json:"status"` // validated | failed | approved | applied
	Checks     []ChangeCheck `json:"checks,omitempty"`
	CreatedAt  int64         `json:"createdAt"`
	ApprovedAt int64         `json:"approvedAt,omitempty"`
	AppliedAt  int64         `json:"appliedAt,omitempty"`
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
