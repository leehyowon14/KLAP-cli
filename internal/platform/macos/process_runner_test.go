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
	"runtime"
	"strconv"
	"strings"
	"syscall"
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
		// Publish from the descendant itself so cleanup can still find it if
		// its launcher is canceled immediately after spawning it.
		path := os.Getenv("KLAP_TEST_DESCENDANT_PID")
		if path == "" {
			os.Exit(9)
		}
		if err := os.WriteFile(path+".tmp", []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(9)
		}
		if err := os.Rename(path+".tmp", path); err != nil {
			os.Exit(9)
		}
		// Fail-safe only; the owning test kills and waits for this process.
		time.Sleep(30 * time.Second)
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

// ownedDescendant registers cleanup before launch, including t.Fatal paths.
// Waiting on Windows is required before Go can remove the test executable.
func ownedDescendant(t *testing.T) func() *os.Process {
	t.Helper()
	path := filepath.Join(t.TempDir(), "descendant.pid")
	t.Setenv("KLAP_TEST_DESCENDANT_PID", path)
	var process *os.Process
	find := func() *os.Process {
		t.Helper()
		if process != nil {
			return process
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			data, err := os.ReadFile(path)
			if err == nil {
				pid, err := strconv.Atoi(string(data))
				if err != nil || pid <= 0 {
					t.Fatalf("invalid descendant PID: %q", data)
				}
				process, err = os.FindProcess(pid)
				if err != nil {
					t.Fatalf("find descendant: %v", err)
				}
				return process
			}
			if !errors.Is(err, os.ErrNotExist) || time.Now().After(deadline) {
				t.Fatalf("read descendant PID: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Cleanup(func() {
		child := find()
		if err := child.Kill(); err != nil {
			t.Errorf("kill descendant: %v", err)
		}
		_, err := child.Wait()
		// Unix cannot wait for a grandchild; after Kill its new parent reaps
		// it. Windows can wait on the process handle regardless of parentage.
		if runtime.GOOS != "windows" && errors.Is(err, syscall.ECHILD) {
			err = child.Release()
		}
		if err != nil {
			t.Errorf("wait/release descendant: %v", err)
		}
	})
	return find
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
