package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// LectureDownloadRun owns cancellation, worker completion and artifact cleanup.
// Construction does not start I/O. Run and CancelAndCleanup are blocking actions.
type LectureDownloadRun struct {
	ctx         context.Context
	cancel      context.CancelFunc
	opts        LectureDownloadPipelineOptions
	operations  lecturePipelineOperations
	startOnce   sync.Once
	cleanupOnce sync.Once
	done        chan struct{}
	result      LectureDownloadPipelineResult
	err         error
	mu          sync.Mutex
	owned       map[string]bool
	snapshotErr error
	cleanupErr  error
}

func (s *Service) NewLectureDownloadRun(ctx context.Context, opts LectureDownloadPipelineOptions) *LectureDownloadRun {
	return newLectureDownloadRun(ctx, opts, s)
}

func newLectureDownloadRun(ctx context.Context, opts LectureDownloadPipelineOptions, operations lecturePipelineOperations) *LectureDownloadRun {
	runCtx, cancel := context.WithCancel(ctx)
	return &LectureDownloadRun{ctx: runCtx, cancel: cancel, opts: opts, operations: operations, done: make(chan struct{}), owned: make(map[string]bool)}
}

func (r *LectureDownloadRun) start() {
	r.startOnce.Do(func() {
		go func() {
			defer close(r.done)
			defer r.cancel()
			if err := r.ctx.Err(); err != nil {
				r.err = err
				return
			}
			r.result, r.err = runLectureDownloadPipeline(r.ctx, r.opts, trackingPipelineOperations{run: r, delegate: r.operations})
		}()
	})
}

func (r *LectureDownloadRun) Run() (LectureDownloadPipelineResult, error) {
	r.start()
	<-r.done
	return r.result, r.err
}
func (r *LectureDownloadRun) Cancel() { r.cancel() }

// CancelAndCleanup joins all writers before removing only paths absent before
// this run first attempted to write them. Existing videos, partials and text stay.
func (r *LectureDownloadRun) CancelAndCleanup() error {
	r.cancel()
	r.start()
	<-r.done
	r.cleanupOnce.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.cleanupErr = r.snapshotErr
		paths := make([]string, 0, len(r.owned))
		for path, owned := range r.owned {
			if owned {
				paths = append(paths, path)
			}
		}
		sort.Strings(paths)
		for _, path := range paths {
			info, err := os.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err == nil && !info.Mode().IsRegular() {
				err = errors.New("정리 대상이 일반 파일이 아닙니다")
			}
			if err == nil {
				err = os.Remove(path)
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				r.cleanupErr = errors.Join(r.cleanupErr, fmt.Errorf("다운로드 파일 정리 실패 %s: %w", path, err))
			}
		}
	})
	return r.cleanupErr
}

func (r *LectureDownloadRun) track(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	path, err := filepath.Abs(path)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.snapshotErr = errors.Join(r.snapshotErr, err)
		return
	}
	for _, candidate := range []string{path, path + ".part"} {
		if _, seen := r.owned[candidate]; seen {
			continue
		}
		_, err := os.Lstat(candidate)
		r.owned[candidate] = errors.Is(err, os.ErrNotExist)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			r.snapshotErr = errors.Join(r.snapshotErr, fmt.Errorf("정리 대상의 기존 상태 확인 실패 %s: %w", candidate, err))
		}
	}
	// A resumed partial may be renamed to the final path before cancellation.
	// Preserve that destination too; cleanup must not discard pre-run data.
	if !r.owned[path+".part"] {
		r.owned[path] = false
	}
}

type trackingPipelineOperations struct {
	run      *LectureDownloadRun
	delegate lecturePipelineOperations
}

func (o trackingPipelineOperations) DownloadAllLectures(ctx context.Context, opts LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
	progress := opts.OnProgress
	opts.OnProgress = func(value LectureDownloadProgress) {
		// This synchronous callback runs before downloadFile opens the output.
		if value.Stage == "download" {
			o.run.track(value.Path)
		}
		if progress != nil {
			progress(value)
		}
	}
	return o.delegate.DownloadAllLectures(ctx, opts)
}
func (o trackingPipelineOperations) TranscribeDownloadedLectures(ctx context.Context, items []LectureDownloadItem, opts LectureTranscriptOptions) LectureTranscriptResult {
	for _, item := range items {
		o.run.track(TranscriptPathForDownload(item.Path))
	}
	return o.delegate.TranscribeDownloadedLectures(ctx, items, opts)
}
