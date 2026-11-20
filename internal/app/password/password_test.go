package password_test

import (
	"errors"
	"noxis/internal/app/password"
	"noxis/internal/domain"
	"strings"
	"testing"
)

type sealer struct{ err error }

func (sealer) Enabled() bool { return true }

func (sealer) Seal(_ string, plain []byte) (string, error) { return "sealed:" + string(plain), nil }

func (s sealer) Open(_ string, sealed string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []byte(strings.TrimPrefix(sealed, "sealed:")), nil
}

// A target with no password is a target that connects without one, not an error: refusing here would
// make a passwordless account unusable.
func TestOpenWithoutASecret(t *testing.T) {
	plain, err := password.Open(sealer{}, domain.Target{Ref: "t"})
	if err != nil || plain != "" {
		t.Fatalf("Open = %q, %v", plain, err)
	}
}

func TestOpenReturnsThePlaintextOnce(t *testing.T) {
	plain, err := password.Open(sealer{}, domain.Target{Ref: "t", Secret: "sealed:hunter2"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if plain != "hunter2" {
		t.Fatalf("Open = %q", plain)
	}
}

// A blob the current key cannot open is a key problem, and the refusal says so without quoting the
// cipher's own error into the API's body.
func TestOpenRefusesAnUnopenableSecret(t *testing.T) {
	_, err := password.Open(sealer{err: errors.New("cipher: message authentication failed")},
		domain.Target{Ref: "t", Secret: "v1:not-really"})
	if !errors.Is(err, domain.ErrUnsupported) {
		t.Fatalf("Open: %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "SECRET_KEY") {
		t.Fatalf("Open: %q does not name the cause a human can act on", err)
	}
}
