package entity

// Result is one operation's outcome on any engine: a grid for reads, counts
// and timings for writes. It is the wire shape shared by the SQL and Redis
// modules, so the console renders both with one component.
type Result struct {
	Columns       []string   `json:"columns,omitempty"`
	Rows          [][]string `json:"rows,omitempty"`
	Affected      int64      `json:"affected"`
	RowsReturned  int        `json:"rowsReturned"`
	EstimatedRows int64      `json:"estimatedRows,omitempty"`
	DurationMs    int64      `json:"durationMs"`
	Message       string     `json:"message"`
}
