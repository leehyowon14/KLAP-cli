package transcript

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

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
