package macos

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestExternalSpecPreservesLiteralTarget(t *testing.T) {
	target := "https://example.invalid/a b?x=1&y=2"
	for _, test := range []struct {
		platform, name string
		args           []string
	}{
		{"darwin", "open", []string{target}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", target}},
		{"linux", "xdg-open", []string{target}},
	} {
		got := externalSpec(test.platform, target)
		if got.Name != test.name || !reflect.DeepEqual(got.Args, test.args) {
			t.Fatalf("%s: %+v", test.platform, got)
		}
	}
	if err := OpenExternal(context.Background(), "  ", "URL"); err == nil {
		t.Fatal("empty target accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := OpenExternal(ctx, target, "URL"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled launch = %v", err)
	}
}

func TestProcessStartReturnsBeforeCompletionAndReapsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done, err := (ProcessRunner{}).Start(ctx, helperSpec(t, "hang"))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		t.Fatalf("early completion: %v", err)
	default:
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("detached helper not joined")
	}
	done, err = (ProcessRunner{}).Start(ctx, helperSpec(t, "echo"))
	if done != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("start = %v, %v", done, err)
	}
}

func TestProcessStartPreservesExitError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done, err := (ProcessRunner{}).Start(ctx, helperSpec(t, "exit"))
	if err != nil {
		t.Fatal(err)
	}
	var processErr *ProcessError
	if err := <-done; !errors.As(err, &processErr) || processErr.Stage != "exit" || processErr.Stderr != "fixture failure" {
		t.Fatalf("exit = %v", err)
	}
}

func TestProcessStartDoesNotWaitForDescendantStdout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	done, err := (ProcessRunner{}).Start(ctx, helperSpec(t, "inherited-stdout"))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("launcher exit = %v", err)
		}
	// Race-instrumented child processes add a one-second exit delay. The
	// descendant lives five seconds, so this still detects waiting for its EOF.
	case <-time.After(3 * time.Second):
		t.Fatal("launcher exit waited for descendant stdout")
	}
}
