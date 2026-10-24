package entity

// Result is the shared engine outcome: read grids, write counts, and timings.
type Result struct {
	Columns       []string   `json:"columns,omitempty"`
	Rows          [][]string `json:"rows,omitempty"`
	Affected      int64      `json:"affected"`
	RowsReturned  int        `json:"rowsReturned"`
	EstimatedRows int64      `json:"estimatedRows,omitempty"`
	DurationMs    int64      `json:"durationMs"`
	Message       string     `json:"message"`
	// ScratchSchema is set by change validation: the throwaway schema the change was verified against, if any.
	ScratchSchema string `json:"scratchSchema,omitempty"`
	// Documents carries raw engine output as Extended JSON (MongoDB reads).
	Documents []string `json:"documents,omitempty"`
}
