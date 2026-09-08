package app

import (
	"context"
	"errors"
	"testing"
)

type fakeMediaResolver struct {
	resolve func(context.Context, string) (string, error)
}

func (f *fakeMediaResolver) ResolveLectureMediaURL(ctx context.Context, id string) (string, error) {
	return f.resolve(ctx, id)
}

func TestDependenciesRequireMedia(t *testing.T) {
	for _, media := range []MediaResolver{nil, (*fakeMediaResolver)(nil)} {
		deps := testDependencies(t)
		deps.Media = media
		if s, err := NewService(deps); s != nil || err == nil {
			t.Fatalf("service=%v, error=%v", s, err)
		}
	}
}

func TestDownloadTaskUsesInjectedMediaResolver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	media := &fakeMediaResolver{resolve: func(got context.Context, id string) (string, error) {
		calls++
		if got != ctx || id != "content" {
			t.Fatalf("context=%v, id=%q", got, id)
		}
		return "", got.Err()
	}}
	row := LectureRow{Lecture: Lecture{ContentID: "content"}}
	var stages []LectureTransferStage
	result := downloadLectureTask(ctx, media, nil, t.TempDir(), false, 1, 1, Course{}, row, 1, func(p LectureDownloadProgress) { stages = append(stages, p.Stage) })
	if calls != 1 || !errors.Is(result.Err, context.Canceled) || len(stages) != 2 || stages[0] != LectureStageResolve || stages[1] != LectureStageError {
		t.Fatalf("calls=%d result=%+v stages=%v", calls, result, stages)
	}
	result = downloadLectureTask(ctx, media, nil, t.TempDir(), false, 1, 1, Course{}, LectureRow{}, 1, nil)
	if calls != 1 || !result.Skipped || result.Err == nil {
		t.Fatalf("empty content calls=%d result=%+v", calls, result)
	}
}
