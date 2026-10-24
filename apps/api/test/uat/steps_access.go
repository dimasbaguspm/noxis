package uat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// accessBody is one permission row.
type accessBody struct {
	Connection string `json:"connection"`
	Member     string `json:"member"`
	MemberType string `json:"memberType"`
	Level      string `json:"level"`
}

// requestBody is one access request.
type requestBody struct {
	ID         string `json:"id"`
	Subject    string `json:"subject"`
	Connection string `json:"connection"`
	Level      string `json:"level"`
	Status     string `json:"status"`
	DecidedBy  string `json:"decidedBy"`
}

func (w *world) accessSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I grant "([^"]*)" the "([^"]*)" level on the current connection$`, w.grantLevel)
	sc.Step(`^the permission list shows "([^"]*)" at "([^"]*)"$`, w.permissionListShows)
	sc.Step(`^I revoke "([^"]*)" from the current connection$`, w.revokeLevel)
	sc.Step("the permission list is empty", w.permissionListEmpty)
	sc.Step(`^I request the "([^"]*)" level on the current connection$`, w.requestAccess)
	sc.Step("the access request list contains it", w.accessRequestListed)
	sc.Step(`^I (approve|reject) the access request$`, w.decideAccessRequest)
	sc.Step(`^the access request is "([^"]*)"$`, w.accessRequestStatus)
	sc.Step("the request is refused as self-decided", w.accessRequestRefused)
	sc.Step("the access matrix lists the current connection", w.accessMatrixLists)
	sc.Step("the user directory contains the root account", w.userDirectoryContainsRoot)
	sc.Step("the root profile carries its email", w.rootProfileCarriesEmail)
	sc.Step("changing my own account is refused", w.selfChangeRefused)
}

func (w *world) grantLevel(ctx context.Context, member, level string) error {
	body := accessBody{Member: member, MemberType: "user", Level: level}
	if err := w.api(http.MethodPut, "/external/v1/engine/sql/connections/"+w.conn+"/permissions", body, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("grant: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) permissionListShows(ctx context.Context, member, level string) error {
	var out struct {
		Permissions []accessBody `json:"permissions"`
	}
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/connections/"+w.conn+"/permissions", nil, &out); err != nil {
		return err
	}
	for _, p := range out.Permissions {
		if p.Member == member && p.Level == level {
			return nil
		}
	}
	return fmt.Errorf("no grant for %s at %s in %+v", member, level, out.Permissions)
}

func (w *world) revokeLevel(ctx context.Context, member string) error {
	path := "/external/v1/engine/sql/connections/" + w.conn + "/permissions?member=" + member
	if err := w.api(http.MethodDelete, path, nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK && w.code != http.StatusNoContent {
		return fmt.Errorf("revoke: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) permissionListEmpty(ctx context.Context) error {
	var out struct {
		Permissions []accessBody `json:"permissions"`
	}
	if err := w.api(http.MethodGet, "/external/v1/engine/sql/connections/"+w.conn+"/permissions", nil, &out); err != nil {
		return err
	}
	if len(out.Permissions) != 0 {
		return fmt.Errorf("permissions after revoke = %+v", out.Permissions)
	}
	return nil
}

func (w *world) requestAccess(ctx context.Context, level string) error {
	body := map[string]string{"connection": w.conn, "level": level, "reason": "uat request"}
	var out requestBody
	if err := w.api(http.MethodPost, "/external/v1/access-requests", body, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("request access: %d %s", w.code, w.body)
	}
	w.accessRequest = out
	return nil
}

func (w *world) accessRequestListed(ctx context.Context) error {
	var out struct {
		Requests []requestBody `json:"requests"`
	}
	if err := w.api(http.MethodGet, "/external/v1/access-requests", nil, &out); err != nil {
		return err
	}
	for _, r := range out.Requests {
		if r.ID == w.accessRequest.ID && r.Connection == w.conn {
			return nil
		}
	}
	return fmt.Errorf("request %s not in %+v", w.accessRequest.ID, out.Requests)
}

func (w *world) decideAccessRequest(ctx context.Context, decision string) error {
	body := map[string]string{"action": decision, "note": "uat " + decision}
	path := "/external/v1/access-requests/" + w.accessRequest.ID + "/action"
	if err := w.api(http.MethodPost, path, body, &w.accessRequest); err != nil {
		return err
	}
	if w.code == http.StatusForbidden {
		w.accessRefused = true
		return nil
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("%s request: %d %s", decision, w.code, w.body)
	}
	w.accessRefused = false
	return nil
}

func (w *world) accessRequestRefused(ctx context.Context) error {
	if !w.accessRefused {
		return fmt.Errorf("deciding your own request was not refused")
	}
	return nil
}

func (w *world) accessRequestStatus(ctx context.Context, want string) error {
	if w.accessRequest.Status != want {
		return fmt.Errorf("request status %q want %q", w.accessRequest.Status, want)
	}
	return nil
}

func (w *world) accessMatrixLists(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/external/v1/access", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, `"`+w.conn+`"`) {
		return fmt.Errorf("access matrix: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) userDirectoryContainsRoot(ctx context.Context) error {
	if err := w.api(http.MethodGet, "/external/v1/users", nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, adminEmail) {
		return fmt.Errorf("user directory: %d %s", w.code, w.body)
	}
	if strings.Contains(w.body, `"password"`) {
		return fmt.Errorf("user directory carries a password: %s", w.body)
	}
	return nil
}

func (w *world) rootProfileCarriesEmail(ctx context.Context) error {
	subject := "local:" + adminEmail
	if err := w.api(http.MethodGet, "/external/v1/users/"+subject, nil, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK || !strings.Contains(w.body, adminEmail) {
		return fmt.Errorf("user profile: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) selfChangeRefused(ctx context.Context) error {
	subject := "local:" + adminEmail
	body := map[string]string{"role": "admin"}
	if err := w.api(http.MethodPatch, "/external/v1/users/"+subject, body, nil); err != nil {
		return err
	}
	if w.code != http.StatusForbidden {
		return fmt.Errorf("self change: %d %s, want 403", w.code, w.body)
	}
	return nil
}
