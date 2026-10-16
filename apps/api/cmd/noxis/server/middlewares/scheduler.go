package middlewares

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The scheduler trigger is authenticated with a shared pair: the API key
// identifies the caller (bearer), the secret signs the request. Both are
// provisioned on noxis and on the external scheduler.
const (
	SchedulerTimestampHeader = "X-Noxis-Timestamp"
	SchedulerSignatureHeader = "X-Noxis-Signature"

	schedulerMaxSkew = 5 * time.Minute
	schedulerMaxBody = 1 << 20 // 1 MiB; the trigger carries no payload today
)

// SchedulerSignature is the hex HMAC-SHA256 over "<timestamp>.<body>", so a
// replay with a different body (or a replayed timestamp) fails.
func SchedulerSignature(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// WithSchedulerAuth gates the scheduler trigger on the shared api key + secret.
// An unset pair denies every call: the trigger is never open.
func WithSchedulerAuth(apiKey, secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" || secret == "" {
			writeUnauthorized(w, "scheduler trigger is not configured")
			return
		}
		presented := bearerToken(r)
		if presented == "" || !hmac.Equal([]byte(presented), []byte(apiKey)) {
			writeUnauthorized(w, "invalid scheduler api key")
			return
		}
		timestamp := r.Header.Get(SchedulerTimestampHeader)
		sent, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			writeUnauthorized(w, "missing or invalid scheduler timestamp")
			return
		}
		if skew := time.Since(time.Unix(sent, 0)); skew > schedulerMaxSkew || skew < -schedulerMaxSkew {
			writeUnauthorized(w, "scheduler timestamp outside the allowed window")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, schedulerMaxBody+1))
		if err != nil || len(body) > schedulerMaxBody {
			writeUnauthorized(w, "invalid scheduler body")
			return
		}
		presented = strings.ToLower(r.Header.Get(SchedulerSignatureHeader))
		if !hmac.Equal([]byte(presented), []byte(SchedulerSignature(secret, timestamp, body))) {
			writeUnauthorized(w, "invalid scheduler signature")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}
