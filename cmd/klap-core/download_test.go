package main

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeDownloader struct {
	progress []app.LectureDownloadProgress
	calls    int
	opts     app.LectureDownloadAllOptions
	result   app.LectureDownloadAllResult
	err      error
}

func (f *fakeDownloader) DownloadAllLectures(ctx context.Context, o app.LectureDownloadAllOptions) (app.LectureDownloadAllResult, error) {
	f.calls++
	f.opts = o
	for _, p := range f.progress {
		if o.OnProgress != nil {
			o.OnProgress(p)
		}
	}
	return f.result, f.err
}
func TestDownloadSelection(t *testing.T) {
	fake := &fakeDownloader{}
	for _, r := range []request{{IDs: nil, Directory: "/tmp"}, {IDs: []string{" "}, Directory: "/tmp"}, {IDs: []string{"a"}, Directory: "relative"}} {
		if runDownloads(context.Background(), fake, r, app.UserOption{}, func(string, any) { t.Fatal("unexpected output") }) == nil {
			t.Fatal("invalid request accepted")
		}
	}
	if fake.calls != 0 {
		t.Fatal("empty selection must never download all lectures")
	}
	fake.result = app.LectureDownloadAllResult{Items: []app.LectureDownloadItem{{Lecture: app.LectureRow{ID: "a"}, Path: "/tmp/a.mp4"}, {Lecture: app.LectureRow{ID: "b"}, Err: errors.New("지원하지 않음")}}}
	var rows []downloadItem
	err := runDownloads(context.Background(), fake, request{IDs: []string{"a", "a", " b ", "missing"}, Directory: "/tmp"}, app.UserOption{}, func(kind string, data any) { rows = data.([]downloadItem) })
	if err != nil || len(fake.opts.LectureIDs) != 3 || fake.opts.Concurrency != 1 || len(rows) != 3 {
		t.Fatalf("selection/result mismatch: %v", err)
	}
	if rows[0].Error != "" || rows[1].Error != "지원하지 않음" || rows[2].Error == "" {
		t.Fatal("partial failure or missing selection lost")
	}
}
func TestDownloadCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runDownloads(ctx, &fakeDownloader{}, request{IDs: []string{"a"}, Directory: "/tmp"}, app.UserOption{}, func(string, any) { t.Fatal("cancelled run reported completion") })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
}

func TestDownloadConcurrency(t *testing.T) {
	for _, pair := range [][2]int{{-1, 1}, {0, 1}, {1, 1}, {3, 3}, {8, 8}, {100, 8}} {
		if got := downloadConcurrency(pair[0]); got != pair[1] {
			t.Fatalf("limit %d: %d", pair[0], got)
		}
	}
	fake := &fakeDownloader{}
	_ = runDownloads(context.Background(), fake, request{IDs: []string{"a"}, Directory: "/tmp", Concurrency: 6, Adaptive: true}, app.UserOption{}, func(string, any) {})
	if fake.opts.Concurrency != 6 || !fake.opts.Adaptive {
		t.Fatal("adaptive options lost")
	}
}

type fakePipeline struct {
	mu     sync.Mutex
	calls  []string
	cancel context.CancelFunc
}

func (f *fakePipeline) DownloadAllLectures(ctx context.Context, o app.LectureDownloadAllOptions) (app.LectureDownloadAllResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, "download-start")
	f.mu.Unlock()
	if o.OnProgress != nil {
		o.OnProgress(app.LectureDownloadProgress{Lecture: app.LectureRow{ID: "a"}, Stage: app.LectureStageDone})
	}
	f.mu.Lock()
	f.calls = append(f.calls, "download-end")
	f.mu.Unlock()
	return app.LectureDownloadAllResult{Items: []app.LectureDownloadItem{{Lecture: app.LectureRow{ID: "a"}, Path: "/tmp/a.mp4"}, {Lecture: app.LectureRow{ID: "bad"}, Err: errors.New("download failure")}, {Lecture: app.LectureRow{ID: "b"}, Path: "/tmp/b.mp4"}}}, nil
}
func (f *fakePipeline) TranscribeDownloadedLectures(ctx context.Context, items []app.LectureDownloadItem, o app.LectureTranscriptOptions) app.LectureTranscriptResult {
	if len(items) != 1 || o.Locale != "en-US" {
		panic("sequential transcription or locale lost")
	}
	f.mu.Lock()
	f.calls = append(f.calls, "transcribe-"+items[0].Lecture.ID)
	f.mu.Unlock()
	if f.cancel != nil {
		f.cancel()
	}
	o.OnProgress(app.LectureTranscriptProgress{Lecture: items[0].Lecture, Stage: app.LectureStageTranscriptError, Progress: 0.4, Err: errors.New("speech unavailable")})
	return app.LectureTranscriptResult{}
}
func TestTranscriptFailurePreservesDownload(t *testing.T) {
	progress := false
	transcript := false
	p := pipelineDownloader{service: &fakePipeline{}, locale: "en-US", send: func(kind string, data any) {
		if kind != "transcript-progress" {
			return
		}
		v := data.(map[string]any)
		if v["Stage"] == "transcript-waiting" {
			return
		}
		transcript = kind == "transcript-progress" && v["Error"] == "speech unavailable"
	}}
	result, err := p.DownloadAllLectures(context.Background(), app.LectureDownloadAllOptions{OnProgress: func(app.LectureDownloadProgress) { progress = true }})
	if err != nil || !progress || !transcript || len(result.Items) != 3 || result.Items[0].Path != "/tmp/a.mp4" {
		t.Fatal("pipeline lost completed download or transcript failure")
	}
}

