package macos

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/reminder"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fakeJSONBridge(t *testing.T, stdout, stderr string, exit int) (string, string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("macOS bridge")
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
	root := t.TempDir()
	request := filepath.Join(root, "request.json")
	path := filepath.Join(root, "bridge")
	script := "#!/bin/sh\ncat > " + quote(request) + "\nprintf '%s' " + quote(stdout) + "\nprintf '%s' " + quote(stderr) + " >&2\nexit " + strconv.Itoa(exit) + "\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path, request
}

func TestReminderBridgeRequestAndWarning(t *testing.T) {
	path, requestPath := fakeJSONBridge(t, `{"created":1,"syncedIds":["stable"]}`, "warning", 0)
	request := reminder.SyncRequest{ListName: "KLAP", Assignments: []reminder.Assignment{{ID: "stable", LegacyIDs: []string{"1:old"}}}}
	result, err := NewReminderBridge(path).Sync(context.Background(), request)
	if err != nil || result.Created != 1 || len(result.SyncedIDs) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	body, err := os.ReadFile(requestPath)
	if err != nil {
		t.Fatal(err)
	}
	var got reminder.SyncRequest
	if err := json.Unmarshal(body, &got); err != nil || got.ListName != "KLAP" || len(got.Assignments) != 1 || got.Assignments[0].LegacyIDs[0] != "1:old" {
		t.Fatalf("request=%s err=%v", body, err)
	}
}

func TestReminderBridgeErrors(t *testing.T) {
	for _, tc := range []struct {
		output string
		exit   int
		want   string
	}{
		{"broken", 0, "응답 파싱 실패"},
		{"", 7, "fixture failure"},
	} {
		path, _ := fakeJSONBridge(t, tc.output, "fixture failure", tc.exit)
		_, err := NewReminderBridge(path).Sync(context.Background(), reminder.SyncRequest{})
		var processErr *ProcessError
		if !errors.As(err, &processErr) || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestReminderBridgeCancellation(t *testing.T) {
	path, _ := fakeJSONBridge(t, "", "", 0)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := NewReminderBridge(path).Sync(ctx, reminder.SyncRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel=%v", err)
	}
}
