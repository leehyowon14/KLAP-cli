package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func TestExternalOpenDefersToInjectedContextAndPreservesError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	want := errors.New("launch failed")
	calls := 0
	m := model{ctx: ctx, active: screenAssignmentDetail, opener: func(got context.Context, target, kind string) error {
		calls++
		if got != ctx || target != "https://example.invalid/detail" || kind != "KLAS URL" {
			t.Fatal("opener arguments changed")
		}
		return want
	}}
	m.assignments.assignmentDetail = app.AssignmentDetailResult{DetailURL: "https://example.invalid/detail"}
	command := m.openCurrentKlasURL()
	if command == nil || calls != 0 {
		t.Fatal("opening was not deferred")
	}
	message := command().(statusMsg)
	if calls != 1 || !errors.Is(message.err, want) {
		t.Fatalf("calls=%d message=%+v", calls, message)
	}
	m.assignments.assignmentDetail.DetailURL = ""
	if command := m.openCurrentKlasURL(); command != nil {
		t.Fatal("empty URL launched")
	}
	if err := (model{}).openExternalURL("url"); err == nil {
		t.Fatal("missing opener accepted")
	}
}

func TestRunRejectsMissingOpenerBeforeStartingUI(t *testing.T) {
	if err := Run(context.Background(), &app.Service{}, nil); err == nil {
		t.Fatal("missing opener accepted")
	}
}
