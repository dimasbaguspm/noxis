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

// TestOutput is what the engine said about itself and which schemas those credentials can see.
// The schemas are the point: the form that is about to register the instance asks the server what
// it governs instead of asking the operator to guess a name.
type TestOutput struct {
	Capabilities domain.Capabilities
	Databases    []string
}

// Run connects, asks who the engine is, lists the schemas it can reach, and disconnects. Database
// is optional here on purpose: a test whose job is to say which schemas exist cannot demand one
// before it has looked.
func (t *Test) Run(ctx context.Context, in TestInput) (_ TestOutput, err error) {
	ctx, end := telemetry.Span(ctx, "target.Test.Run")
	defer func() { end(err) }()

	if !constants.IsDriver(in.Driver) {
		return TestOutput{}, fmt.Errorf("%w: driver %q is not one of %s",
			domain.ErrInvalid, in.Driver, strings.Join(constants.Drivers, ", "))
	}
	if strings.TrimSpace(in.Host) == "" {
		return TestOutput{}, fmt.Errorf("%w: host is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(in.Username) == "" {
		return TestOutput{}, fmt.Errorf("%w: username is required", domain.ErrInvalid)
	}
	tlsMode := in.TLSMode
	if tlsMode == "" {
		tlsMode = constants.TLSPreferred
	}
	if !constants.IsTLSMode(tlsMode) {
		return TestOutput{}, fmt.Errorf("%w: tls mode %q is not one of %s",
			domain.ErrInvalid, tlsMode, strings.Join(constants.TLSModes, ", "))
	}
	port := in.Port
	if port == 0 {
		port = constants.DefaultMySQLPort
	}
	if port < 1 || port > 65535 {
		return TestOutput{}, fmt.Errorf("%w: port %d is outside 1-65535",
			domain.ErrInvalid, port)
	}

	// The draft carries no ref and no sealed secret: a probe reads only what the driver needs to
	// dial, and there is nothing here to seal.
	draft := domain.Target{
		Driver:   in.Driver,
		Host:     in.Host,
		Port:     port,
		Database: strings.TrimSpace(in.Database),
		Username: in.Username,
		TLSMode:  tlsMode,
		Options:  in.Options,
	}
	caps, err := t.Driver.Probe(ctx, draft, in.Password)
	if err != nil {
		return TestOutput{}, err
	}

	// A server that answered but will not enumerate its schemas is still a server the credentials
	// reached, which is what this call is for: the list is the courtesy, not the verdict.
	databases, _ := t.Driver.Databases(ctx, draft, in.Password)
	return TestOutput{Capabilities: caps, Databases: databases}, nil
}
