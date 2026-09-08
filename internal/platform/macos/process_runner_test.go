package macos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The test executable is the child; no Swift, permissions or external services.
func TestProcessHelper(t *testing.T) {
	args := os.Args
	if len(args) < 2 || args[len(args)-2] != "--klap-process-helper" {
		return
	}
	switch args[len(args)-1] {
	case "echo":
		_, _ = fmt.Fprint(os.Stderr, "harmless warning")
		_, _ = io.Copy(os.Stdout, os.Stdin)
	case "exit":
		_, _ = fmt.Fprint(os.Stderr, "fixture failure")
		os.Exit(7)
	case "hang":
		time.Sleep(time.Minute)
	case "linger":
		time.Sleep(5 * time.Second)
	case "inherited-stdout":
		child := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$", "--", "--klap-process-helper", "linger")
		child.Stdout = os.Stdout
		if err := child.Start(); err != nil {
			os.Exit(9)
		}
		_ = child.Process.Release()
	case "malformed":
		_, _ = fmt.Fprintln(os.Stdout, "not-json")
		for {
			_, _ = fmt.Fprintln(os.Stdout, strings.Repeat("x", 8192))
		}
	}
	os.Exit(0)
}

func helperSpec(t *testing.T, mode string) ProcessSpec {
	t.Helper()
	name, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return ProcessSpec{Name: name, Args: []string{"-test.run=^TestProcessHelper$", "--", "--klap-process-helper", mode}}
}

func TestProcessRunnerSeparatesStreams(t *testing.T) {
	spec := helperSpec(t, "echo")
	spec.Input = []byte("{\"ok\":true}")
	var result map[string]bool
	err := (ProcessRunner{}).Run(context.Background(), spec, func(r io.Reader) error { return json.NewDecoder(r).Decode(&result) })
	if err != nil || !result["ok"] {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func TestProcessRunnerExitAndStartErrors(t *testing.T) {
	err := (ProcessRunner{}).Run(context.Background(), helperSpec(t, "exit"), nil)
	var processErr *ProcessError
	if !errors.As(err, &processErr) || processErr.Stage != "exit" || processErr.Stderr != "fixture failure" {
		t.Fatalf("exit=%v", err)
	}
	err = (ProcessRunner{}).Run(context.Background(), ProcessSpec{Name: filepath.Join(t.TempDir(), "missing")}, nil)
	if !errors.As(err, &processErr) || processErr.Stage != "start" {
		t.Fatalf("start=%v", err)
	}
}

func TestProcessRunnerCancellation(t *testing.T) {
	for _, alreadyCanceled := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		if alreadyCanceled {
			cancel()
		}
		started := time.Now()
		err := (ProcessRunner{}).Run(ctx, helperSpec(t, "hang"), nil)
		cancel()
		want := context.DeadlineExceeded
		if alreadyCanceled {
			want = context.Canceled
		}
		if !errors.Is(err, want) || time.Since(started) > 3*time.Second {
			t.Fatalf("cancel=%v duration=%v", err, time.Since(started))
		}
	}
}

func TestProcessRunnerDecoderFailureCancelsAndJoins(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started := time.Now()
	err := (ProcessRunner{}).Run(ctx, helperSpec(t, "malformed"), func(r io.Reader) error { var result any; return json.NewDecoder(r).Decode(&result) })
	var processErr *ProcessError
	if !errors.As(err, &processErr) || processErr.Stage != "decode" || errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 2*time.Second {
		t.Fatalf("decode=%v duration=%v", err, time.Since(started))
	}
}

func TestBridgeSpecSupportsScriptAndExecutable(t *testing.T) {
	script := bridgeSpec(" bridge.swift ", nil)
	if script.Name != "swift" || len(script.Args) != 1 || script.Args[0] != "bridge.swift" {
		t.Fatalf("script=%+v", script)
	}
	binary := bridgeSpec(" /tmp/TranscriptBridge ", nil)
	if binary.Name != "/tmp/TranscriptBridge" || len(binary.Args) != 0 {
		t.Fatalf("binary=%+v", binary)
	}
}
