package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestRunWritesStartupFailureAndReturnsNonZeroExitCode(t *testing.T) {
	wantErr := errors.New("settings store 초기화 실패")
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"dashboard"}, &stderr, func(context.Context, []string) error {
		return wantErr
	})

	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if got := stderr.String(); got != wantErr.Error()+"\n" {
		t.Fatalf("run() stderr = %q", got)
	}
}

func TestRunReturnsSuccessWithoutStderr(t *testing.T) {
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"help"}, &stderr, func(context.Context, []string) error {
		return nil
	})

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("run() stderr = %q", stderr.String())
	}
}
