package download

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyWithProgressReportsOffsetAndTotal(t *testing.T) {
	var dst bytes.Buffer
	var events []int64
	written, err := copyWithProgress(&dst, strings.NewReader("abcdef"), 4, 10, func(bytesWritten int64, totalBytes int64) {
		if totalBytes != 10 {
			t.Fatalf("totalBytes = %d, want 10", totalBytes)
		}
		events = append(events, bytesWritten)
	})
	if err != nil {
		t.Fatalf("copyWithProgress() error = %v", err)
	}
	if written != 6 || dst.String() != "abcdef" {
		t.Fatalf("copyWithProgress() = %d, %q", written, dst.String())
	}
	if len(events) == 0 || events[len(events)-1] != 10 {
		t.Fatalf("progress events = %v, want final 10", events)
	}
}

func TestCleanupPartialDownloadRespectsKeepPartial(t *testing.T) {
	dir := t.TempDir()
	removePath := filepath.Join(dir, "remove.part")
	keepPath := filepath.Join(dir, "keep.part")
	if err := os.WriteFile(removePath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(remove) error = %v", err)
	}
	if err := os.WriteFile(keepPath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(keep) error = %v", err)
	}

	cleanupPartialDownload(removePath, false)
	if _, err := os.Stat(removePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removePath stat error = %v, want not exist", err)
	}
	cleanupPartialDownload(keepPath, true)
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("keepPath stat error = %v", err)
	}
}
