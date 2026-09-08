package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptPath(t *testing.T) {
	if got := transcriptPath(filepath.Join("downloads", "컴퓨터그래픽스", "video", "lecture.mp4")); got != filepath.Join("downloads", "컴퓨터그래픽스", "transcription", "lecture.txt") {
		t.Fatalf("transcriptPath() = %q", got)
	}
	if got := transcriptPath(filepath.Join("downloads", "lecture")); got != filepath.Join("downloads", "lecture.txt") {
		t.Fatalf("transcriptPath() without extension = %q", got)
	}
}

func TestLectureDownloadItemNeedsTranscriptForSkippedVideo(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "컴퓨터그래픽스", "video", "lecture.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(video) error = %v", err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	item := LectureDownloadItem{Path: videoPath, Skipped: true}
	if !LectureDownloadItemNeedsTranscript(item) {
		t.Fatal("LectureDownloadItemNeedsTranscript() expected true without transcript")
	}

	outputPath := filepath.Join(root, "컴퓨터그래픽스", "transcription", "lecture.txt")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(transcript) error = %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("text"), 0o644); err != nil {
		t.Fatalf("WriteFile(transcript) error = %v", err)
	}
	if LectureDownloadItemNeedsTranscript(item) {
		t.Fatal("LectureDownloadItemNeedsTranscript() expected false with existing transcript")
	}
}

func TestTranscribeDownloadedLecturesReturnsPreflightFailure(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "컴퓨터그래픽스")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile(blocker) error = %v", err)
	}
	videoPath := filepath.Join(blocker, "video", "lecture.mp4")
	row := LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: Lecture{ContentID: "a", Title: "소개"}}

	result := (&Service{}).TranscribeDownloadedLectures(context.Background(), []LectureDownloadItem{{
		Lecture: row,
		Path:    videoPath,
	}}, LectureTranscriptOptions{})

	if len(result.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(result.Items))
	}
	if result.Items[0].Err == nil {
		t.Fatalf("Items[0].Err is nil")
	}
	if result.Items[0].InputPath != videoPath || result.Items[0].Lecture.CourseName != "컴퓨터그래픽스" {
		t.Fatalf("Items[0] = %+v", result.Items[0])
	}
}

func TestLectureTranscriptContextIncludesCourseAndCodeSwitching(t *testing.T) {
	got := lectureTranscriptContext(LectureRow{
		CourseName: "컴퓨터그래픽스",
		Lecture: Lecture{
			ModuleTitle: "14주차",
			Title:       "렌더링 파이프라인",
		},
	})
	for _, want := range []string{"광운대학교", "컴퓨터그래픽스", "14주차", "렌더링 파이프라인", "code switching"} {
		if !containsString(got, want) {
			t.Fatalf("lectureTranscriptContext() missing %q: %v", want, got)
		}
	}
}

func TestLectureWeekOrderFallsBackToCourseOrder(t *testing.T) {
	rows := []LectureRow{
		{ID: "1:a", Lecture: Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "첫번째"}},
		{ID: "1:b", Lecture: Lecture{ContentID: "b", ModuleTitle: "1주차", Title: "두번째"}},
		{ID: "1:c", Lecture: Lecture{ContentID: "c", ModuleTitle: "2주차", Title: "첫번째"}},
	}
	orders := lectureRowWeekOrders(rows)
	if orders["1:a"] != 1 || orders["1:b"] != 2 || orders["1:c"] != 1 {
		t.Fatalf("lectureRowWeekOrders() = %#v", orders)
	}
}
