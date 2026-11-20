package domain

// Column is one result column, typed as the engine names it: normalising the type would be a
// different string per engine pretending to be one.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Rows is one bounded result set. Truncated is the honest answer where a keyset cannot be
// inferred from arbitrary SQL: there is no cursor to follow, so the API says what it did.
type Rows struct {
	Columns    []Column    `json:"columns"`
	Values     [][]*string `json:"values"`
	RowCount   int         `json:"row_count"`
	Truncated  bool        `json:"truncated"`
	DurationMS int64       `json:"duration_ms"`
}

// Table is one table in a schema, as the engine reports it. Rows is an estimate from
// information_schema, not a count: a count would cost what the reviewer is trying to avoid.
type Table struct {
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	Rows      int64  `json:"rows"`
	SizeBytes int64  `json:"size_bytes"`
}

// ColumnInfo is a column's definition, not a result column.
type ColumnInfo struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Key      string  `json:"key"`
	Extra    string  `json:"extra"`
	Default  *string `json:"default"`
}

// IndexInfo is one index, in key order.
type IndexInfo struct {
	Name    string   `json:"name"`
	Unique  bool     `json:"unique"`
	Columns []string `json:"columns"`
}

// TableDetail is what a reviewer needs about one table before proposing a change to it.
type TableDetail struct {
	Table
	Columns   []ColumnInfo `json:"columns"`
	Indexes   []IndexInfo  `json:"indexes"`
	CreateSQL string       `json:"create_sql"`
}
