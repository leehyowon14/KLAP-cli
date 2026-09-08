package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptBridgeCandidatesPreferBuiltBinary(t *testing.T) {
	got := transcriptBridgeCandidates("bridges/macos")
	want := []string{
		filepath.Join("bridges", "macos", ".build", "release", "TranscriptBridge"),
		filepath.Join("bridges", "macos", ".build", "debug", "TranscriptBridge"),
		filepath.Join("bridges", "macos", "transcribe.swift"),
	}
	if len(got) != len(want) {
		t.Fatalf("transcriptBridgeCandidates() length = %d, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("transcriptBridgeCandidates()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestExecutableDirectoryResolvesSymlink(t *testing.T) {
	root := t.TempDir()
	stagedDir := filepath.Join(root, "Caskroom", "klap", "1.0.0")
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(staged dir) error = %v", err)
	}
	executable := filepath.Join(stagedDir, "klap")
	if err := os.WriteFile(executable, []byte("binary"), 0o755); err != nil {
		t.Fatalf("WriteFile(executable) error = %v", err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bin dir) error = %v", err)
	}
	symlink := filepath.Join(binDir, "klap")
	if err := os.Symlink(executable, symlink); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	want, err := filepath.EvalSymlinks(stagedDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(staged dir) error = %v", err)
	}
	if got := executableDirectory(symlink); got != want {
		t.Fatalf("executableDirectory() = %q, want %q", got, want)
	}
}

func TestExecutableDirectoryFallsBackForMissingTarget(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "bin", "klap")
	if got, want := executableDirectory(executable), filepath.Dir(executable); got != want {
		t.Fatalf("executableDirectory() = %q, want %q", got, want)
	}
}
