package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeLectureScreenService struct {
	calls   int
	ctx     context.Context
	options app.LectureListOptions
	err     error
}

func (s *fakeLectureScreenService) LectureList(ctx context.Context, options app.LectureListOptions) ([]app.LectureRow, error) {
	s.calls++
	s.ctx, s.options = ctx, options
	return []app.LectureRow{{ID: "lecture-id"}}, s.err
}
func TestLectureLoaderDefersIOAndPreservesContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	want := errors.New("lecture failed")
	service := &fakeLectureScreenService{err: want}
	cmd := loadLectures(ctx, service, true, true)
	if service.calls != 0 {
		t.Fatal("eager IO")
	}
	msg := cmd().(loadMsg)
	if service.calls != 1 || service.ctx != ctx || !service.options.Refresh || msg.screen != screenLectures || !msg.prefetch || !errors.Is(msg.err, want) || len(msg.lectures) != 1 {
		t.Fatalf("service=%+v msg=%+v", service, msg)
	}
}
func TestLectureChildSelectionAndActions(t *testing.T) {
	m := lectureScreenModel{}
	if !strings.Contains(m.View(80, 24), "온라인 강의가 없습니다") {
		t.Fatal("empty view")
	}
	if _, ok := m.selectedRow(80); ok {
		t.Fatal("empty selection")
	}
	m.Loaded([]app.LectureRow{{ID: "a", CourseName: "first"}, {ID: "b", CourseName: "second"}, {ID: "c", CourseName: "first"}})
	m.Update(tea.KeyMsg{Type: tea.KeyDown}, 80, false)
	row, ok := m.selectedRow(80)
	if !ok || row.ID != "c" {
		t.Fatalf("selected=%+v", row)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight}, 80, false)
	row, ok = m.selectedRow(80)
	if !ok || row.ID != "b" || m.pager.contentCursor != 0 {
		t.Fatalf("selected=%+v", row)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight}, 80, true)
	if m.pager.contentCourse != 1 {
		t.Fatal("loading navigation")
	}
	for _, key := range []string{"a", "ㅁ", "d", "ㅇ"} {
		action, handled := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}, 80, false)
		target := screenAttendConfirm
		if key == "d" || key == "ㅇ" {
			target = screenDownloadSelect
		}
		if !handled || !action.navigate || action.target != target {
			t.Fatalf("%s: %+v", key, action)
		}
	}
	if _, handled := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, 80, true); handled {
		t.Fatal("attend during loading")
	}
	if _, handled := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}, 80, true); !handled {
		t.Fatal("download compatibility during loading")
	}
	for _, msg := range []tea.Msg{tea.WindowSizeMsg{}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}} {
		if _, handled := m.Update(msg, 80, false); handled {
			t.Fatal("global key consumed")
		}
	}
	m.Reset()
	if len(m.lectureRows) != 0 || m.pager.contentCourse != 0 {
		t.Fatal("reset")
	}
}
