package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"noxis/internal/constants"
)

// Target is one MySQL or MariaDB server this build governs. The password appears only as
// Secret — a sealed blob the domain never opens, because sealing is an adapter's job —
// and never as a field that could reach a response by accident.
type Target struct {
	Ref       string
	Name      string
	Driver    string
	Host      string
	Port      int
	Database  string
	Username  string
	Secret    string
	TLSMode   string
	Options   map[string]string
	CreatedBy string
	// The measured engine: what the last probe saw. Empty until a probe succeeds.
	ServerVersion   string
	LastProbeAt     *time.Time
	LastProbeStatus string
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var targetName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

// Validate is the only thing standing between a typed name and the state file. Every
// rule here is one a reviewer would want to refuse before the row exists.
func (t Target) Validate() error {
	if !targetName.MatchString(t.Name) {
		return fmt.Errorf("%w: name %q must be 2 to 63 characters of a-z, 0-9, - or _", ErrInvalid, t.Name)
	}
	if !constants.IsDriver(t.Driver) {
		return fmt.Errorf("%w: driver %q is not one of %s", ErrInvalid, t.Driver, strings.Join(constants.Drivers, ", "))
	}
	if strings.TrimSpace(t.Host) == "" {
		return fmt.Errorf("%w: host is required", ErrInvalid)
	}
	if t.Port < 1 || t.Port > 65535 {
		return fmt.Errorf("%w: port %d is outside 1-65535", ErrInvalid, t.Port)
	}
	if strings.TrimSpace(t.Database) == "" {
		return fmt.Errorf("%w: database is required", ErrInvalid)
	}
	if strings.TrimSpace(t.Username) == "" {
		return fmt.Errorf("%w: username is required", ErrInvalid)
	}
	if !constants.IsTLSMode(t.TLSMode) {
		return fmt.Errorf("%w: tls mode %q is not one of %s", ErrInvalid, t.TLSMode, strings.Join(constants.TLSModes, ", "))
	}
	if t.CreatedBy == "" {
		return fmt.Errorf("%w: a target needs the caller that created it", ErrInvalid)
	}
	return nil
}

// Archived reports whether the target has been put away. An archived target is never
// removed: the journal still points at it.
func (t Target) Archived() bool { return t.ArchivedAt != nil }
