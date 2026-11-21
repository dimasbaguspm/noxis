package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"noxis/internal/common/release"
	"strings"
	"testing"
	"time"
)

// A checksum that is not in the published list is a refusal, not a warning: an upgrade that cannot verify
// what it downloaded must not run it.
func TestChecksumReadsTheLineForThisPlatform(t *testing.T) {
	const hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The shape `sha256sum` writes: the hash, two spaces, the file. The other platform is there too, so
		// the reader has to pick the right line.
		_, _ = w.Write([]byte(hash + "  noxis-darwin-amd64\n" + hash + "  " + assetName() + "\n"))
	}))
	defer server.Close()

	got, err := checksum(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("checksum: %v", err)
	}
	if got != hash {
		t.Fatalf("checksum = %q, want %q", got, hash)
	}
}

func TestChecksumRefusesAPlatformItDoesNotList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  noxis-darwin-amd64\n"))
	}))
	defer server.Close()

	if _, err := checksum(t.Context(), server.URL); err == nil {
		t.Fatal("a platform the list does not name was accepted")
	}
}

// download answers what it wrote and what it hashed in one pass, which is what makes "verify what you got"
// possible without reading the file twice.
func TestDownloadAnswersTheHashOfWhatItWrote(t *testing.T) {
	body := "the published binary"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	var got bytes.Buffer
	written, hash, err := download(t.Context(), server.URL, &got)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if got.String() != body || written != int64(len(body)) {
		t.Fatalf("download wrote %q (%d bytes)", got.String(), written)
	}
	sum := sha256.Sum256([]byte(body))
	if hash != hex.EncodeToString(sum[:]) {
		t.Fatalf("hash = %q", hash)
	}
}

// The asset a platform asks for is the name the publish job uses, in the one tag that is published.
func TestAssetURLNamesThePublishedBinary(t *testing.T) {
	url := assetURL()
	if !strings.HasSuffix(url, "/"+assetName()) {
		t.Fatalf("assetURL = %q, want it to end with %q", url, assetName())
	}
	if !strings.HasPrefix(assetName(), "noxis-") {
		t.Fatalf("assetName = %q", assetName())
	}
	published := release.Info{Version: "nightly", Commit: "9f3c2e8"}
	if got := publishedString(&published); got != "nightly+9f3c2e8" {
		t.Fatalf("publishedString = %q", got)
	}
}

// A transfer that stops sending bytes is refused, and says so: a bare "context deadline exceeded" after five
// minutes told the operator nothing about which step gave up.
func TestAStalledDownloadIsRefused(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = writer.Close() }()

	body := newGuardedBody(reader, 20*time.Millisecond)
	defer func() { _ = body.Close() }()

	_, err := io.Copy(io.Discard, body)
	if !errors.Is(err, errStalled) {
		t.Fatalf("err = %v, want errStalled", err)
	}
	if !strings.Contains(err.Error(), "nothing for") {
		t.Fatalf("err = %q, want it to name the idle window", err)
	}
}

// A slow transfer is not a stalled one: the deadline moves with the bytes, so a link that dribbles finishes
// however long it takes. Twenty reads with a pause between them outlast the idle window and still arrive.
func TestASlowDownloadIsNotInterrupted(t *testing.T) {
	const chunks = 20
	reader, writer := io.Pipe()
	body := newGuardedBody(reader, 40*time.Millisecond)
	defer func() { _ = body.Close() }()

	go func() {
		for range chunks {
			time.Sleep(10 * time.Millisecond)
			_, _ = writer.Write([]byte("x"))
		}
		_ = writer.Close()
	}()

	written, err := io.Copy(io.Discard, body)
	if err != nil {
		t.Fatalf("a slow transfer was refused: %v", err)
	}
	if written != chunks {
		t.Fatalf("read %d bytes, want %d", written, chunks)
	}
}
