package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

func TestLectureDownloadFormatting(t *testing.T) {
	row := app.LectureRow{
		CourseName: "오픈소스소프트웨어실습",
		Lecture: app.Lecture{
			ModuleTitle: "14주차",
			Title:       "기말 보강 영상",
		},
	}
	if got := lectureDownloadLabel(row); got != "오픈소스소프트웨어실습 · 14주차 · 기말 보강 영상" {
		t.Fatalf("lectureDownloadLabel() = %q", got)
	}
	progress := app.LectureDownloadProgress{Bytes: 512, TotalBytes: 1024}
	if got := lectureDownloadPercent(progress); got != 0.5 {
		t.Fatalf("lectureDownloadPercent() = %f", got)
	}
	if got := formatDownloadBytes(1536); got != "1.5 KB" {
		t.Fatalf("formatDownloadBytes() = %q", got)
	}
	if got := downloadStageLabel("transcribe"); got != "전사중" {
		t.Fatalf("downloadStageLabel(transcribe) = %q", got)
	}
}

func TestLectureDownloadProgressListUsesProgressRows(t *testing.T) {
	rows := []app.LectureRow{
		{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "소개"}},
		{ID: "1:b", CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{ContentID: "b", ModuleTitle: "2주차", Title: "렌더링"}},
	}
	m := lectureDownloadModel{
		width:  96,
		height: 24,
		items:  initialDownloadStatusLines(rows),
	}
	view := m.renderProgressList(96)
	if strings.Contains(view, "QUEUE") {
		t.Fatalf("renderProgressList() leaked QUEUE header: %q", view)
	}
	if !strings.Contains(view, "컴퓨터그래픽스 · 1주차 · 소개") || !strings.Contains(view, "░") {
		t.Fatalf("renderProgressList() missing progress rows: %q", view)
	}
}

func TestLectureDownloadTranscriptStatusPreservesDoneOverwrite(t *testing.T) {
	row := app.LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{ContentID: "a", Title: "소개"}}
	m := lectureDownloadModel{items: initialDownloadStatusLines([]app.LectureRow{row})}
	m.upsertTranscriptStatusLine(app.LectureTranscriptProgress{Lecture: row, Stage: "transcribe", OutputPath: "lecture.txt", Progress: 0.42})
	m.upsertStatusLine(app.LectureDownloadProgress{Lecture: row, Stage: "done", Path: "lecture.mp4", Bytes: 10, TotalBytes: 10})
	if got := m.items[0].status; got != "transcribe" {
		t.Fatalf("status = %q, want transcribe", got)
	}
	if got := itemProgressPercent(m.items[0]); got != 0.42 {
		t.Fatalf("transcript itemProgressPercent() = %f", got)
	}
	if got := itemProgressPercent(downloadStatusLine{status: "download", bytes: 5, total: 10}); got != 0.5 {
		t.Fatalf("download itemProgressPercent() = %f", got)
	}
}

func TestLectureDownloadTranscribedTextOmitsPath(t *testing.T) {
	m := lectureDownloadModel{}
	got := m.itemProgressText(downloadStatusLine{status: "transcribed", path: "/tmp/lecture.txt"})
	if !strings.Contains(got, "전사완료") || strings.Contains(got, "lecture.txt") {
		t.Fatalf("itemProgressText(transcribed) = %q", got)
	}
}

func TestLectureDownloadProgressCursorWraps(t *testing.T) {
	rows := []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
		{ID: "1:b", CourseName: "A", Lecture: app.Lecture{ContentID: "b"}},
	}
	m := lectureDownloadModel{items: initialDownloadStatusLines(rows)}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	got := updated.(lectureDownloadModel)
	if got.cursor != 1 {
		t.Fatalf("up from top cursor = %d, want 1", got.cursor)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(lectureDownloadModel)
	if got.cursor != 0 {
		t.Fatalf("down from bottom cursor = %d, want 0", got.cursor)
	}
}

func TestModelDownloadSelectionIDs(t *testing.T) {
	m := model{download: downloadScreenModel{downloadRows: []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
		{ID: "1:b", CourseName: "B", Lecture: app.Lecture{ContentID: "b"}},
	},
		downloadSelected: map[string]bool{"1:b": true}},
	}
	ids := m.download.selectedDownloadIDs()
	if len(ids) != 1 || ids[0] != "1:b" {
		t.Fatalf("selectedDownloadIDs() = %v", ids)
	}
}

func TestDownloadSelectionGroupsByCourse(t *testing.T) {
	m := model{download: downloadScreenModel{downloadRows: []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
		{ID: "1:b", CourseName: "A", Lecture: app.Lecture{ContentID: "b"}},
		{ID: "2:c", CourseName: "B", Lecture: app.Lecture{ContentID: "c"}},
	},
		downloadSelected: map[string]bool{}},
	}
	groups := m.download.downloadGroups()
	if len(groups) != 2 || groups[0].name != "A" || len(groups[0].rows) != 2 || groups[1].name != "B" {
		t.Fatalf("downloadGroups() = %+v", groups)
	}

	m.download.toggleDownloadCurrent()
	if !m.download.downloadSelected["1:a"] || !m.download.downloadSelected["1:b"] || m.download.downloadSelected["2:c"] {
		t.Fatalf("course toggle selected = %+v", m.download.downloadSelected)
	}
	m.download.toggleDownloadAll()
	if !m.download.downloadSelected["2:c"] {
		t.Fatalf("global toggle should include every course: %+v", m.download.downloadSelected)
	}
	m.download.toggleDownloadAll()
	if m.download.downloadSelected["1:a"] || m.download.downloadSelected["1:b"] || m.download.downloadSelected["2:c"] {
		t.Fatalf("global toggle should clear every course: %+v", m.download.downloadSelected)
	}
}

