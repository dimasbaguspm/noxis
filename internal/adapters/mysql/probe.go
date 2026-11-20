package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
	"noxis/internal/ports"
)

// driverName is the Go driver's registration name, not an engine name: one driver serves
// both engines.
const driverName = "mysql"

// DB is the engine adapter.
type DB struct {
	probeTimeout time.Duration
}

var _ ports.Driver = (*DB)(nil)

// New builds the adapter with the probe ceiling the composition root read from the
// environment.
func New(probeTimeout time.Duration) *DB {
	if probeTimeout <= 0 {
		probeTimeout = 5 * time.Second
	}
	return &DB{probeTimeout: probeTimeout}
}

// probeQuery reads only facts about the server itself. It is the single query this adapter
// runs, and it touches no table.
const probeQuery = "SELECT @@version, @@version_comment, @@max_allowed_packet"

// Probe connects, asks the server who it is, and disconnects. The measured flavour wins over
// the declared driver: `domain.CapabilitiesFor` says so in its notes when the two disagree.
func (d *DB) Probe(ctx context.Context, target domain.Target, password string) (caps domain.Capabilities, err error) {
	ctx, end := telemetry.Span(ctx, "mysql.DB.Probe")
	defer func() { end(err) }()

	pool, err := sql.Open(driverName, dsnOf(target, password))
	if err != nil {
		return domain.Capabilities{}, connectError(err, password)
	}
	defer func() { _ = pool.Close() }()
	pool.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(ctx, d.probeTimeout)
	defer cancel()

	var (
		version   string
		comment   string
		maxPacket int64
	)
	if err := pool.QueryRowContext(ctx, probeQuery).Scan(&version, &comment, &maxPacket); err != nil {
		return domain.Capabilities{}, connectError(err, password)
	}

	server := domain.ParseServerVersion(version, comment)
	caps = domain.CapabilitiesFor(target.Driver, server)
	if maxPacket <= 0 {
		caps.Notes = append(caps.Notes, "the server did not report max_allowed_packet")
	}
	return caps, nil
}

// connectError keeps a refusal quotable: it says what the driver said, with the password cut out
// of the message in case the driver echoed the DSN back.
func connectError(err error, password string) error {
	return fmt.Errorf("%w: %s", domain.ErrUnsupported, scrub(err, password))
}
