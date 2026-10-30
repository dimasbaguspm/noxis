package entity

import (
	"errors"
	"testing"
	"time"

	"noxis/internal/constants"
)

func TestApproveAcceptsOnlyValidatedStates(t *testing.T) {
	for _, status := range []string{constants.ChangeValidated, constants.ChangeUnverified} {
		change := Change{ID: "change", Status: status}
		if err := change.Approve("reviewer", 10); err != nil {
			t.Fatalf("approve %s: %v", status, err)
		}
		if change.Status != constants.ChangeApproved || change.ApprovedAt != 10 || change.ApprovedBy != "reviewer" {
			t.Fatalf("approved change = %+v", change)
		}
	}
	for _, status := range []string{constants.ChangeFailed, constants.ChangeApplied, constants.ChangeRejected} {
		change := Change{Status: status}
		if err := change.Approve("reviewer", 10); err == nil {
			t.Fatalf("approve %s succeeded", status)
		}
	}
}

func TestRejectRefusesARunningOrAppliedChange(t *testing.T) {
	applying := Change{Status: constants.ChangeApplying}
	if err := applying.Reject("reviewer", 10); !errors.Is(err, ErrChangeConflict) {
		t.Fatalf("reject applying error = %v, want the claim sentinel", err)
	}
	applied := Change{Status: constants.ChangeApplied}
	if err := applied.Reject("reviewer", 10); err == nil {
		t.Fatal("reject applied change succeeded")
	}
	validated := Change{Status: constants.ChangeValidated}
	if err := validated.Reject("reviewer", 10); err != nil || validated.RejectedBy != "reviewer" {
		t.Fatalf("reject actor=%q error=%v", validated.RejectedBy, err)
	}
}

func TestScheduleApplyableAndDueBoundaries(t *testing.T) {
	now := time.Now().Unix()
	change := Change{ID: "change", Status: constants.ChangeApproved}
	future := now + 60
	if err := change.Schedule(future, "reviewer", now); err != nil {
		t.Fatalf("schedule future: %v", err)
	}
	if change.ScheduledBy != "reviewer" {
		t.Fatalf("scheduled by=%q, want reviewer", change.ScheduledBy)
	}
	if err := change.Applyable(future - 1); err == nil {
		t.Fatal("future schedule was applyable")
	}
	if err := change.Applyable(future); err != nil {
		t.Fatalf("due schedule was not applyable: %v", err)
	}
	if !change.Due(future) || change.Due(future - 1) {
		t.Fatalf("due boundary at %d is wrong", future)
	}
	if err := change.Schedule(0, "reviewer", now); err != nil {
		t.Fatalf("clear schedule: %v", err)
	}
	if change.Due(future) {
		t.Fatal("immediate change was due")
	}
	change.Status = constants.ChangeApplying
	if err := change.Applyable(future); !errors.Is(err, ErrChangeConflict) {
		t.Fatalf("applying state error = %v, want the claim sentinel", err)
	}
}

func TestScheduleRefusesAPastSlot(t *testing.T) {
	now := time.Now().Unix()
	change := Change{Status: constants.ChangeApproved}
	if err := change.Schedule(now-1, "reviewer", now); err == nil {
		t.Fatal("a slot in the past was accepted and would never be due")
	}
}

func TestMarkAppliedAndMarkFailedRecordTheOutcome(t *testing.T) {
	applied := Change{Status: constants.ChangeApplying}
	applied.MarkApplied(Result{Affected: 3, DurationMs: 12, Message: "ok"}, 99)
	if applied.Status != constants.ChangeApplied || applied.AppliedAffected != 3 || applied.AppliedAt != 99 {
		t.Fatalf("applied change = %+v", applied)
	}
	if len(applied.Checks) != 1 || !applied.Checks[0].OK {
		t.Fatalf("applied checks = %+v", applied.Checks)
	}
	failed := Change{Status: constants.ChangeApplying}
	failed.MarkFailed(errors.New("engine said no"))
	if failed.Status != constants.ChangeFailed || failed.Checks[0].Error != "engine said no" {
		t.Fatalf("failed change = %+v", failed)
	}
}
