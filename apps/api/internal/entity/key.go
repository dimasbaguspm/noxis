package entity

// Key is one cache entry; TTL -1 means no expiry. KeyPage.Truncated marks counts as lower bounds.
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
