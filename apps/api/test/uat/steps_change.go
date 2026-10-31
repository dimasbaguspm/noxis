package uat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cucumber/godog"

	"noxis/test/uat/kit"
)

// changeBody is the wire shape of a change (only the fields UAT asserts on).
type changeBody struct {
	ID               string      `json:"id"`
	Notes            string      `json:"notes"`
	Status           string      `json:"status"`
	UnverifiedReason string      `json:"unverifiedReason"`
	Checks           []checkBody `json:"checks"`
	EstDurationMs    int64       `json:"estDurationMs"`
	EstAffected      int64       `json:"estAffected"`
	ScratchSchema    string      `json:"scratchSchema"`
	ScheduledAt      int64       `json:"scheduledAt"`
	BatchID          string      `json:"batchId"`
}

// checkBody is one validation outcome on a change.
type checkBody struct {
	Target string `json:"target"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Error  string `json:"error"`
}

// tickBody is the /internal/callback/scheduler response shape.
type tickBody struct {
	Applied []string `json:"applied"`
	Failed  []string `json:"failed"`
}

func (w *world) changeSteps(sc *godog.ScenarioContext) {
	sc.Step(`^I submit a change with query "([^"]*)" and rollback "([^"]*)"$`, w.submitChange)
	sc.Step(`^the change is "([^"]*)"$`, w.changeStatus)
	sc.Step(`^the change is unverified because "([^"]*)"$`, w.changeUnverifiedReason)
	sc.Step("I approve the change", w.approve)
	sc.Step("I apply the change", w.apply)
	sc.Step(`^I submit a change scheduled at now with query "([^"]*)" and rollback "([^"]*)"$`, w.submitScheduledChange)
	sc.Step("a scheduler tick applies the due change", w.tickScheduler)
	sc.Step("applying is rejected", w.applyRejected)
	sc.Step("I reject the change", w.rejectChange)
	sc.Step("the change list still contains it", w.changeStillPresent)
	sc.Step("the change was not recorded", w.changeNotRecorded)
	sc.Step("the change was verified in a scratch schema", w.changeScratchSchema)
	sc.Step("I open the change", w.openChange)
	sc.Step("I schedule the change for now", w.scheduleChange)
	sc.Step("the change is scheduled", w.changeScheduled)
	sc.Step(`^the change records an estimated duration and affected count$`, w.changeEstimates)
}

func (w *world) submitChange(ctx context.Context, query, rollback string) error {
	w.change = changeBody{}
	body := map[string]string{
		"connection": w.conn, "notes": "uat change", "query": query, "rollback": rollback,
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit change: %d", w.code)
	}
	w.changeID = w.change.ID
	return nil
}

func (w *world) changeStatus(ctx context.Context, want string) error {
	if w.change.Status != want {
		return fmt.Errorf("change status %q want %q (checks: %v)", w.change.Status, want, w.change.Checks)
	}
	return nil
}

// changeUnverifiedReason checks why a change could not be proved, so the queue can group what no driver proved.
func (w *world) changeUnverifiedReason(ctx context.Context, want string) error {
	if w.change.Status != "unverified" {
		return fmt.Errorf("change status %q want \"unverified\" (checks: %v)", w.change.Status, w.change.Checks)
	}
	if w.change.UnverifiedReason != want {
		return fmt.Errorf("unverified reason %q want %q (checks: %v)", w.change.UnverifiedReason, want, w.change.Checks)
	}
	return nil
}

func (w *world) approve(ctx context.Context) error {
	return w.changeAction("/approve")
}

func (w *world) apply(ctx context.Context) error {
	return w.changeAction("/apply")
}

// submitScheduledChange is submitChange with a due schedule, so the tick has something to pick up without waiting.
func (w *world) submitScheduledChange(ctx context.Context, query, rollback string) error {
	w.change = changeBody{}
	body := map[string]any{
		"connection": w.conn, "notes": "uat change", "query": query, "rollback": rollback,
		"scheduledAt": time.Now().Unix(),
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit scheduled change: %d", w.code)
	}
	w.changeID = w.change.ID
	return nil
}

