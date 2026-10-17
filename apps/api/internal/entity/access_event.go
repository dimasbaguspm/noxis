package entity

// AccessEvent is one line of the access audit trail: who did what, to whom, on
// which connection, and when. The table is append-only.
type AccessEvent struct {
	ID         string `json:"id"`
	At         int64  `json:"at"`
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	Subject    string `json:"subject,omitempty"`
	Connection string `json:"connection,omitempty"`
	Level      string `json:"level,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Note       string `json:"note,omitempty"`
}
