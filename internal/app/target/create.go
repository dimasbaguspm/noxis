package target

import (
	"context"
	"errors"
	"noxis/internal/common/ref"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"time"
)

// Create registers one target. It is the only way a credential enters the state file.
type Create struct{ Deps }

// NewCreate builds the use case.
func NewCreate(d Deps) *Create { return &Create{Deps: d} }

// CreateInput is the transport-independent shape of a registration. Password is plaintext
// here and nowhere else: it is sealed before the target is stored. IdempotencyKey is the caller's
// name for this request, empty when the caller named none.
type CreateInput struct {
	Name           string
	Driver         string
	Host           string
	Port           int
	Database       string
	Username       string
	Password       string
	TLSMode        string
	Options        map[string]string
	Actor          string
	IdempotencyKey string
}

// CreateOutput is the target this call answers with, and whether that row was made by an earlier call
// carrying the same key. A replay is answered 200 instead of 201, and journals nothing new.
type CreateOutput struct {
	Target   domain.Target
	Replayed bool
}

// Run validates, seals, stores and journals. The returned target never carries the
// plaintext password.
func (c *Create) Run(ctx context.Context, in CreateInput) (out0 CreateOutput, err error) {
	ctx, end := telemetry.Span(ctx, "target.Create.Run")
	defer func() { end(err) }()

	if in.IdempotencyKey != "" {
		existing, lookupErr := c.Targets.ByKey(ctx, in.IdempotencyKey)
		switch {
		case lookupErr == nil:
			return CreateOutput{Target: existing, Replayed: true}, nil
		case !errors.Is(lookupErr, domain.ErrNotFound):
			return CreateOutput{}, lookupErr
		}
	}

	now := time.Now().UTC()
	target := domain.Target{
		Ref:            ref.New(),
		Name:           in.Name,
		Driver:         in.Driver,
		Host:           in.Host,
		Port:           in.Port,
		Database:       in.Database,
		Username:       in.Username,
		TLSMode:        in.TLSMode,
		Options:        in.Options,
		CreatedBy:      in.Actor,
		IdempotencyKey: in.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if target.Port == 0 {
		target.Port = constants.DefaultMySQLPort
	}
	if target.TLSMode == "" {
		target.TLSMode = constants.TLSPreferred
	}
	if err := target.Validate(); err != nil {
		return CreateOutput{}, err
	}
	sealed, err := c.seal(target.Ref, in.Password)
	if err != nil {
		return CreateOutput{}, err
	}
	target.Secret = sealed

	stored, err := c.Targets.Create(ctx, target)
	if err != nil {
		return CreateOutput{}, err
	}
	// A key collision inside the store means another call won the race with the same key: the row it
	// made is the answer, and this call journals nothing.
	if stored.Ref != target.Ref {
		return CreateOutput{Target: stored, Replayed: true}, nil
	}
	if err := c.record(ctx, stored, constants.JournalCreated, in.Actor, map[string]string{
		"name":   stored.Name,
		"driver": stored.Driver,
	}); err != nil {
		return CreateOutput{}, err
	}
	return CreateOutput{Target: stored}, nil
}
