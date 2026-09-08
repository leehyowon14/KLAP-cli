package app

import (
	"context"
	"path/filepath"
	"testing"
)

type fakeDownloader struct {
	download func(context.Context, string, string, bool, func(int64, int64)) (int64, error)
}

func (*fakeDownloader) VideoPath(root, course string, lecture LectureFileName, media string, order int) string {
	return filepath.Join(root, "fixture.mp4")
}
func (f *fakeDownloader) DownloadFile(ctx context.Context, url, path string, keep bool, progress func(int64, int64)) (int64, error) {
	return f.download(ctx, url, path, keep, progress)
}

func TestDependenciesRequireDownloader(t *testing.T) {
	for _, downloader := range []Downloader{nil, (*fakeDownloader)(nil)} {
		deps := testDependencies(t)
		deps.Downloader = downloader
		if s, err := NewService(deps); s != nil || err == nil {
			t.Fatalf("service=%v err=%v", s, err)
		}
	}
}

func TestDownloadTaskUsesInjectedDownloader(t *testing.T) {
	ctx := context.Background()
	media := &fakeMediaResolver{resolve: func(context.Context, string) (string, error) { return "https://example.test/video.mp4", nil }}
	calls := 0
	var stages []LectureTransferStage
	downloader := &fakeDownloader{download: func(got context.Context, url, path string, keep bool, progress func(int64, int64)) (int64, error) {
		calls++
		if got != ctx || url != "https://example.test/video.mp4" || path == "" || !keep {
			t.Fatalf("args: %v %q %q %v", got, url, path, keep)
		}
		if len(stages) != 2 || stages[1] != LectureStageDownload {
			t.Fatalf("write started before tracking event: %v", stages)
		}
		progress(8, 8)
		return 8, nil
	}}
	result := downloadLectureTask(ctx, media, downloader, t.TempDir(), true, 1, 1, Course{Name: "Course"}, LectureRow{Lecture: Lecture{ContentID: "id", Title: "Title"}}, 1, func(p LectureDownloadProgress) { stages = append(stages, p.Stage) })
	if calls != 1 || result.Err != nil || result.Bytes != 8 || stages[len(stages)-1] != LectureStageDone {
		t.Fatalf("result=%+v calls=%d stages=%v", result, calls, stages)
	}
}

func TestLectureFileNameNormalizesWeek(t *testing.T) {
	for _, tc := range []struct {
		lecture Lecture
		week    int
	}{
		{Lecture{WeekNo: "14", ModuleTitle: "1주차"}, 14},
		{Lecture{ModuleTitle: "1주차: 소개"}, 1},
		{Lecture{WeekNo: "bad"}, 0},
	} {
		got := lectureFileName(tc.lecture)
		if got.Week != tc.week || got.ModuleTitle != tc.lecture.ModuleTitle {
			t.Fatalf("got=%+v", got)
		}
	}
}

func (*fakeDownloader) Status(dir string) (DownloadStatusResult, error) {
	return DownloadStatusResult{Dir: dir}, nil
}
func TestDownloadStatusUsesInjectedAdapter(t *testing.T) {
	service := &Service{downloader: &fakeDownloader{}}
	got, err := service.DownloadStatus("explicit")
	if err != nil || got.Dir != "explicit" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