func (w *world) tickScheduler(ctx context.Context) error {
	if err := w.tick(ctx); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("scheduler tick: status %d body %s", w.code, w.body)
	}
	return nil
}

// tick drives POST /internal/callback/scheduler and, on success, reloads the change under test so the next status assertion sees the result.
func (w *world) tick(ctx context.Context) error {
	if err := w.send(kit.Request{
		Method: http.MethodPost,
		Path:   "/internal/callback/scheduler",
		Sign:   signScheduler,
	}); err != nil {
		return err
	}
	w.ticked = tickBody{}
	_ = w.resp.Decode(&w.ticked)
	if w.code != http.StatusOK || w.changeID == "" {
		return nil
	}
	w.change = changeBody{}
	return w.api(http.MethodGet, w.enginePath("/changes/"+w.changeID), nil, &w.change)
}

// changeAction POSTs to a change sub-resource and records the new state.
func (w *world) changeAction(action string) error {
	w.change = changeBody{}
	if err := w.api(http.MethodPost, w.enginePath("/changes/"+w.changeID+action), map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("%s: %d", action, w.code)
	}
	return nil
}

func (w *world) applyRejected(ctx context.Context) error {
	w.change = changeBody{}
	if err := w.api(http.MethodPost, w.enginePath("/changes/"+w.changeID+"/apply"), map[string]string{}, &w.change); err != nil {
		return err
	}
	if w.code == http.StatusOK {
		return errors.New("apply must require approval first")
	}
	return nil
}

func (w *world) rejectChange(ctx context.Context) error {
	return w.changeAction("/reject")
}

// changeStillPresent proves rejection keeps the artifact (no hard delete).
func (w *world) changeStillPresent(ctx context.Context) error {
	var body struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, w.enginePath("/changes"), nil, &body); err != nil {
		return err
	}
	for i := range body.Changes {
		if body.Changes[i].ID == w.changeID {
			return nil
		}
	}
	return fmt.Errorf("change %s disappeared from the list", w.changeID)
}

// changeNotRecorded proves rejected proposals never enter the store: the submit response carries no id and the list does not grow.
func (w *world) changeNotRecorded(ctx context.Context) error {
	if w.changeID != "" {
		return fmt.Errorf("failed change kept an id: %s", w.changeID)
	}
	var body struct {
		Changes []changeBody `json:"changes"`
	}
	if err := w.api(http.MethodGet, w.enginePath("/changes"), nil, &body); err != nil {
		return err
	}
	for i := range body.Changes {
		if body.Changes[i].Notes == "uat change" && body.Changes[i].Status == "failed" {
			return fmt.Errorf("failed change was recorded: %+v", body.Changes[i])
		}
	}
	return nil
}

// openChange reads the artifact back by id, proving the detail route.
func (w *world) openChange(ctx context.Context) error {
	w.change = changeBody{}
	if err := w.api(http.MethodGet, w.enginePath("/changes/"+w.changeID), nil, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK || w.change.ID != w.changeID {
		return fmt.Errorf("open change: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) scheduleChange(ctx context.Context) error {
	body := map[string]int64{"scheduledAt": time.Now().Add(time.Minute).Unix()}
	if err := w.api(http.MethodPost, w.enginePath("/changes/"+w.changeID+"/schedule"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("schedule: %d %s", w.code, w.body)
	}
	return nil
}

func (w *world) changeScheduled(ctx context.Context) error {
	if w.change.ScheduledAt == 0 {
		return fmt.Errorf("change is not scheduled: %+v", w.change)
	}
	return nil
}

func (w *world) changeScratchSchema(ctx context.Context) error {
	if !strings.HasPrefix(w.change.ScratchSchema, "noxis_verify_") {
		return fmt.Errorf("change was not verified in a scratch schema: %q", w.change.ScratchSchema)
	}
	return nil
}

func (w *world) changeEstimates(ctx context.Context) error {
	if w.change.EstDurationMs < 0 || w.change.EstAffected < 0 {
		return fmt.Errorf("negative estimates: %+v", w.change)
	}
	if w.change.Status != "applied" {
		return fmt.Errorf("estimates are recorded on validation; status=%s", w.change.Status)
	}
	return nil
}
