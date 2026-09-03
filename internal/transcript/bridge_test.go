package transcript

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeBridge struct {
	path        string
	requestPath string
}

func writeFakeBridge(t *testing.T, stdout string, stderr string, exitCode int) fakeBridge {
	t.Helper()
	quote := func(value string) string {
		return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
	}
	dir := t.TempDir()
	requestPath := filepath.Join(dir, "request.json")
	content := "#!/bin/sh\ninput=$(cat)\nprintf '%s' \"$input\" > " + quote(requestPath) + "\n" +
		"case \"$input\" in *\\\"jobs\\\"*) ;; *) echo 'missing jobs request' >&2; exit 9 ;; esac\n"
	if stdout != "" {
		content += "printf '%s' " + quote(stdout) + "\n"
	}
	if stderr != "" {
		content += "printf '%s' " + quote(stderr) + " >&2\n"
	}
	if exitCode != 0 {
		content += "exit " + strconv.Itoa(exitCode) + "\n"
	}
	path := filepath.Join(dir, "fake-transcript-bridge")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(fake bridge) error = %v", err)
	}
	return fakeBridge{path: path, requestPath: requestPath}
}

func readTranscriptFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "bridges", "macos", "testdata", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}
	return strings.TrimSpace(string(body))
}

func readFakeBridgeRequest(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(fake request) error = %v", err)
	}
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("Unmarshal(fake request) error = %v", err)
	}
	return request
}

func TestMacOSBridgeCommandSpecUsesSwiftForScript(t *testing.T) {
	name, args := NewMacOSBridge("bridges/macos/transcribe.swift").commandSpec()
	if name != "swift" {
		t.Fatalf("command name = %q, want swift", name)
	}
	if len(args) != 1 || args[0] != "bridges/macos/transcribe.swift" {
		t.Fatalf("command args = %#v", args)
	}
}

func TestMacOSBridgeCommandSpecRunsBinaryDirectly(t *testing.T) {
	name, args := NewMacOSBridge("bridges/macos/.build/release/TranscriptBridge").commandSpec()
	if name != "bridges/macos/.build/release/TranscriptBridge" {
		t.Fatalf("command name = %q", name)
	}
	if len(args) != 0 {
		t.Fatalf("command args = %#v, want empty", args)
	}
}

func TestMacOSBridgeTranscribeWithProgressUsesContext(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS transcript bridge is darwin-only")
	}
	bridgePath := filepath.Join(t.TempDir(), "bridge")
	if err := os.WriteFile(bridgePath, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(bridge) error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := NewMacOSBridge(bridgePath).TranscribeWithProgress(ctx, Request{Jobs: []Job{{
		InputPath:  "input.mp4",
		OutputPath: "output.txt",
	}}}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("TranscribeWithProgress() error = %v, want deadline exceeded", err)
	}
}

func TestMacOSBridgeTranscribeUsesResponseFixture(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS transcript bridge is darwin-only")
	}
	bridge := writeFakeBridge(t, readTranscriptFixture(t, "response.json"), "", 0)
	job := Job{
		InputPath:  "input.mp4",
		OutputPath: "output.txt",
		Locale:     "ko-KR",
		ContextualStrings: []string{
			"광운대학교",
			"컴퓨터그래픽스",
		},
	}
	response, err := NewMacOSBridge(bridge.path).Transcribe(context.Background(), Request{Jobs: []Job{job}})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	request := readFakeBridgeRequest(t, bridge.requestPath)
	wantRequest := map[string]any{
		"jobs": []any{map[string]any{
			"inputPath":         "input.mp4",
			"outputPath":        "output.txt",
			"locale":            "ko-KR",
			"contextualStrings": []any{"광운대학교", "컴퓨터그래픽스"},
		}},
	}
	if !reflect.DeepEqual(request, wantRequest) {
		t.Fatalf("bridge request = %#v, want %#v", request, wantRequest)
	}
	if len(response.Results) != 1 || response.Results[0].Text != "전사 결과" || response.Results[0].InputPath != "input.mp4" || response.Results[0].OutputPath != "output.txt" {
		t.Fatalf("Transcribe() response = %+v", response)
	}
}

func TestMacOSBridgeTranscribeWithProgressUsesNDJSONFixture(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS transcript bridge is darwin-only")
	}
	bridge := writeFakeBridge(t, readTranscriptFixture(t, "progress-response.ndjson"), "", 0)
	job := Job{InputPath: "input.mp4", OutputPath: "output.txt"}
	var progress []Progress
	response, err := NewMacOSBridge(bridge.path).TranscribeWithProgress(context.Background(), Request{Jobs: []Job{job}}, func(item Progress) {
		progress = append(progress, item)
	})
	if err != nil {
		t.Fatalf("TranscribeWithProgress() error = %v", err)
	}
	if len(progress) != 1 || progress[0].Progress != 0.5 || progress[0].InputPath != "input.mp4" || progress[0].OutputPath != "output.txt" {
		t.Fatalf("progress = %+v", progress)
	}
	request := readFakeBridgeRequest(t, bridge.requestPath)
	wantRequest := map[string]any{
		"jobs": []any{map[string]any{
			"inputPath":  "input.mp4",
			"outputPath": "output.txt",
		}},
		"progress": true,
	}
	if !reflect.DeepEqual(request, wantRequest) {
		t.Fatalf("bridge request = %#v, want %#v", request, wantRequest)
	}
	if len(response.Results) != 1 || response.Results[0].Text != "전사 결과" {
		t.Fatalf("response = %+v", response)
	}
}

func TestMacOSBridgeReportsMalformedJSON(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS transcript bridge is darwin-only")
	}
	bridge := writeFakeBridge(t, `{not-json`, "fixture stderr", 0)
	if _, err := NewMacOSBridge(bridge.path).Transcribe(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "응답 파싱 실패") {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if _, err := NewMacOSBridge(bridge.path).TranscribeWithProgress(context.Background(), Request{}, nil); err == nil || !strings.Contains(err.Error(), "이벤트 파싱 실패") {
		t.Fatalf("TranscribeWithProgress() error = %v", err)
	}
}

func TestMacOSBridgeReportsStderrAndNonZeroExit(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS transcript bridge is darwin-only")
	}
	bridge := writeFakeBridge(t, "", "bridge exploded", 7)
	if _, err := NewMacOSBridge(bridge.path).Transcribe(context.Background(), Request{}); err == nil || !strings.Contains(err.Error(), "bridge exploded") {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if _, err := NewMacOSBridge(bridge.path).TranscribeWithProgress(context.Background(), Request{}, nil); err == nil || !strings.Contains(err.Error(), "bridge exploded") {
		t.Fatalf("TranscribeWithProgress() error = %v", err)
	}
}
