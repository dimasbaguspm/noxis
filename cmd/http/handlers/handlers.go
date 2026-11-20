// Package handlers is the HTTP transport. A handler reads a request, calls one use case
// and writes an envelope: no SQL, no policy and no business rule lives here.
package handlers

import (
	"log/slog"

	"noxis/internal/app/proposal"
	"noxis/internal/app/read"
	"noxis/internal/app/target"
)

// Deps is what the handlers call. It is assembled once in the composition root and holds
// use cases only — never a store, a driver or a database handle.
type Deps struct {
	Log       *slog.Logger
	Targets   TargetUseCases
	Read      ReadUseCases
	Proposals ProposalUseCases
}

// ProposalUseCases are the proposal use cases, one field per route.
type ProposalUseCases struct {
	Create   *proposal.Create
	List     *proposal.List
	Get      *proposal.Get
	Rehearse *proposal.Rehearse
}

// ReadUseCases are the read-only use cases, one field per route.
type ReadUseCases struct {
	Query     *read.Query
	Databases *read.Databases
	Tables    *read.Tables
	Table     *read.Table
}

// TargetUseCases are the target use cases, one field per endpoint group so a route table
// can name exactly one of them.
type TargetUseCases struct {
	Create  *target.Create
	List    *target.List
	Get     *target.Get
	Update  *target.Update
	Archive *target.Archive
	Probe   *target.Probe
}

// Handlers carries Deps so a route table can name one method per endpoint.
type Handlers struct{ Deps }

// New builds the handler set.
func New(d Deps) *Handlers { return &Handlers{Deps: d} }
