package handlers

// The request bodies, named. They are types rather than inline structs so the generated document can
// name the shape it publishes: an annotation that points at a type the compiler checks is a
// document that cannot drift from the handler that decodes it.

// CreateTargetRequest is what a client registers a server with. The password travels once, here,
// and never comes back.
type CreateTargetRequest struct {
	Name     string            `json:"name" example:"prod"`
	Driver   string            `json:"driver" example:"mysql" enums:"mysql,mariadb"`
	Host     string            `json:"host" example:"db.internal"`
	Port     int               `json:"port" example:"3306"`
	Database string            `json:"database" example:"app"`
	Username string            `json:"username" example:"noxis"`
	Password string            `json:"password" example:"not-on-the-wire-back"`
	TLSMode  string            `json:"tls_mode" example:"preferred"`
	Options  map[string]string `json:"options,omitempty"`
}

// UpdateTargetRequest is a patch: a field that is absent is a field nobody asked to change.
type UpdateTargetRequest struct {
	Name     *string            `json:"name,omitempty"`
	Username *string            `json:"username,omitempty"`
	Password *string            `json:"password,omitempty"`
	TLSMode  *string            `json:"tls_mode,omitempty"`
	Options  *map[string]string `json:"options,omitempty"`
}

// QueryRequest is one statement a read route may run. `limit` bounds the rows it returns.
type QueryRequest struct {
	SQL   string `json:"sql" example:"SELECT id FROM users"`
	Limit int    `json:"limit,omitempty" example:"100"`
}

// CreateProposalRequest is a change and the rollback that would undo it. An empty rollback makes the
// change irreversible, which an apply must then acknowledge by name.
type CreateProposalRequest struct {
	Title    string `json:"title" example:"add a column"`
	Query    string `json:"query" example:"ALTER TABLE users ADD COLUMN note TEXT"`
	Rollback string `json:"rollback,omitempty" example:"ALTER TABLE users DROP COLUMN note"`
	Note     string `json:"note,omitempty"`
	Tier     string `json:"tier,omitempty" enums:"static,shadow,rehearsal"`
}

// RehearseRequest re-runs the proof, optionally at a different tier.
type RehearseRequest struct {
	Tier string `json:"tier,omitempty" enums:"static,shadow,rehearsal"`
}

// DecisionRequest is the sentence a reviewer leaves behind.
type DecisionRequest struct {
	Note string `json:"note,omitempty"`
}

// ApplyRequest acknowledges a change that cannot be undone.
type ApplyRequest struct {
	ConfirmDestructive bool `json:"confirm_destructive,omitempty"`
}
