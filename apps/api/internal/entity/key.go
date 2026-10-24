package entity

// Key is one memory-cache browse entry: its name, type and TTL (-1 = no expiry). KeyPage is a browse result; Truncated reports that the keyspace held more keys than the limit, so the counts are a lower bound.
type Key struct {
	Name string `json:"name"`
	Type string `json:"type"`
	TTL  int64  `json:"ttlMs"`
	Size int64  `json:"sizeBytes"`
}

type KeyPage struct {
	Keys      []Key  `json:"keys"`
	Total     int64  `json:"total"`
	Truncated bool   `json:"truncated"`
	Pattern   string `json:"pattern"`
}
