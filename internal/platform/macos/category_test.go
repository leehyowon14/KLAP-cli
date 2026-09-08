package macos

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCategoryBridgeProtocol(t *testing.T) {
	path, requestPath := fakeJSONBridge(t, `{"reminders":["KLAP"],"calendars":["학사일정"]}`, "warning", 0)
	result, err := NewCategoryBridge(path).List(context.Background())
	if err != nil || len(result.Reminders) != 1 || result.Reminders[0] != "KLAP" || len(result.Calendars) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	body, err := os.ReadFile(requestPath)
	if err != nil || len(body) != 0 {
		t.Fatalf("request=%q err=%v", body, err)
	}
}

func TestCategoryBridgeErrorsAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		output string
		exit   int
		want   string
	}{
		{"broken", 0, "응답 파싱 실패"},
		{"", 7, "fixture failure"},
	} {
		path, _ := fakeJSONBridge(t, tc.output, "fixture failure", tc.exit)
		_, err := NewCategoryBridge(path).List(context.Background())
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
	_, err := NewCategoryBridge(path).List(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel=%v", err)
	}
}
