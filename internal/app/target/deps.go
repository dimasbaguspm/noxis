// Package target holds the use cases that govern targets: one file per use case, one Run
// each, and no use case calling another.
package target

import (
	"context"
	"fmt"
	"noxis/internal/common/ref"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"time"
)

// Deps is what every target use case is built from, set once in the composition root.
type Deps struct {
	Targets ports.Targets
	Journal ports.Journal
	Sealer  ports.Sealer
	Driver  ports.Driver
}

// record appends one journal line about a target. The write it describes has already
// happened, so a failed append is reported and never rolls the write back.
func (d Deps) record(ctx context.Context, subject domain.Target, kind, actor string, detail map[string]string) error {
	return d.Journal.Append(ctx, domain.JournalEntry{
		Ref:         ref.New(),
		SubjectRef:  subject.Ref,
		SubjectKind: constants.SubjectTarget,
		Kind:        kind,
		Actor:       actor,
		Detail:      detail,
		CreatedAt:   time.Now().UTC(),
	})
}

// seal turns a plaintext password into the blob the state file keeps. It refuses to store a
// password it cannot seal: an unencrypted credential in the state file is a bug, not a
// fallback.
func (d Deps) seal(targetRef, password string) (string, error) {
	if password == "" {
		return "", nil
	}
	if !d.Sealer.Enabled() {
		return "", fmt.Errorf("%w: SECRET_KEY is unset, so a password cannot be kept", domain.ErrUnsupported)
	}
	return d.Sealer.Seal(targetRef, []byte(password))
}

// openPassword opens the sealed blob for the one use case that connects. The plaintext never
// leaves this package, is never logged, and is never part of a returned value.
func (d Deps) openPassword(target domain.Target) (string, error) {
	if target.Secret == "" {
		return "", nil
	}
	plain, err := d.Sealer.Open(target.Ref, target.Secret)
	if err != nil {
		return "", fmt.Errorf("%w: the stored password cannot be opened, which usually means SECRET_KEY changed: %w",
			domain.ErrUnsupported, err)
	}
	return string(plain), nil
}

// requireRef rejects anything that is not a reference, so a path param is a 400 rather than a
// lookup that cannot match.
func requireRef(value string) (string, error) {
	if !ref.Valid(value) {
		return "", domain.NotAReference(value)
	}
	return value, nil
}
