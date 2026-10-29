package app

import (
	"path/filepath"
	"testing"
)

func TestAppTranscriptFolder(t *testing.T) {
	for _, folder := range []string{"Videos", "Video", "video"} {
		got := TranscriptPathForDownload(filepath.Join("/tmp", "KLAP", "자료구조", folder, "강의.mp4"))
		if got != "/tmp/KLAP/자료구조/Scripts/강의.txt" {
			t.Fatal(got)
		}
	}
}
