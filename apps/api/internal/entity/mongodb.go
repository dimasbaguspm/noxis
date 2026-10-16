package entity

// Database is one database with its size, as `dbStats` reports it.
type Database struct {
	Name           string `json:"name"`
	SizeBytes      int64  `json:"sizeBytes"`
	StorageBytes   int64  `json:"storageBytes"`
	IndexSizeBytes int64  `json:"indexSizeBytes"`
	Collections    int64  `json:"collections"`
	Objects        int64  `json:"objects"`
	AvgObjectSize  int64  `json:"avgObjectSizeBytes"`
}

// Collection is one collection with its size, as `collStats` reports it.
type Collection struct {
	Name           string `json:"name"`
	Type           string `json:"type"` // collection | view | timeseries
	Count          int64  `json:"count"`
	SizeBytes      int64  `json:"sizeBytes"`
	StorageBytes   int64  `json:"storageBytes"`
	AvgObjectSize  int64  `json:"avgObjectSizeBytes"`
	Indexes        int64  `json:"indexes"`
	IndexSizeBytes int64  `json:"indexSizeBytes"`
	Capped         bool   `json:"capped"`
}

// MongoRequest is one CRUD request: the filter and the value are separate
// documents, both written as MongoDB Extended JSON.
type MongoRequest struct {
	Op         string `json:"op"`
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Filter     string `json:"filter,omitempty"`
	Value      string `json:"value,omitempty"`
	Sort       string `json:"sort,omitempty"`
	Limit      int64  `json:"limit,omitempty"`
	Many       bool   `json:"many,omitempty"`
}
