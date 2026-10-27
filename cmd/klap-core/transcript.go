package main

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"os"
	"path/filepath"
	"time"
)

func transcriptExists(path string) bool {
	info, err := os.Stat(app.TranscriptPathForDownload(path))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

type transcriptSample struct {
	stage    app.LectureTransferStage
	progress float64
	at       time.Time
}

func transcriptReporter(send func(string, any)) func(app.LectureTranscriptProgress) {
	last := map[string]transcriptSample{}
	return func(v app.LectureTranscriptProgress) {
		now := time.Now()
		previous, seen := last[v.Lecture.ID]
		if seen && v.Stage == app.LectureStageTranscribe && previous.stage == v.Stage && (v.Progress == previous.progress || now.Sub(previous.at) < 250*time.Millisecond) {
			return
		}
		last[v.Lecture.ID] = transcriptSample{v.Stage, v.Progress, now}
		message := ""
		if v.Err != nil {
			message = v.Err.Error()
		}
		send("transcript-progress", map[string]any{"ID": v.Lecture.ID, "Stage": v.Stage, "Path": v.OutputPath, "Error": message, "Progress": v.Progress})
	}
}

func init() {
	handlers["lecture-transcribe"] = func(ctx context.Context, s *app.Service, r request) error {
		if !filepath.IsAbs(r.Directory) || r.ID == "" {
			return fmt.Errorf("저장된 강의를 선택해 주세요")
		}
		users, err := s.Users(ctx)
		if err != nil {
			return err
		}
		user, account, err := notificationAccount(users)
		if err != nil {
			return err
		}
		if r.ExpectedAccount == "" || r.ExpectedAccount != account {
			return fmt.Errorf("계정이 변경되었습니다")
		}
		files, err := s.DownloadInventory(ctx, app.LectureDownloadAllOptions{User: user, Dir: r.Directory})
		if err != nil {
			return err
		}
		for _, file := range files {
			if file.ID == r.ID {
				emit("transcript-queue", []string{r.ID})
				s.TranscribeDownloadedLectures(ctx, []app.LectureDownloadItem{{Lecture: app.LectureRow{ID: r.ID}, Path: file.Path}}, app.LectureTranscriptOptions{Locale: r.Locale, OnProgress: transcriptReporter(emit)})
				if ctx.Err() != nil {
					return ctx.Err()
				}
				emit("transcript-result", true)
				return nil
			}
		}
		return fmt.Errorf("저장된 영상 파일을 찾지 못했습니다")
	}
}
