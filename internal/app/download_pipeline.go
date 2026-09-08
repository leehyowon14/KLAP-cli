package app

import (
	"context"
	"strings"
)

type LectureDownloadPipelineOptions struct {
	Download              LectureDownloadAllOptions
	Transcribe            bool
	TranscriptLocale      string
	TranscriptConcurrency int
	OnEvent               func(LecturePipelineEvent)
}

// Exactly one progress payload is present in each event. Events are serialized.
type LecturePipelineEvent struct {
	Download   *LectureDownloadProgress
	Transcript *LectureTranscriptProgress
}

type LectureDownloadPipelineResult struct {
	Download   LectureDownloadAllResult
	Transcript LectureTranscriptResult
}

type lecturePipelineOperations interface {
	DownloadAllLectures(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error)
	TranscribeDownloadedLectures(context.Context, []LectureDownloadItem, LectureTranscriptOptions) LectureTranscriptResult
}

func (s *Service) RunLectureDownloadPipeline(ctx context.Context, opts LectureDownloadPipelineOptions) (LectureDownloadPipelineResult, error) {
	return runLectureDownloadPipeline(ctx, opts, s)
}

type pipelineDownloadDone struct {
	result LectureDownloadAllResult
	err    error
}
type pipelineTranscriptDone struct{ result LectureTranscriptResult }

func runLectureDownloadPipeline(ctx context.Context, opts LectureDownloadPipelineOptions, operations lecturePipelineOperations) (LectureDownloadPipelineResult, error) {
	messages := make(chan any, 16)
	emit := func(event LecturePipelineEvent) {
		if opts.OnEvent != nil {
			opts.OnEvent(event)
		}
	}
	go func() {
		download := opts.Download
		download.OnProgress = func(progress LectureDownloadProgress) { messages <- progress }
		result, err := operations.DownloadAllLectures(ctx, download)
		messages <- pipelineDownloadDone{result: result, err: err}
	}()
	limit := max(1, opts.TranscriptConcurrency)
	queue := []LectureDownloadItem{}
	seen := map[string]bool{}
	enqueue := func(item LectureDownloadItem) {
		if !opts.Transcribe || ctx.Err() != nil || !LectureDownloadItemNeedsTranscript(item) {
			return
		}
		key := lecturePipelineKey(item.Lecture)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		queue = append(queue, item)
		progress := LectureTranscriptProgress{Lecture: item.Lecture, InputPath: item.Path, OutputPath: TranscriptPathForDownload(item.Path), Stage: LectureStageTranscribe}
		emit(LecturePipelineEvent{Transcript: &progress})
	}
	active := 0
	downloadDone := false
	var result LectureDownloadPipelineResult
	var downloadErr error
	for !downloadDone || active > 0 || len(queue) > 0 {
		if ctx.Err() != nil {
			queue = nil
		}
		for active < limit && len(queue) > 0 {
			item := queue[0]
			queue = queue[1:]
			active++
			go func() {
				transcribed := operations.TranscribeDownloadedLectures(ctx, []LectureDownloadItem{item}, LectureTranscriptOptions{Locale: opts.TranscriptLocale, OnProgress: func(progress LectureTranscriptProgress) { messages <- progress }})
				messages <- pipelineTranscriptDone{result: transcribed}
			}()
		}
		if downloadDone && active == 0 {
			break
		}
		switch message := (<-messages).(type) {
		case LectureDownloadProgress:
			if opts.Download.OnProgress != nil {
				opts.Download.OnProgress(message)
			}
			emit(LecturePipelineEvent{Download: &message})
			if message.Stage == LectureStageDone {
				enqueue(LectureDownloadItem{Lecture: message.Lecture, Path: message.Path, Bytes: message.Bytes, Skipped: message.Skipped, Err: message.Err})
			}
		case LectureTranscriptProgress:
			emit(LecturePipelineEvent{Transcript: &message})
		case pipelineDownloadDone:
			downloadDone = true
			result.Download = message.result
			downloadErr = message.err
			for _, item := range message.result.Items {
				enqueue(item)
			}
		case pipelineTranscriptDone:
			active--
			result.Transcript.Items = append(result.Transcript.Items, message.result.Items...)
			for _, item := range message.result.Items {
				stage := LectureStageTranscribed
				if item.Err != nil {
					stage = LectureStageTranscriptError
				}
				progress := LectureTranscriptProgress{Lecture: item.Lecture, InputPath: item.InputPath, OutputPath: item.OutputPath, Stage: stage, Err: item.Err}
				emit(LecturePipelineEvent{Transcript: &progress})
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, downloadErr
}

func lecturePipelineKey(row LectureRow) string {
	if id := strings.TrimSpace(row.ID); id != "" {
		return id
	}
	parts := []string{row.TermValue, row.CourseName, row.Lecture.ContentID, row.Lecture.ModuleTitle, row.Lecture.Title}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "\x00")
}
