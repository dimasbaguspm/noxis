package entity

// Table and Column describe SQL relations and catalog-declared keys; names never imply constraints.
type Table struct {
	ID            string   `json:"id,omitempty"`
	Schema        string   `json:"schema"`
	Name          string   `json:"name"`
	Columns       []Column `json:"columns"`
	Indexes       []Index  `json:"indexes,omitempty"`
	EstimatedRows *int64   `json:"estimatedRows,omitempty"`
}

// Index is one catalog index with its columns in declaration order.
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique,omitempty"`
	Primary bool     `json:"primary,omitempty"`
}

// Column is one attribute. PrimaryKey and Unique are declared constraints; References is set only by a declared foreign key.
type Column struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Nullable   bool       `json:"nullable"`
	PrimaryKey bool       `json:"primaryKey,omitempty"`
	Unique     bool       `json:"unique,omitempty"`
	References *Reference `json:"references,omitempty"`
}

// Reference is the column a foreign key points at.
type Reference struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Column string `json:"column"`
}

// SchemaGraph contains a redacted connection and its declared tables, keys and references.
type SchemaGraph struct {
	Connection SchemaConnection `json:"connection"`
	Tables     []Table          `json:"tables"`
}

// SchemaConnection identifies which connection the diagram describes.
type SchemaConnection struct {
	Name     string `json:"name"`
	Driver   string `json:"driver"`
	Family   string `json:"family"`
	Database string `json:"database"`
}
