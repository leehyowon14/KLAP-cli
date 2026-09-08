package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBridgeDiscovery(t *testing.T) {
	for _, product := range []string{"ReminderBridge", "CalendarBridge", "CategoryBridge", "TranscriptBridge"} {
		t.Run(product, func(t *testing.T) {
			root := t.TempDir()
			executable := filepath.Join(root, "installed", "klap")
			candidates := bridgeCandidates(product, executable, filepath.Join(root, "work"), filepath.Join(root, "source"))
			write := func(path string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("fake"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if got := selectBridgePath(candidates); got != candidates[0] {
				t.Fatalf("missing fallback = %q", got)
			}
			dev := candidates[2]
			write(dev)
			if got := selectBridgePath(candidates); got != dev {
				t.Fatalf("development artifact = %q", got)
			}
			if err := os.MkdirAll(candidates[0], 0o755); err != nil {
				t.Fatal(err)
			}
			if got := selectBridgePath(candidates); got != dev {
				t.Fatalf("directory accepted = %q", got)
			}
			write(candidates[1])
			if got := selectBridgePath(candidates); got != candidates[1] {
				t.Fatalf("installed artifact = %q", got)
			}
		})
	}
}

func TestBridgeEnvironmentOverrides(t *testing.T) {
	for _, test := range []struct {
		name    string
		resolve func() string
	}{
		{"KLAP_REMINDER_BRIDGE", defaultReminderBridgePath},
		{"KLAP_CALENDAR_BRIDGE", defaultCalendarBridgePath},
		{"KLAP_CATEGORY_BRIDGE", defaultCategoryBridgePath},
		{"KLAP_TRANSCRIPT_BRIDGE", defaultTranscriptBridgePath},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.name, "/explicit/custom.swift")
			if got := test.resolve(); got != "/explicit/custom.swift" {
				t.Fatalf("override = %q", got)
			}
		})
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
