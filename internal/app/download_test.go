package app

import (
	"bytes"
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
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
	row := LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "a", Title: "소개"}}

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
		Lecture: klas.Lecture{
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

func TestDownloadStatusReportsPartialFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4"), []byte("done"), 0o644); err != nil {
		t.Fatalf("WriteFile(done) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4.part"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(partial) error = %v", err)
	}

	got, err := (&Service{}).DownloadStatus(dir)
	if err != nil {
		t.Fatalf("DownloadStatus() error = %v", err)
	}
	if got.Files != 1 || got.PartialFiles != 1 || got.PartialBytes != int64(len("partial")) {
		t.Fatalf("DownloadStatus() = %+v", got)
	}
}

func TestLectureFilenameSanitizesPathComponents(t *testing.T) {
	got := lectureVideoPath("root", "오픈소스/실습", klas.Lecture{
		ModuleTitle: "1주차: 소개",
		Title:       "Git? GitHub* 시작",
	}, "https://media.example.com/video.mp4?token=1", 1)
	want := filepath.Join("root", "오픈소스_실습", "video", "1-1. Git_ GitHub_ 시작.mp4")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureFilenameUsesKlasWeekSequence(t *testing.T) {
	got := lectureVideoPath("root", "컴퓨터그래픽스", klas.Lecture{
		WeekNo:      "14",
		WeeklySeq:   "2",
		ModuleTitle: "보강",
		Title:       "기말/정리",
	}, "https://media.example.com/final.mov", 2)
	want := filepath.Join("root", "컴퓨터그래픽스", "video", "14-2. 기말_정리.mov")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureWeekOrderFallsBackToCourseOrder(t *testing.T) {
	rows := []LectureRow{
		{ID: "1:a", Lecture: klas.Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "첫번째"}},
		{ID: "1:b", Lecture: klas.Lecture{ContentID: "b", ModuleTitle: "1주차", Title: "두번째"}},
		{ID: "1:c", Lecture: klas.Lecture{ContentID: "c", ModuleTitle: "2주차", Title: "첫번째"}},
	}
	orders := lectureRowWeekOrders(rows)
	if orders["1:a"] != 1 || orders["1:b"] != 2 || orders["1:c"] != 1 {
		t.Fatalf("lectureRowWeekOrders() = %#v", orders)
	}
}
