package tui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func TestDownloadPipelineTerminalResultPreservesTranscriptStatus(t *testing.T) {
	row := app.LectureRow{ID: "lecture", CourseName: "과목", Lecture: app.Lecture{Title: "강의"}}
	for _, wantErr := range []error{nil, errors.New("transcript failed")} {
		m := lectureDownloadModel{ctx: context.Background(), updates: make(chan tea.Msg, 1)}
		next, _ := m.Update(lectureTranscriptProgressMsg{progress: app.LectureTranscriptProgress{Lecture: row, OutputPath: "lecture.txt", Stage: "transcribe"}})
		m = next.(lectureDownloadModel)
		if m.done {
			t.Fatal("progress prematurely completed pipeline")
		}
		next, _ = m.Update(lectureDownloadDoneMsg{all: app.LectureDownloadAllResult{Items: []app.LectureDownloadItem{{Lecture: row, Path: "lecture.mp4", Skipped: true}}}, transcript: app.LectureTranscriptResult{Items: []app.LectureTranscriptItem{{Lecture: row, InputPath: "lecture.mp4", OutputPath: "lecture.txt", Err: wantErr}}}})
		m = next.(lectureDownloadModel)
		wantStage := "transcribed"
		if wantErr != nil {
			wantStage = "transcript-error"
		}
		if !m.done || len(m.items) != 1 || m.items[0].status != wantStage || !errors.Is(m.items[0].err, wantErr) || m.items[0].skipped {
			t.Fatalf("model=%#v", m)
		}
	}
}

func TestDownloadProgressStreamKeepsFIFOAndCancelsWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	updates := make(chan tea.Msg, 2)
	updates <- lectureDownloadProgressMsg{progress: app.LectureDownloadProgress{Stage: "done"}}
	updates <- lectureDownloadDoneMsg{}
	if _, ok := waitLectureDownloadProgress(ctx, updates)().(lectureDownloadProgressMsg); !ok {
		t.Fatal("progress order lost")
	}
	if _, ok := waitLectureDownloadProgress(ctx, updates)().(lectureDownloadDoneMsg); !ok {
		t.Fatal("terminal order lost")
	}
	cancel()
	if got := waitLectureDownloadProgress(ctx, updates)(); got != nil {
		t.Fatalf("cancel message=%#v", got)
	}
}
