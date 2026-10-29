package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingDownloadFile(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		size       int
		want       bool
	}{{"complete", "Videos/1. 강의.mp4", 1, true}, {"legacy", "video/1. 강의.mp4", 1, true}, {"extension", "Videos/1. 강의.MOV", 1, true}, {"partial", "Videos/1. 강의.mp4.part", 1, false}, {"empty", "Videos/1. 강의.mp4", 0, false}, {"different", "Videos/2. 강의.mp4", 1, false}} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, tc.file)
			os.MkdirAll(filepath.Dir(file), 0700)
			os.WriteFile(file, make([]byte, tc.size), 0600)
			got, size := existingDownloadFile(filepath.Join(root, "Videos", "1. 강의.mp4"))
			if (got != "") != tc.want || tc.want && size != int64(tc.size) {
				t.Fatalf("got %s %d", got, size)
			}
		})
	}
}
