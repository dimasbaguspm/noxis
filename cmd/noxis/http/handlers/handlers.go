// Package handlers is the HTTP transport. A handler reads a request, calls one use case
// and writes an envelope: no SQL, no policy and no business rule lives here.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"noxis/internal/app/proposal"
	"noxis/internal/app/read"
	"noxis/internal/app/target"
	"noxis/internal/common/release"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
)

// Deps is what the handlers call. It is assembled once in the composition root and holds
// use cases only — never a store, a driver or a database handle.
type Deps struct {
	Log       *slog.Logger
	Targets   TargetUseCases
	Read      ReadUseCases
	Proposals ProposalUseCases
	// Release answers what the publish job last uploaded, for the version route. It is the only thing
	// the API asks a third party, and a failure there is a missing notice rather than an error.
	Release ReleaseChecker
}

// ReleaseChecker is the one question the version route asks.
type ReleaseChecker interface {
	Latest(ctx context.Context) (*release.Info, error)
}

// ProposalUseCases are the proposal use cases, one field per route.
type ProposalUseCases struct {
	Create   *proposal.Create
	List     *proposal.List
	Get      *proposal.Get
	Rehearse *proposal.Rehearse
	Decide   *proposal.Decide
	Apply    *proposal.Apply
	Events   *proposal.Events
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
	Test    *target.Test
}

// Handlers carries Deps so a route table can name one method per endpoint.
type Handlers struct{ Deps }

// New builds the handler set.
func New(d Deps) *Handlers { return &Handlers{Deps: d} }

// idempotencyKey reads the caller's name for this create, or "" when they named none. A key that is too
// long or holds anything but printable ASCII is refused rather than trimmed: two different keys that
// were cut to the same prefix would be one key, and one create would be silently lost.
func idempotencyKey(r *http.Request) (string, error) {
	key := strings.TrimSpace(r.Header.Get(constants.HeaderIdempotencyKey))
	if key == "" {
		return "", nil
	}
	if len(key) > constants.MaxIdempotencyKeyLen {
		return "", fmt.Errorf("%w: %s is longer than %d characters",
			domain.ErrInvalid, constants.HeaderIdempotencyKey, constants.MaxIdempotencyKeyLen)
	}
	for _, char := range key {
		if char < 0x20 || char > 0x7e {
			return "", fmt.Errorf("%w: %s may hold printable ASCII only", domain.ErrInvalid, constants.HeaderIdempotencyKey)
		}
	}
	return key, nil
}
