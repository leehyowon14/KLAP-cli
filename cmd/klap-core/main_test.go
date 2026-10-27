package main

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

type fakeAttender struct {
	calls  []string
	cancel context.CancelFunc
}

func (f *fakeAttender) AttendLecture(ctx context.Context, id string, o app.LectureAttendOptions) (app.LectureAttendResult, error) {
	f.calls = append(f.calls, id)
	if o.User.StudentID != "selected" {
		panic("account must be pinned")
	}
	if !o.RequireEligible {
		panic("eligibility must be checked")
	}
	if f.cancel != nil {
		f.cancel()
		return app.LectureAttendResult{}, ctx.Err()
	}
	if id == "fail" {
		return app.LectureAttendResult{}, errors.New("failure")
	}
	return app.LectureAttendResult{Progress: app.LectureProgress{Completed: id != "incomplete"}}, nil
}
func TestAttendanceQueue(t *testing.T) {
	f := &fakeAttender{}
	kinds := []string{}
	var summary map[string]any
	err := runAttendance(context.Background(), f, []string{"", "ok", "ok", "fail", "incomplete", "next"}, app.UserOption{StudentID: "selected"}, func(k string, v any) {
		kinds = append(kinds, k)
		if k == "done" {
			summary = v.(map[string]any)
		}
	})
	if err != nil || len(f.calls) != 4 {
		t.Fatalf("err=%v calls=%v", err, f.calls)
	}
	if summary["success"] != 2 || summary["failed"] != 2 || summary["total"] != 4 {
		t.Fatal(summary)
	}
	completions := 0
	for _, k := range kinds {
		if k == "completed" {
			completions++
		}
	}
	if completions != 2 {
		t.Fatalf("unconfirmed completion emitted: %v", kinds)
	}
}
func TestCancelNeverStartsNextLecture(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeAttender{cancel: cancel}
	err := runAttendance(ctx, f, []string{"one", "two"}, app.UserOption{StudentID: "selected"}, func(k string, _ any) {
		if k == "done" || k == "completed" {
			t.Fatalf("incorrect event: %s", k)
		}
	})
	if !errors.Is(err, context.Canceled) || len(f.calls) != 1 {
		t.Fatalf("err=%v calls=%v", err, f.calls)
	}
}
func TestEmptyQueue(t *testing.T) {
	if runAttendance(context.Background(), &fakeAttender{}, []string{""}, app.UserOption{StudentID: "selected"}, func(string, any) { t.Fatal("unexpected event") }) == nil {
		t.Fatal("expected empty selection error")
	}
}

func TestNotificationAccount(t *testing.T) {
	rows := []app.UserRow{{User: app.User{StudentID: "inactive"}}, {User: app.User{StudentID: "active"}, Current: true}}
	user, key, err := notificationAccount(rows)
	if err != nil || user.StudentID != "active" || len(key) != 64 || key == "active" {
		t.Fatal("wrong account")
	}
	_, again, _ := notificationAccount(rows)
	if key != again {
		t.Fatal("unstable identity")
	}
	rows[1].User.StudentID = "other"
	_, other, _ := notificationAccount(rows)
	if other == key {
		t.Fatal("accounts collided")
	}
	if _, _, err := notificationAccount(nil); err == nil {
		t.Fatal("missing current account accepted")
	}
	if _, _, err := notificationAccount([]app.UserRow{{Current: true}}); err == nil {
		t.Fatal("empty account accepted")
	}
}
