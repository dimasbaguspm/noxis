// Package password opens a sealed credential for the use cases that must connect. It exists so
// the connect and read paths agree on what a refusal means, and so the plaintext lives in one
// place: it is returned to the caller that needs it and never stored, returned upward, or logged.
package password

import (
	"fmt"

	"noxis/internal/domain"
	"noxis/internal/ports"
)

// Open returns the plaintext password for a target. A row sealed with another key is a key
// problem, and the refusal names it.
func Open(sealer ports.Sealer, target domain.Target) (string, error) {
	if target.Secret == "" {
		return "", nil
	}
	plain, err := sealer.Open(target.Ref, target.Secret)
	if err != nil {
		return "", fmt.Errorf("%w: the stored password cannot be opened, which usually means SECRET_KEY changed: %s",
			domain.ErrUnsupported, err)
	}
	return string(plain), nil
}
