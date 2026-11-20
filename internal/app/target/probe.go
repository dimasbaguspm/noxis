package target

import (
	"context"
	"fmt"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"time"
)

// Probe is the only use case that opens a connection to a target. It records what it found on
// the target itself — including a failure — because "nobody can reach this" is a fact the
// next reader needs as much as a version string.
type Probe struct{ Deps }

// NewProbe builds the use case.
func NewProbe(d Deps) *Probe { return &Probe{Deps: d} }

// ProbeInput names the target and the caller.
type ProbeInput struct {
	Ref   string
	Actor string
}

// ProbeOutput is what the probe measured, plus the target as it now stands.
type ProbeOutput struct {
	Target       domain.Target
	Capabilities domain.Capabilities
}

// Run connects, asks the server who it is, writes the answer back and journals it. A failed
// probe is stored and returned: the caller learns both.
func (p *Probe) Run(ctx context.Context, in ProbeInput) (ProbeOutput, error) {
	ref, err := requireRef(in.Ref)
	if err != nil {
		return ProbeOutput{}, err
	}
	target, err := p.Targets.ByRef(ctx, ref)
	if err != nil {
		return ProbeOutput{}, err
	}
	if target.Archived() {
		return ProbeOutput{}, fmt.Errorf("%w: target %q is archived", domain.ErrConflict, target.Name)
	}

	password, err := p.openPassword(target)
	if err != nil {
		return ProbeOutput{}, err
	}

	caps, probeErr := p.Driver.Probe(ctx, target, password)

	now := time.Now().UTC()
	target.LastProbeAt = &now
	target.UpdatedAt = now
	if probeErr != nil {
		target.LastProbeStatus = constants.ProbeFailed
		stored, err := p.Targets.Update(ctx, target)
		if err != nil {
			return ProbeOutput{}, err
		}
		if err := p.record(ctx, stored, constants.JournalProbed, in.Actor, map[string]string{
			"status": constants.ProbeFailed,
			"error":  truncate(probeErr.Error()),
		}); err != nil {
			return ProbeOutput{}, err
		}
		return ProbeOutput{Target: stored}, probeErr
	}

	target.LastProbeStatus = constants.ProbeOK
	target.ServerVersion = caps.Version
	stored, err := p.Targets.Update(ctx, target)
	if err != nil {
		return ProbeOutput{}, err
	}
	if err := p.record(ctx, stored, constants.JournalProbed, in.Actor, map[string]string{
		"status":  constants.ProbeOK,
		"flavor":  caps.Flavor,
		"version": caps.Version,
	}); err != nil {
		return ProbeOutput{}, err
	}
	return ProbeOutput{Target: stored, Capabilities: caps}, nil
}

// truncate keeps a journal line readable: the driver's message is quoted, not archived whole.
func truncate(message string) string {
	const limit = 300
	if len(message) <= limit {
		return message
	}
	return message[:limit] + "…"
}
