package entity

// NodeStatus is the last echo a slave reported about its runtime: actual
// state lives on the machine (docker ps), this is only the master's window
// into it. One row per node, updated on every report.
type NodeStatus struct {
	Node   string
	State  string // ok | error | up | offline
	Detail string
	TS     int64 // unix seconds
}

// Slave is a row in the master's registry (sqlite, slaves table).
type Slave struct {
	ID           string
	Name         string
	IP           string
	RegisteredAt int64 // unix seconds
}
