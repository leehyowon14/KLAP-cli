package download

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadStatusReportsPartialFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4"), []byte("done"), 0o644); err != nil {
		t.Fatalf("WriteFile(done) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4.part"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(partial) error = %v", err)
	}

	got, err := NewClient(nil).Status(dir)
	if err != nil {
		t.Fatalf("DownloadStatus() error = %v", err)
	}
	if got.Files != 1 || got.PartialFiles != 1 || got.PartialBytes != int64(len("partial")) {
		t.Fatalf("DownloadStatus() = %+v", got)
	}
}

func TestStatusMissingRootAndLatestLimit(t *testing.T) {
	root := t.TempDir()
	got, err := NewClient(nil).Status(filepath.Join(root, "missing"))
	if err != nil || got.Files != 0 {
		t.Fatalf("missing=%+v err=%v", got, err)
	}
	for i := 0; i < 12; i++ {
		path := filepath.Join(root, fmt.Sprintf("%02d.mp4", i))
		if err := os.WriteFile(path, []byte("v"), 0600); err != nil {
			t.Fatal(err)
		}
		stamp := time.Unix(int64(i+1), 0)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	got, err = NewClient(nil).Status(root)
	if err != nil || got.Files != 12 || got.Bytes != 12 || len(got.Items) != 10 || filepath.Base(got.Items[0].Path) != "11.mp4" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
