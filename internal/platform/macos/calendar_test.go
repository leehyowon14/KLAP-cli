package macos

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/calendar"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCalendarBridgeRequestAndWarning(t *testing.T) {
	path, requestPath := fakeJSONBridge(t, `{"created":1,"syncedIds":["event"]}`, "warning", 0)
	request := calendar.SyncRequest{CalendarName: "KLAP", Events: []calendar.Event{{ID: "event", AllDay: true, KnownSourceHash: "baseline"}}}
	result, err := NewCalendarBridge(path).Sync(context.Background(), request)
	if err != nil || result.Created != 1 || len(result.SyncedIDs) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	body, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var got calendar.SyncRequest
	if err := json.Unmarshal(body, &got); err != nil || got.CalendarName != "KLAP" || len(got.Events) != 1 || got.Events[0].KnownSourceHash != "baseline" || got.Events[0].ForceUpdate {
		t.Fatalf("request=%s err=%v", body, err)
	}
}

func TestCalendarBridgeErrorsAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		output string
		exit   int
		want   string
	}{
		{"broken", 0, "응답 파싱 실패"},
		{"", 7, "fixture failure"},
	} {
		path, _ := fakeJSONBridge(t, tc.output, "fixture failure", tc.exit)
		_, err := NewCalendarBridge(path).Sync(context.Background(), calendar.SyncRequest{})
		var processErr *ProcessError
		if !errors.As(err, &processErr) || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("err=%v", err)
		}
	}
	path, _ := fakeJSONBridge(t, "", "", 0)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := NewCalendarBridge(path).Sync(ctx, calendar.SyncRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel=%v", err)
	}
}
