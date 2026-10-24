package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestWithSchedulerAuth(t *testing.T) {
	const key, secret = "sched-key", "sched-secret"
	now := strconv.FormatInt(time.Now().Unix(), 10)
	stale := strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)

	sign := func(ts, body string) string { return SchedulerSignature(secret, ts, []byte(body)) }

	tests := []struct {
		name       string
		key        string
		secret     string
		auth       string
		timestamp  string
		signature  string
		wantStatus int
	}{
		{"valid", key, secret, "Bearer " + key, now, sign(now, ""), http.StatusOK},
		{"unset pair denies", "", "", "Bearer " + key, now, sign(now, ""), http.StatusUnauthorized},
		{"missing api key", key, secret, "", now, sign(now, ""), http.StatusUnauthorized},
		{"wrong api key", key, secret, "Bearer nope", now, sign(now, ""), http.StatusUnauthorized},
		{"missing timestamp", key, secret, "Bearer " + key, "", sign(now, ""), http.StatusUnauthorized},
		{"stale timestamp", key, secret, "Bearer " + key, stale, sign(stale, ""), http.StatusUnauthorized},
		{"wrong secret", key, "other", "Bearer " + key, now, sign(now, ""), http.StatusUnauthorized},
		{"empty signature", key, secret, "Bearer " + key, now, "", http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called bool
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			handler := WithSchedulerAuth(tc.key, tc.secret, next)

			req := httptest.NewRequest(http.MethodPost, "/internal/callback/scheduler", http.NoBody)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			if tc.timestamp != "" {
				req.Header.Set(SchedulerTimestampHeader, tc.timestamp)
			}
			if tc.signature != "" {
				req.Header.Set(SchedulerSignatureHeader, tc.signature)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d", rec.Code, tc.wantStatus)
			}
			if wantCalled := tc.wantStatus == http.StatusOK; called != wantCalled {
				t.Fatalf("next called=%v want=%v", called, wantCalled)
			}
		})
	}
}

// TestSchedulerSignatureBodyBinding proves a valid timestamp cannot be reused with a different body.
func TestSchedulerSignatureBodyBinding(t *testing.T) {
	const secret = "s"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	if SchedulerSignature(secret, ts, []byte("a")) == SchedulerSignature(secret, ts, []byte("b")) {
		t.Fatal("signature must bind the body")
	}
}
