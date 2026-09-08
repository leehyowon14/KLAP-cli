package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeDownloadScreenService struct {
	reads   int
	builds  int
	options app.LectureDownloadPipelineOptions
	err     error
}

func (s *fakeDownloadScreenService) LectureList(context.Context, app.LectureListOptions) ([]app.LectureRow, error) {
	return nil, s.err
}
func (s *fakeDownloadScreenService) DownloadSettings() (app.DownloadSettings, error) {
	s.reads++
	return app.DownloadSettings{Concurrency: 2}, s.err
}
func (s *fakeDownloadScreenService) TranscriptSettings() (app.TranscriptSettings, error) {
	s.reads++
	return app.TranscriptSettings{Concurrency: 3}, s.err
}
func (s *fakeDownloadScreenService) NewLectureDownloadRun(_ context.Context, options app.LectureDownloadPipelineOptions) *app.LectureDownloadRun {
	s.builds++
	s.options = options
	return &app.LectureDownloadRun{}
}
func TestDownloadChildPreparationDefersSettingsAndSnapshotsSelection(t *testing.T) {
	service := &fakeDownloadScreenService{}
	m := downloadScreenModel{downloadRows: []app.LectureRow{{ID: "id", Lecture: app.Lecture{ContentID: "video"}}}, downloadSelected: map[string]bool{"id": true}, downloadTranscribe: true}
	action := m.prepare(context.Background(), service)
	if service.reads != 0 || service.builds != 0 || !m.preparing {
		t.Fatal("eager preparation")
	}
	duplicate, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenDownloadConfirm, false, context.Background(), service)
	if duplicate.cmd != nil {
		t.Fatal("duplicate preparation")
	}
	m.downloadSelected["id"] = false
	msg := action.cmd().(downloadPreparedMsg)
	defer msg.progress.cancel()
	if service.reads != 2 || service.builds != 1 || len(msg.progress.request.LectureIDs) != 1 || msg.progress.request.LectureIDs[0] != "id" || msg.progress.request.Concurrency != 2 || msg.progress.request.TranscriptConcurrency != 3 || !msg.progress.request.Transcribe {
		t.Fatalf("request=%+v service=%+v", msg.progress.request, service)
	}
	action, handled := m.Lifecycle(msg)
	if !handled || m.preparing || !action.navigate || action.target != screenDownloadProgress || action.cmd == nil || m.downloadProgress == nil {
		t.Fatal("prepared transition")
	}
	// The returned Init command is deliberately not run: this test never starts a download.
}
func TestDownloadChildPreparationFailureRetainsConfirmation(t *testing.T) {
	want := errors.New("settings failed")
	service := &fakeDownloadScreenService{err: want}
	m := downloadScreenModel{}
	action := m.prepare(context.Background(), service)
	action, handled := m.Lifecycle(action.cmd())
	if !handled || !errors.Is(action.err, want) || action.navigate || m.preparing || service.builds != 0 {
		t.Fatal("preparation failure")
	}
	action = m.prepare(context.Background(), service)
	if action.cmd == nil {
		t.Fatal("retry unavailable")
	}
}
func TestDownloadChildIgnoresStalePreparation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := downloadScreenModel{preparing: true, generation: 2}
	action, handled := m.Lifecycle(downloadPreparedMsg{generation: 1, progress: &lectureDownloadModel{ctx: ctx, cancel: cancel}})
	if !handled || action.navigate || m.downloadProgress != nil || !m.preparing || ctx.Err() == nil {
		t.Fatal("stale preparation not discarded")
	}
}
func TestDownloadChildBackReloadsLectures(t *testing.T) {
	root := model{active: screenDownloadSelect}
	next, cmd := root.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(model)
	if got.active != screenLectures || !got.loading || cmd == nil {
		t.Fatal("selection back must load lectures")
	}
}

func TestDownloadPreparationStatusAndTranscriptHeader(t *testing.T) {
	m := downloadScreenModel{preparing: true}
	for _, route := range []screen{screenDownloadConfirm, screenDownloadLanguage} {
		view := m.View(80, 24, route, false, nil)
		if !strings.Contains(view, "Transcript") || !strings.Contains(view, "준비하고 있습니다") {
			t.Fatal("missing header or preparation status")
		}
		want := errors.New("settings unavailable")
		if !strings.Contains(m.View(80, 24, route, false, want), want.Error()) {
			t.Fatal("preparation error hidden")
		}
	}
}