func TestDownloadRowsStartUnselected(t *testing.T) {
	m := model{active: screenDownloadSelect}
	updated, _ := m.Update(downloadRowsMsg{rows: []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
	}})
	got := updated.(model)
	if len(got.download.downloadSelected) != 0 {
		t.Fatalf("downloadSelected default = %+v, want empty", got.download.downloadSelected)
	}
}

func TestIntegratedDownloadFlowStartsSelectedTranscript(t *testing.T) {
	m := model{
		ctx:     context.Background(),
		service: newTUITestService(t),
		active:  screenDownloadSelect, download: downloadScreenModel{downloadRows: []app.LectureRow{{
			ID:         "course/1:lecture/video",
			CourseName: "운영체제",
			Lecture:    app.Lecture{ContentID: "video", Title: "프로세스"},
		}},
			downloadSelected: map[string]bool{}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.active != screenDownloadConfirm {
		t.Fatalf("active after selection = %v, want screenDownloadConfirm", m.active)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(model)
	if m.active != screenDownloadLanguage || !m.download.downloadTranscribe {
		t.Fatalf("confirm state active=%v transcribe=%t", m.active, m.download.downloadTranscribe)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd == nil || !m.download.preparing {
		t.Fatal("missing preparation command")
	}
	updated, cmd = m.Update(cmd())
	m = updated.(model)
	if cmd == nil || m.active != screenDownloadProgress || m.download.downloadProgress == nil {
		t.Fatalf("progress state active=%v progress nil=%t cmd nil=%t", m.active, m.download.downloadProgress == nil, cmd == nil)
	}
	defer m.download.downloadProgress.cancel()
	request := m.download.downloadProgress.request
	if !request.Transcribe || request.TranscriptLocale != "ko-KR" || len(request.LectureIDs) != 1 || request.LectureIDs[0] != "course/1:lecture/video" {
		t.Fatalf("download request = %+v", request)
	}
}

func TestDownloadSelectIgnoresSelectionWhileLoading(t *testing.T) {
	m := model{
		active:  screenDownloadSelect,
		loading: true, download: downloadScreenModel{downloadRows: []app.LectureRow{{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}}},
			downloadSelected:   map[string]bool{},
			downloadCourse:     0,
			downloadCursor:     1,
			downloadTranscribe: false},
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got := updated.(model)
	if got.download.downloadSelected["1:a"] {
		t.Fatalf("loading selection changed: %+v", got.download.downloadSelected)
	}
	if got.active != screenDownloadSelect {
		t.Fatalf("active = %v, want screenDownloadSelect", got.active)
	}
}

func TestDownloadSelectionLeftRightChangesCourse(t *testing.T) {
	m := model{
		active: screenDownloadSelect, download: downloadScreenModel{downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
			{ID: "2:b", CourseName: "B", Lecture: app.Lecture{ContentID: "b"}},
		},
			downloadSelected: map[string]bool{}},
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := updated.(model)
	if got.download.downloadCourse != 1 || got.download.downloadCursor != 0 {
		t.Fatalf("right key course=%d cursor=%d", got.download.downloadCourse, got.download.downloadCursor)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = updated.(model)
	if got.download.downloadCourse != 0 || got.download.downloadCursor != 0 {
		t.Fatalf("left key course=%d cursor=%d", got.download.downloadCourse, got.download.downloadCursor)
	}
}

func TestDownloadSelectCursorWraps(t *testing.T) {
	m := model{
		active: screenDownloadSelect, download: downloadScreenModel{downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "A", Lecture: app.Lecture{ContentID: "b"}},
		},
			downloadSelected: map[string]bool{}},
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	got := updated.(model)
	if got.download.downloadCursor != 2 {
		t.Fatalf("up from top downloadCursor = %d, want 2", got.download.downloadCursor)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	if got.download.downloadCursor != 0 {
		t.Fatalf("down from bottom downloadCursor = %d, want 0", got.download.downloadCursor)
	}
}

func TestTranscriptLanguageDefaultsToKoreanWithoutWarning(t *testing.T) {
	m := model{download: downloadScreenModel{downloadLanguage: 0}}
	if got := m.download.selectedTranscriptLocale(); got != "ko-KR" {
		t.Fatalf("selectedTranscriptLocale() = %q", got)
	}
	view := m.download.renderDownloadLanguageView(96)
	if !strings.Contains(view, "ko-KR") || !strings.Contains(view, "전사 주 언어를 선택하세요.") || strings.Contains(view, "language switching(code switching)") {
		t.Fatalf("renderDownloadLanguageView() = %q", view)
	}
}
