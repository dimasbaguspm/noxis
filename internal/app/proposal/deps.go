// Package proposal holds the use cases that carry a proposed change: create, list, read, and
// rehearse. One file per use case, one Run each; the shared rehearsal step is a package helper, not
// another use case.
package proposal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"noxis/internal/app/password"
	"noxis/internal/common/ref"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"noxis/internal/ports"
)

// Deps is what every proposal use case is built from.
type Deps struct {
	Proposals  ports.Proposals
	Rehearsals ports.Rehearsals
	Targets    ports.Targets
	Journal    ports.Journal
	Sealer     ports.Sealer
	Driver     ports.Driver
	// Timeout bounds one rehearsal. It comes from QUERY_TIMEOUT: a rehearsal is a query, and a
	// query that hangs is a query that was never bounded.
	Timeout time.Duration
}

// rehearse proves one proposal and stores the evidence. Every attempt is stored, including the
// ones that failed: "we tried and it did not prove" belongs in the record.
func (d Deps) rehearse(ctx context.Context, proposal domain.Proposal) (domain.Proposal, domain.Rehearsal, error) {
	target, err := d.Targets.ByRef(ctx, proposal.TargetRef)
	if err != nil {
		return proposal, domain.Rehearsal{}, err
	}
	plain, _, err := password.Open(d.Sealer, target)
	if err != nil {
		return proposal, domain.Rehearsal{}, err
	}

	bounded, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()
	rehearsal, err := d.Driver.Rehearse(bounded, target, plain, domain.Change{
		Query:    proposal.Query,
		Rollback: proposal.Rollback,
		Tier:     proposal.Tier,
	})

	// One measurement per attempt, whatever the verdict: a rehearsal that proved nothing is still
	// work the engine did.
	telemetry.Rehearsal(ctx, string(rehearsal.Tier), string(rehearsal.Status), target.Driver,
		float64(rehearsal.DurationMS)/1000)
	rehearsal.Ref = ref.New()
	rehearsal.ProposalRef = proposal.Ref
	if rehearsal.CreatedAt.IsZero() {
		rehearsal.CreatedAt = time.Now().UTC()
	}
	if err != nil {
		// The engine could not be reached at all: there is no evidence to store, but the attempt
		// still has to be visible.
		proposal.Status = domain.StatusFailed
		proposal.Failure = err.Error()
		proposal.UpdatedAt = time.Now().UTC()
		stored, updateErr := d.Proposals.UpdateProposal(ctx, proposal)
		if updateErr != nil {
			return proposal, rehearsal, updateErr
		}
		return stored, rehearsal, err
	}

	if _, err := d.Rehearsals.CreateRehearsal(ctx, rehearsal); err != nil {
		return proposal, rehearsal, err
	}

	proposal.RehearsalRef = rehearsal.Ref
	proposal.Irreversible = rehearsal.Irreversible
	proposal.UpdatedAt = time.Now().UTC()
	proposal.Failure = ""
	if rehearsal.Usable() {
		proposal.Status = domain.StatusVerified
	} else {
		proposal.Status = domain.StatusFailed
		proposal.Failure = rehearsal.Error
		if strings.TrimSpace(proposal.Failure) == "" {
			proposal.Failure = string(rehearsal.Status) + ": " + rehearsal.Note
		}
	}
	stored, err := d.Proposals.UpdateProposal(ctx, proposal)
	if err != nil {
		return proposal, rehearsal, err
	}
	if err := d.record(ctx, stored, constants.JournalRehearsed, storageDetail(rehearsal)); err != nil {
		return stored, rehearsal, err
	}
	return stored, rehearsal, nil
}

// record appends one journal line about a proposal.
func (d Deps) record(ctx context.Context, subject domain.Proposal, kind string, detail map[string]string) error {
	return d.Journal.Append(ctx, domain.JournalEntry{
		Ref:         ref.New(),
		SubjectRef:  subject.Ref,
		SubjectKind: constants.SubjectProposal,
		Kind:        kind,
		Actor:       actorOf(subject),
		Detail:      detail,
		CreatedAt:   time.Now().UTC(),
	})
}

// actorOf is the caller that touched the proposal last: the proposer until someone decides, which
// is the truth until #754 adds decisions.
func actorOf(proposal domain.Proposal) string { return proposal.ProposedBy }

func storageDetail(rehearsal domain.Rehearsal) map[string]string {
	detail := map[string]string{
		"status": string(rehearsal.Status),
		"tier":   string(rehearsal.Tier),
	}
	if rehearsal.Irreversible {
		detail["irreversible"] = "true"
	}
	if rehearsal.Error != "" {
		detail["error"] = truncate(rehearsal.Error)
	}
	return detail
}

func truncate(message string) string {
	const limit = 300
	if len(message) <= limit {
		return message
	}
	return message[:limit] + "…"
}

// requireRef rejects anything that is not a reference before it becomes a lookup.
func requireRef(value string) (string, error) {
	if !ref.Valid(value) {
		return "", fmt.Errorf("%w: %q is not a reference", domain.ErrInvalid, value)
	}
	return value, nil
}
