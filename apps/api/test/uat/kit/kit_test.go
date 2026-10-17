package kit

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestSendCapturesStatusHeadersAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo", "1")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	resp, err := (&Client{BaseURL: srv.URL}).Send(Request{Method: http.MethodGet, Path: "/"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.Status != http.StatusTeapot {
		t.Errorf("status = %d, want 418", resp.Status)
	}
	if got := resp.Headers.Get("X-Echo"); got != "1" {
		t.Errorf("X-Echo = %q, want 1", got)
	}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := resp.Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !out.OK {
		t.Errorf("decoded body = %+v, want ok", out)
	}
}

func TestDefaultTokenHeaderOverrideAndClear(t *testing.T) {
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "default"}

	if _, err := c.Send(Request{Method: http.MethodGet, Path: "/"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := seen.Get("Authorization"); got != "Bearer default" {
		t.Errorf("default auth = %q, want Bearer default", got)
	}

	if _, err := c.Send(Request{Method: http.MethodGet, Path: "/", Headers: map[string]string{"Authorization": "Bearer override", "X-UAT": "yes"}}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := seen.Get("Authorization"); got != "Bearer override" {
		t.Errorf("override auth = %q, want Bearer override", got)
	}
	if got := seen.Get("X-UAT"); got != "yes" {
		t.Errorf("X-UAT = %q, want yes", got)
	}

	if _, err := c.Send(Request{Method: http.MethodGet, Path: "/", Headers: map[string]string{"Authorization": ""}}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := seen.Get("Authorization"); got != "" {
		t.Errorf("cleared auth = %q, want empty", got)
	}
}

func TestCookieRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err == nil && c.Value == "abc" {
			w.Header().Set("X-Seen", "abc")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
	}))
	defer srv.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("jar: %v", err)
	}
	c := &Client{BaseURL: srv.URL, Jar: jar}

	if _, err := c.Send(Request{Method: http.MethodGet, Path: "/"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	resp, err := c.Send(Request{Method: http.MethodGet, Path: "/"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := resp.Headers.Get("X-Seen"); got != "abc" {
		t.Errorf("second request carried no cookie (X-Seen=%q)", got)
	}
}

func TestPerRequestCookie(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("stub"); err == nil {
			got = c.Value
		}
	}))
	defer srv.Close()
	ck := &http.Cookie{Name: "stub", Value: "1"}
	if _, err := (&Client{BaseURL: srv.URL}).Send(Request{Method: http.MethodGet, Path: "/", Cookies: []*http.Cookie{ck}}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got != "1" {
		t.Errorf("per-request cookie = %q, want 1", got)
	}
}

func TestSignRunsLast(t *testing.T) {
	var auth, sig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		sig = r.Header.Get("X-Sig")
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Token: "user"}
	_, err := c.Send(Request{
		Method: http.MethodPost, Path: "/", Body: []byte("{}"),
		Sign: func(req *http.Request, body []byte) {
			req.Header.Set("Authorization", "Bearer scheduler")
			req.Header.Set("X-Sig", string(body))
		},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if auth != "Bearer scheduler" {
		t.Errorf("auth = %q, want the signer to win", auth)
	}
	if sig != "{}" {
		t.Errorf("sig = %q, want the body {}", sig)
	}
}
