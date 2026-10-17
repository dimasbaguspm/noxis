package entity

// Access request statuses.
const (
	RequestPending   = "pending"
	RequestApproved  = "approved"
	RequestRejected  = "rejected"
	RequestCancelled = "cancelled"
	RequestExpired   = "expired"
)

// AccessRequest is a user asking for a level on one connection, and the
// decision made on it. The row is the audit of who asked, who decided, and when.
type AccessRequest struct {
	ID           string `json:"id"`
	Subject      string `json:"subject"`
	Connection   string `json:"connection"`
	Level        string `json:"level"`
	Reason       string `json:"reason,omitempty"`
	Status       string `json:"status"`
	RequestedAt  int64  `json:"requestedAt"`
	DecidedBy    string `json:"decidedBy,omitempty"`
	DecidedAt    int64  `json:"decidedAt,omitempty"`
	DecisionNote string `json:"decisionNote,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"`
}
