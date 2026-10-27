package main

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptExists(t *testing.T) {
	video := filepath.Join(t.TempDir(), "lecture.mp4")
	text := app.TranscriptPathForDownload(video)
	if transcriptExists(video) {
		t.Fatal("missing transcript marked complete")
	}
	os.WriteFile(text, nil, 0600)
	if transcriptExists(video) {
		t.Fatal("empty transcript marked complete")
	}
	os.WriteFile(text, []byte("전사 내용"), 0600)
	if !transcriptExists(video) {
		t.Fatal("valid transcript not found")
	}
	os.Remove(text)
	os.Mkdir(text, 0700)
	if transcriptExists(video) {
		t.Fatal("directory marked complete")
	}
}

func TestTranscriptReporterPreservesTransitions(t *testing.T) {
	var stages []app.LectureTransferStage
	report := transcriptReporter(func(_ string, v any) { stages = append(stages, v.(map[string]any)["Stage"].(app.LectureTransferStage)) })
	for i := 0; i < 100; i++ {
		report(app.LectureTranscriptProgress{Lecture: app.LectureRow{ID: "a"}, Stage: app.LectureStageTranscribe, Progress: 0.5})
	}
	report(app.LectureTranscriptProgress{Lecture: app.LectureRow{ID: "a"}, Stage: app.LectureStageTranscribed})
	report(app.LectureTranscriptProgress{Lecture: app.LectureRow{ID: "b"}, Stage: app.LectureStageTranscribe})
	report(app.LectureTranscriptProgress{Lecture: app.LectureRow{ID: "b"}, Stage: app.LectureStageTranscriptError})
	if len(stages) != 4 {
		t.Fatalf("redundant updates or lost terminal events: %v", stages)
	}
}