func TestProgressThrottleKeepsTransitions(t *testing.T) {
	f := &fakeDownloader{}
	for i := int64(0); i < 100; i++ {
		f.progress = append(f.progress, app.LectureDownloadProgress{Lecture: app.LectureRow{ID: "a"}, Stage: app.LectureStageDownload, Bytes: i})
	}
	for _, stage := range []app.LectureTransferStage{"paused", app.LectureStageDownload, app.LectureStageDone} {
		f.progress = append(f.progress, app.LectureDownloadProgress{Lecture: app.LectureRow{ID: "a"}, Stage: stage, Bytes: 100})
	}
	stages := []app.LectureTransferStage{}
	_ = runDownloads(context.Background(), f, request{IDs: []string{"a"}, Directory: "/tmp"}, app.UserOption{}, func(kind string, value any) {
		if kind == "download-progress" {
			stages = append(stages, value.(map[string]any)["Stage"].(app.LectureTransferStage))
		}
	})
	if len(stages) != 4 || stages[1] != "paused" || stages[3] != app.LectureStageDone {
		t.Fatalf("transition events lost: %v", stages)
	}
}

func TestDownloadThenSequentialTranscription(t *testing.T) {
	f := &fakePipeline{}
	p := pipelineDownloader{service: f, locale: "en-US", send: func(string, any) {}}
	_, err := p.DownloadAllLectures(context.Background(), app.LectureDownloadAllOptions{})
	if err != nil || strings.Join(f.calls, ",") != "download-start,download-end,transcribe-a,transcribe-b" {
		t.Fatalf("wrong phase order: %v %v", f.calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	f = &fakePipeline{cancel: cancel}
	p.service = f
	_, err = p.DownloadAllLectures(ctx, app.LectureDownloadAllOptions{})
	if !errors.Is(err, context.Canceled) || strings.Join(f.calls, ",") != "download-start,download-end,transcribe-a" {
		t.Fatalf("cancel must stop next transcription: %v", f.calls)
	}
}

func TestTranscriptQueueAndProgressEvents(t *testing.T) {
	var ids []string
	var progress float64
	p := pipelineDownloader{service: &fakePipeline{}, locale: "en-US", send: func(kind string, value any) {
		if kind == "transcript-queue" {
			ids = value.([]string)
		}
		if kind == "transcript-progress" && value.(map[string]any)["Stage"] != "transcript-waiting" {
			progress = value.(map[string]any)["Progress"].(float64)
		}
	}}
	_, err := p.DownloadAllLectures(context.Background(), app.LectureDownloadAllOptions{})
	if err != nil || strings.Join(ids, ",") != "a,b" || progress != 0.4 {
		t.Fatalf("queue/progress lost: %v %v %v", ids, progress, err)
	}
}

type overlappingPipeline struct {
	started     chan struct{}
	release     chan struct{}
	mu          sync.Mutex
	active, max int
	order       []string
}

func (f *overlappingPipeline) DownloadAllLectures(ctx context.Context, o app.LectureDownloadAllOptions) (app.LectureDownloadAllResult, error) {
	a := app.LectureDownloadItem{Lecture: app.LectureRow{ID: "a"}, Path: "/tmp/a.mp4"}
	b := app.LectureDownloadItem{Lecture: app.LectureRow{ID: "b"}, Path: "/tmp/b.mp4"}
	o.OnProgress(app.LectureDownloadProgress{Lecture: a.Lecture, Path: a.Path, Stage: app.LectureStageDone})
	select {
	case <-f.started:
	case <-ctx.Done():
		return app.LectureDownloadAllResult{}, ctx.Err()
	}
	o.OnProgress(app.LectureDownloadProgress{Lecture: b.Lecture, Path: b.Path, Stage: app.LectureStageDone})
	o.OnProgress(app.LectureDownloadProgress{Lecture: b.Lecture, Path: b.Path, Stage: app.LectureStageDone})
	close(f.release)
	return app.LectureDownloadAllResult{Items: []app.LectureDownloadItem{a, b}}, nil
}
func (f *overlappingPipeline) TranscribeDownloadedLectures(ctx context.Context, items []app.LectureDownloadItem, o app.LectureTranscriptOptions) app.LectureTranscriptResult {
	f.mu.Lock()
	f.active++
	if f.active > f.max {
		f.max = f.active
	}
	f.order = append(f.order, items[0].Lecture.ID)
	f.mu.Unlock()
	if items[0].Lecture.ID == "a" {
		close(f.started)
		select {
		case <-f.release:
		case <-ctx.Done():
		}
	}
	o.OnProgress(app.LectureTranscriptProgress{Lecture: items[0].Lecture, Stage: app.LectureStageTranscribed})
	f.mu.Lock()
	f.active--
	f.mu.Unlock()
	return app.LectureTranscriptResult{}
}
func TestOverlappingDownloadsSingleTranscriptWorker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	f := &overlappingPipeline{started: make(chan struct{}), release: make(chan struct{})}
	finished := false
	p := pipelineDownloader{service: f, locale: "ko-KR", send: func(kind string, _ any) {
		if kind == "download-finished" {
			finished = true
		}
	}}
	_, err := p.DownloadAllLectures(ctx, app.LectureDownloadAllOptions{})
	if err != nil || !finished || f.max != 1 || strings.Join(f.order, ",") != "a,b" {
		t.Fatalf("overlap/dedup/single worker: %v %v %d", err, f.order, f.max)
	}
}
