package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type pipelineOperationsStub struct {
	download   func(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error)
	transcribe func(context.Context, []LectureDownloadItem, LectureTranscriptOptions) LectureTranscriptResult
}

func (s pipelineOperationsStub) DownloadAllLectures(ctx context.Context, o LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
	return s.download(ctx, o)
}
func (s pipelineOperationsStub) TranscribeDownloadedLectures(ctx context.Context, items []LectureDownloadItem, o LectureTranscriptOptions) LectureTranscriptResult {
	return s.transcribe(ctx, items, o)
}

func TestLecturePipelineTranscribesBeforeDownloadFinishesAndDeduplicates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := make(chan struct{})
	item := LectureDownloadItem{Lecture: LectureRow{ID: "lecture"}, Path: "video.mp4"}
	transcriptions := 0
	events := 0
	operations := pipelineOperationsStub{
		download: func(ctx context.Context, o LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
			o.OnProgress(LectureDownloadProgress{Lecture: item.Lecture, Path: item.Path, Stage: "done"})
			select {
			case <-started:
			case <-ctx.Done():
				return LectureDownloadAllResult{}, ctx.Err()
			}
			return LectureDownloadAllResult{Items: []LectureDownloadItem{item, item}}, nil
		},
		transcribe: func(_ context.Context, items []LectureDownloadItem, o LectureTranscriptOptions) LectureTranscriptResult {
			transcriptions++
			if len(items) != 1 || items[0].Path != item.Path || o.Locale != "ko-KR" {
				t.Error("transcript arguments changed")
			}
			close(started)
			return LectureTranscriptResult{Items: []LectureTranscriptItem{{Lecture: item.Lecture, InputPath: item.Path, OutputPath: "video.txt"}}}
		},
	}
	result, err := runLectureDownloadPipeline(ctx, LectureDownloadPipelineOptions{Transcribe: true, TranscriptLocale: "ko-KR", OnEvent: func(e LecturePipelineEvent) {
		events++
		if (e.Download == nil) == (e.Transcript == nil) {
			t.Error("invalid event")
		}
	}}, operations)
	if err != nil || transcriptions != 1 || len(result.Transcript.Items) != 1 || events < 3 {
		t.Fatalf("result=%#v err=%v calls=%d events=%d", result, err, transcriptions, events)
	}
}

func TestLecturePipelineLimitsWorkersAndWaitsForCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 10)
	release := make(chan struct{})
	var mu sync.Mutex
	active, peak := 0, 0
	operations := pipelineOperationsStub{
		download: func(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
			items := []LectureDownloadItem{}
			for _, id := range []string{"a", "b", "c", "d"} {
				items = append(items, LectureDownloadItem{Lecture: LectureRow{ID: id}, Path: id + ".mp4"})
			}
			return LectureDownloadAllResult{Items: items}, nil
		},
		transcribe: func(ctx context.Context, items []LectureDownloadItem, _ LectureTranscriptOptions) LectureTranscriptResult {
			mu.Lock()
			active++
			peak = max(peak, active)
			mu.Unlock()
			started <- struct{}{}
			<-ctx.Done()
			<-release
			mu.Lock()
			active--
			mu.Unlock()
			return LectureTranscriptResult{Items: []LectureTranscriptItem{{Lecture: items[0].Lecture, Err: ctx.Err()}}}
		},
	}
	finished := make(chan error, 1)
	go func() {
		_, err := runLectureDownloadPipeline(ctx, LectureDownloadPipelineOptions{Transcribe: true, TranscriptConcurrency: 2}, operations)
		finished <- err
	}()
	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("workers did not start")
		}
	}
	cancel()
	select {
	case <-finished:
		t.Fatal("returned before workers stopped")
	default:
	}
	close(release)
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pipeline did not stop")
	}
	mu.Lock()
	defer mu.Unlock()
	if active != 0 || peak != 2 || len(started) != 0 {
		t.Fatalf("active=%d peak=%d extra=%d", active, peak, len(started))
	}
}

func TestLecturePipelineRetainsDownloadPartialFailure(t *testing.T) {
	want := errors.New("download failed")
	operations := pipelineOperationsStub{download: func(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
		return LectureDownloadAllResult{Items: []LectureDownloadItem{{Lecture: LectureRow{ID: "a"}, Path: "a.mp4"}}}, want
	}, transcribe: func(context.Context, []LectureDownloadItem, LectureTranscriptOptions) LectureTranscriptResult {
		t.Fatal("transcription disabled")
		return LectureTranscriptResult{}
	}}
	result, err := runLectureDownloadPipeline(context.Background(), LectureDownloadPipelineOptions{}, operations)
	if !errors.Is(err, want) || len(result.Download.Items) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestLecturePipelineSkippedVideosOnlyTranscribeMissingText(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.mp4")
	if err := os.WriteFile(TranscriptPathForDownload(existing), []byte("saved transcript"), 0600); err != nil {
		t.Fatal(err)
	}
	items := []LectureDownloadItem{
		{Lecture: LectureRow{ID: "existing"}, Path: existing, Skipped: true},
		{Lecture: LectureRow{ID: "missing"}, Path: filepath.Join(root, "missing.mp4"), Skipped: true},
		{Lecture: LectureRow{ID: "failed"}, Path: "failed.mp4", Err: errors.New("download error")},
		{Lecture: LectureRow{ID: "blank"}},
	}
	calls := 0
	wantErr := errors.New("transcript error")
	operations := pipelineOperationsStub{
		download: func(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
			return LectureDownloadAllResult{Items: items}, nil
		},
		transcribe: func(_ context.Context, batch []LectureDownloadItem, _ LectureTranscriptOptions) LectureTranscriptResult {
			calls++
			if len(batch) != 1 || batch[0].Lecture.ID != "missing" {
				t.Errorf("unexpected batch=%#v", batch)
			}
			return LectureTranscriptResult{Items: []LectureTranscriptItem{{Lecture: batch[0].Lecture, Err: wantErr}}}
		},
	}
	result, err := runLectureDownloadPipeline(context.Background(), LectureDownloadPipelineOptions{Transcribe: true}, operations)
	if err != nil || calls != 1 || len(result.Transcript.Items) != 1 || !errors.Is(result.Transcript.Items[0].Err, wantErr) {
		t.Fatalf("result=%#v err=%v calls=%d", result, err, calls)
	}
}
