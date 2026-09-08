package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadRunCleanupWaitsForWriterAndPreservesExistingFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "new.mp4")
	existing := filepath.Join(root, "existing.mp4")
	resumed := filepath.Join(root, "resumed.mp4")
	if err := os.WriteFile(existing, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resumed+".part", []byte("previous partial"), 0600); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	canceled := make(chan struct{})
	ops := pipelineOperationsStub{download: func(ctx context.Context, o LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
		o.OnProgress(LectureDownloadProgress{Stage: "download", Path: path})
		o.OnProgress(LectureDownloadProgress{Stage: "download", Path: existing})
		o.OnProgress(LectureDownloadProgress{Stage: "download", Path: resumed})
		close(started)
		<-ctx.Done()
		close(canceled)
		<-release
		if err := os.WriteFile(path, []byte("late writer"), 0600); err != nil {
			t.Error(err)
		}
		if err := os.Rename(resumed+".part", resumed); err != nil {
			t.Error(err)
		}
		return LectureDownloadAllResult{}, ctx.Err()
	}}
	run := newLectureDownloadRun(context.Background(), LectureDownloadPipelineOptions{}, ops)
	finished := make(chan error, 1)
	go func() { _, err := run.Run(); finished <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("run not started")
	}
	cleaned := make(chan error, 1)
	go func() { cleaned <- run.CancelAndCleanup() }()
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("run not canceled")
	}
	select {
	case <-cleaned:
		t.Fatal("cleanup did not wait for writer")
	default:
	}
	close(release)
	select {
	case err := <-cleaned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup blocked")
	}
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new file remains: %v", err)
	}
	for _, saved := range []string{existing, resumed} {
		if _, err := os.Stat(saved); err != nil {
			t.Fatalf("existing removed: %s %v", saved, err)
		}
	}
	if err := run.CancelAndCleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadRunCancelBeforeStartDoesNotInvokeOperations(t *testing.T) {
	run := newLectureDownloadRun(context.Background(), LectureDownloadPipelineOptions{}, pipelineOperationsStub{download: func(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
		t.Error("canceled run performed IO")
		return LectureDownloadAllResult{}, nil
	}})
	if err := run.CancelAndCleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := run.Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestDownloadRunCleanupReportsNonFileAndPreservesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.mp4")
	run := newLectureDownloadRun(context.Background(), LectureDownloadPipelineOptions{}, pipelineOperationsStub{download: func(_ context.Context, o LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
		o.OnProgress(LectureDownloadProgress{Stage: "download", Path: path})
		return LectureDownloadAllResult{}, nil
	}})
	if _, err := run.Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := run.CancelAndCleanup(); err == nil {
		t.Fatal("cleanup failure hidden")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("directory removed: %v", err)
	}
}

func TestDownloadRunTracksTranscriptBeforeWriting(t *testing.T) {
	for _, preexisting := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "lecture.mp4")
		output := TranscriptPathForDownload(path)
		if preexisting {
			if err := os.WriteFile(output, []byte("previous"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		item := LectureDownloadItem{Lecture: LectureRow{ID: "lecture"}, Path: path}
		ops := pipelineOperationsStub{
			download: func(context.Context, LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
				return LectureDownloadAllResult{Items: []LectureDownloadItem{item}}, nil
			},
			transcribe: func(context.Context, []LectureDownloadItem, LectureTranscriptOptions) LectureTranscriptResult {
				if err := os.WriteFile(output, []byte("transcribed"), 0600); err != nil {
					t.Error(err)
				}
				return LectureTranscriptResult{Items: []LectureTranscriptItem{{Lecture: item.Lecture, OutputPath: output}}}
			},
		}
		run := newLectureDownloadRun(context.Background(), LectureDownloadPipelineOptions{Transcribe: true}, ops)
		if _, err := run.Run(); err != nil {
			t.Fatal(err)
		}
		if err := run.CancelAndCleanup(); err != nil {
			t.Fatal(err)
		}
		_, err := os.Stat(output)
		if preexisting && err != nil {
			t.Fatalf("existing transcript removed: %v", err)
		}
		if !preexisting && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("new transcript retained: %v", err)
		}
	}
}
