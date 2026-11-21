package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"noxis/internal/client"
	"noxis/internal/constants"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// An agent reads a code, not a sentence: the failure has to arrive in the vocabulary the API already
// answered with, so the caller never matches prose to learn what went wrong.
func TestReportCarriesTheAPIsCode(t *testing.T) {
	var out bytes.Buffer
	Report(&out, &client.Error{Status: http.StatusNotFound, Code: constants.CodeNotFound, Message: "no such instance"}, true)

	var decoded struct {
		Error Failure `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("the failure is not json: %v (%q)", err, out.String())
	}
	if decoded.Error.Code != constants.CodeNotFound || decoded.Error.Status != http.StatusNotFound {
		t.Fatalf("failure = %+v, want the api's code and status", decoded.Error)
	}
	if decoded.Error.Message != "no such instance" {
		t.Fatalf("message = %q, want the api's own", decoded.Error.Message)
	}
}

// A command that never reached the server has no status to report and must not invent one.
func TestReportNamesTheClientForALocalFailure(t *testing.T) {
	var out bytes.Buffer
	Report(&out, errors.New("give the change with --sql or --file"), true)

	var decoded struct {
		Error Failure `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("the failure is not json: %v (%q)", err, out.String())
	}
	if decoded.Error.Code != constants.CodeClient || decoded.Error.Status != 0 {
		t.Fatalf("failure = %+v, want the client code and no status", decoded.Error)
	}
}

// Without --json the reader is a person, and the line they see is the one they have always seen.
func TestReportKeepsTheHumanLine(t *testing.T) {
	var out bytes.Buffer
	Report(&out, errors.New("boom"), false)
	if got := out.String(); got != "error: boom\n" {
		t.Fatalf("output = %q", got)
	}
}

// proposalServer answers the statuses in order and counts the asks, so a test can assert both the
// ending and the fact that the wait stopped asking when it got there.
func proposalServer(t *testing.T, statuses []string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var asks atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		index := int(asks.Add(1)) - 1
		if index >= len(statuses) {
			index = len(statuses) - 1
		}
		w.Header().Set("Content-Type", constants.MediaTypeJSON)
		_, _ = fmt.Fprintf(w, `{"data":{"ref":"01j0abcdefghjkmnpqrstvwxyz","status":%q}}`, statuses[index])
	}))
	t.Cleanup(server.Close)
	return server, &asks
}

func TestWaitForStopsOnTheFirstSettledStatus(t *testing.T) {
	server, asks := proposalServer(t, []string{"draft", "verified", "applied", "applied"})
	api, err := client.New(server.URL, "token")
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	proposal, err := waitFor(context.Background(), api, "01j0abcdefghjkmnpqrstvwxyz",
		settledStatuses, time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if proposal.Status != "applied" {
		t.Fatalf("status = %q, want applied", proposal.Status)
	}
	if got := asks.Load(); got != 3 {
		t.Fatalf("asked %d times, want 3: the wait must stop asking once it settled", got)
	}
}

// A wait that never ends is worse than one that fails: the deadline is what makes the exit code mean
// something.
func TestWaitForGivesUpAtTheDeadline(t *testing.T) {
	server, _ := proposalServer(t, []string{"applying"})
	api, err := client.New(server.URL, "token")
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	_, err = waitFor(context.Background(), api, "01j0abcdefghjkmnpqrstvwxyz",
		settledStatuses, time.Millisecond, 40*time.Millisecond)
	if err == nil {
		t.Fatal("a wait past its deadline returned no error")
	}
	if !strings.Contains(err.Error(), "was still") {
		t.Fatalf("error = %q, want the status it was stuck in", err)
	}
}

// --all is one loop, so it is one test: every page appended, and the cursor of the last page is the
// one the caller would have paged by hand.
func TestWalkAllReadsEveryPage(t *testing.T) {
	pages := map[string]client.Page[int]{
		"":   {Items: []int{1, 2}, NextCursor: "c1"},
		"c1": {Items: []int{3}, NextCursor: "c2"},
		"c2": {Items: []int{4}, NextCursor: ""},
	}
	page, err := walkAll(pages[""], func(cursor string) (client.Page[int], error) {
		next, ok := pages[cursor]
		if !ok {
			return client.Page[int]{}, fmt.Errorf("unexpected cursor %q", cursor)
		}
		return next, nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if got := fmt.Sprint(page.Items); got != "[1 2 3 4]" {
		t.Fatalf("items = %s", got)
	}
	if page.NextCursor != "" {
		t.Fatalf("next = %q, want the end of the list", page.NextCursor)
	}
}

// A server that keeps answering a cursor must not hold the command open forever.
func TestWalkAllRefusesARunawayCursor(t *testing.T) {
	_, err := walkAll(client.Page[int]{Items: []int{1}, NextCursor: "same"},
		func(string) (client.Page[int], error) {
			return client.Page[int]{Items: []int{1}, NextCursor: "same"}, nil
		})
	if err == nil {
		t.Fatal("a cursor that points at itself walked forever")
	}
	if !strings.Contains(err.Error(), "stopped after") {
		t.Fatalf("error = %q, want the page ceiling", err)
	}
}

// A probe needs the password it is probing: the stored one is sealed and never read back, so a probe
// without one would report a failure about the tool rather than about the credential.
func TestTargetSetProbeNeedsAPassword(t *testing.T) {
	cmd := newTargetSet(&Session{APIURL: "http://127.0.0.1:1", Token: "token"})
	cmd.SetArgs([]string{"01j0abcdefghjkmnpqrstvwxyz", "--username", "other", "--probe"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--probe needs --password") {
		t.Fatalf("error = %v, want the password requirement", err)
	}
}

// Naming no field is not an empty update: it is a command that cannot know what was asked for.
func TestTargetSetNeedsAField(t *testing.T) {
	cmd := newTargetSet(&Session{APIURL: "http://127.0.0.1:1", Token: "token"})
	cmd.SetArgs([]string{"01j0abcdefghjkmnpqrstvwxyz"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "name a field to change") {
		t.Fatalf("error = %v, want the field requirement", err)
	}
}
