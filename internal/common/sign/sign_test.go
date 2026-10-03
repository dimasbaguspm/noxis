package sign

import "testing"

func TestSignVerify(t *testing.T) {
	secret := []byte("s3cret")
	data := []byte("GET /v1/status")
	sig := Sign(secret, data)

	if !Verify(secret, data, sig) {
		t.Fatal("genuine signature must verify")
	}
	if Verify(secret, []byte("GET /v1/nodes"), sig) {
		t.Fatal("tampered data must not verify")
	}
	if Verify([]byte("other"), data, sig) {
		t.Fatal("wrong secret must not verify")
	}
}
