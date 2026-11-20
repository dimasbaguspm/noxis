package target

import (
	"context"
	"noxis/internal/common/ref"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"time"
)

// Create registers one target. It is the only way a credential enters the state file.
type Create struct{ Deps }

// NewCreate builds the use case.
func NewCreate(d Deps) *Create { return &Create{Deps: d} }

// CreateInput is the transport-independent shape of a registration. Password is plaintext
// here and nowhere else: it is sealed before the target is stored.
type CreateInput struct {
	Name     string
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	TLSMode  string
	Options  map[string]string
	Actor    string
}

// Run validates, seals, stores and journals. The returned target never carries the
// plaintext password.
func (c *Create) Run(ctx context.Context, in CreateInput) (domain.Target, error) {
	now := time.Now().UTC()
	target := domain.Target{
		Ref:       ref.New(),
		Name:      in.Name,
		Driver:    in.Driver,
		Host:      in.Host,
		Port:      in.Port,
		Database:  in.Database,
		Username:  in.Username,
		TLSMode:   in.TLSMode,
		Options:   in.Options,
		CreatedBy: in.Actor,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if target.Port == 0 {
		target.Port = constants.DefaultMySQLPort
	}
	if target.TLSMode == "" {
		target.TLSMode = constants.TLSPreferred
	}
	if err := target.Validate(); err != nil {
		return domain.Target{}, err
	}
	sealed, err := c.seal(target.Ref, in.Password)
	if err != nil {
		return domain.Target{}, err
	}
	target.Secret = sealed

	stored, err := c.Targets.Create(ctx, target)
	if err != nil {
		return domain.Target{}, err
	}
	if err := c.record(ctx, stored, constants.JournalCreated, in.Actor, map[string]string{
		"name":   stored.Name,
		"driver": stored.Driver,
	}); err != nil {
		return domain.Target{}, err
	}
	return stored, nil
}
