package httpProcessor

import (
	"errors"
	"testing"
)

func TestPageSnapshotDoesNotMutateVisibleCursorHistory(t *testing.T) {
	p := Page{cursors: map[int]string{100: "original"}}
	pending := p.Snapshot()
	pending.cursors[100] = "late-response"
	if p.cursors[100] != "original" {
		t.Fatal("pending request changed visible cursor")
	}
}

func TestViewLoadScopeRejectsLateResponseAndCancelsOnLeave(t *testing.T) {
	var scope LoadScope
	first := scope.Start()
	second := scope.Start()
	if first.Context.Err() == nil {
		t.Fatal("older request not canceled")
	}
	if first.Finish(errors.New("late failure")) {
		t.Fatal("stale response accepted")
	}
	if loading, err := scope.State(); !loading || err != nil {
		t.Fatal("old response changed current state")
	}
	scope.Cancel()
	if second.Context.Err() == nil || second.Finish(nil) {
		t.Fatal("hidden view accepted response")
	}
	third := scope.Start()
	failure := errors.New("retryable")
	if !third.Finish(failure) {
		t.Fatal("current load rejected")
	}
	if loading, err := scope.State(); loading || !errors.Is(err, failure) {
		t.Fatal("failure state lost")
	}
	retry := scope.Start()
	if !retry.Finish(nil) {
		t.Fatal("retry rejected")
	}
	if loading, err := scope.State(); loading || err != nil {
		t.Fatal("retry did not clear error")
	}
}
