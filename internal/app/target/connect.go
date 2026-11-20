package target

import (
	"context"
	"fmt"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
)

// Test proves a configuration nobody has stored yet. It opens one connection with the
// credentials as typed, so a typo is refused in the form that produced it instead of becoming
// a row the next operator has to debug. Nothing is written: no target, no journal line.
type Test struct{ Deps }

// NewTest builds the use case.
func NewTest(d Deps) *Test { return &Test{Deps: d} }

// TestInput is a registration that has not been made. It is CreateInput without the identity
// the state file would give it.
type TestInput struct {
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	TLSMode  string
	Options  map[string]string
}

// Run connects and disconnects. The answer is what the engine said about itself.
func (t *Test) Run(ctx context.Context, in TestInput) (_ domain.Capabilities, err error) {
	ctx, end := telemetry.Span(ctx, "target.Test.Run")
	defer func() { end(err) }()

	if !constants.IsDriver(in.Driver) {
		return domain.Capabilities{}, fmt.Errorf("%w: driver %q is not one of %s",
			domain.ErrInvalid, in.Driver, strings.Join(constants.Drivers, ", "))
	}
	if strings.TrimSpace(in.Host) == "" {
		return domain.Capabilities{}, fmt.Errorf("%w: host is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(in.Database) == "" {
		return domain.Capabilities{}, fmt.Errorf("%w: database is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(in.Username) == "" {
		return domain.Capabilities{}, fmt.Errorf("%w: username is required", domain.ErrInvalid)
	}
	tlsMode := in.TLSMode
	if tlsMode == "" {
		tlsMode = constants.TLSPreferred
	}
	if !constants.IsTLSMode(tlsMode) {
		return domain.Capabilities{}, fmt.Errorf("%w: tls mode %q is not one of %s",
			domain.ErrInvalid, tlsMode, strings.Join(constants.TLSModes, ", "))
	}
	port := in.Port
	if port == 0 {
		port = constants.DefaultMySQLPort
	}
	if port < 1 || port > 65535 {
		return domain.Capabilities{}, fmt.Errorf("%w: port %d is outside 1-65535",
			domain.ErrInvalid, port)
	}

	// The draft carries no ref and no sealed secret: a probe reads only what the driver needs to
	// dial, and there is nothing here to seal.
	draft := domain.Target{
		Driver:   in.Driver,
		Host:     in.Host,
		Port:     port,
		Database: in.Database,
		Username: in.Username,
		TLSMode:  tlsMode,
		Options:  in.Options,
	}
	return t.Driver.Probe(ctx, draft, in.Password)
}
