// Package proposal holds the use cases that carry a proposed change: create, list, read, and
// rehearse. One file per use case, one Run each; the shared rehearsal step is a package helper, not
// another use case.
package proposal

import (
	"context"
	"fmt"
	"noxis/internal/app/password"
	"noxis/internal/common/ref"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"strings"
	"time"
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
	// ProbeTimeout bounds the probe that follows a successful apply, from PROBE_TIMEOUT: it is a
	// connection, and a connection that hangs is one nobody bounded.
	ProbeTimeout time.Duration
}

// rehearse proves one proposal and stores the evidence. Every attempt is stored, including the
// ones that failed: "we tried and it did not prove" belongs in the record.
func (d Deps) rehearse(ctx context.Context, proposal domain.Proposal) (domain.Proposal, domain.Rehearsal, error) {
	target, err := d.Targets.ByRef(ctx, proposal.TargetRef)
	if err != nil {
		return proposal, domain.Rehearsal{}, err
	}
	plain, err := password.Open(d.Sealer, target)
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
		// The engine could not be reached at all. There is no diff to show, but the attempt is still
		// an attempt: it is stored as a failed rehearsal, exactly like one that proved nothing.
		rehearsal.Status = domain.RehearsalFailed
		rehearsal.Error = err.Error()
		if _, storeErr := d.Rehearsals.CreateRehearsal(ctx, rehearsal); storeErr != nil {
			return proposal, rehearsal, storeErr
		}
		proposal.RehearsalRef = rehearsal.Ref
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
	// The proposal records the tier that actually ran, not the one that was asked for: a rehearsal can
	// degrade to a static check, and this is the field a reviewer reads to know what was proven.
	if rehearsal.Tier != "" {
		proposal.Tier = rehearsal.Tier
	}
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

// reprobe records what the engine is now that a change landed. It runs after the apply is committed, so
// it is deliberately best-effort: a probe that fails is a fact about the connection, not about the
// change, and it is stored as a failed probe rather than raised as a failed apply.
func (d Deps) reprobe(ctx context.Context, target domain.Target, plain, actor string) error {
	// A caller that configured no probe timeout means "the caller's context bounds it", not "expire now".
	bounded, cancel := ctx, context.CancelFunc(func() {})
	if d.ProbeTimeout > 0 {
		bounded, cancel = context.WithTimeout(ctx, d.ProbeTimeout)
	}
	defer cancel()

	caps, probeErr := d.Driver.Probe(bounded, target, plain)
	now := time.Now().UTC()
	target.LastProbeAt = &now
	target.UpdatedAt = now
	detail := map[string]string{"status": constants.ProbeOK, "flavor": caps.Flavor, "version": caps.Version}
	if probeErr != nil {
		target.LastProbeStatus = constants.ProbeFailed
		detail = map[string]string{"status": constants.ProbeFailed, "error": truncate(probeErr.Error())}
	} else {
		target.LastProbeStatus = constants.ProbeOK
		target.ServerVersion = caps.Version
	}
	stored, err := d.Targets.Update(bounded, target)
	if err != nil {
		return err
	}
	return d.Journal.Append(bounded, domain.JournalEntry{
		Ref:         ref.New(),
		SubjectRef:  stored.Ref,
		SubjectKind: constants.SubjectTarget,
		Kind:        constants.JournalProbed,
		Actor:       actor,
		Detail:      detail,
		CreatedAt:   now,
	})
}
