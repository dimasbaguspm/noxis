package entity

// SearchTarget is one index a search platform holds, with what the platform reports about it.
type SearchTarget struct {
	Name      string `json:"name"`
	Documents int64  `json:"documents"`
	SizeBytes int64  `json:"sizeBytes"`
}

// SearchRequest is one request against a search platform: the target it reads and the platform's own query document,
// which no other engine can interpret, so it travels as text rather than as a shape shared with SQL or documents.
type SearchRequest struct {
	Op     string `json:"op"`              // search | delete
	Target string `json:"target"`          // the index the request reads
	Query  string `json:"query,omitempty"` // the platform's own query document, empty means everything
	Limit  int64  `json:"limit,omitempty"` // how many hits a read renders
}

// SearchTargetList is the targets envelope, so the console reads one field instead of a bare array.
type SearchTargetList struct {
	Targets []SearchTarget `json:"targets"`
}

// SearchChangeRequest is one submitted search change: the connection and notes of a change, plus the request itself,
// flat so the console sends one object.
type SearchChangeRequest struct {
	Connection string `json:"connection"`
	Notes      string `json:"notes"`
	SearchRequest
}
