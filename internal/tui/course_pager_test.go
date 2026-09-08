package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
	"time"
)

func TestCoursePagedContentGroupsByCourse(t *testing.T) {
	m := model{
		active: screenLectures,
		width:  96,
		lectureRows: []app.LectureRow{
			{CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{Title: "렌더링", ModuleTitle: "1주차", Progress: "20", ContentID: "a"}},
			{CourseName: "오픈소스소프트웨어실습", Lecture: app.Lecture{Title: "Git", ModuleTitle: "2주차", Progress: "0", ContentID: "b"}},
			{CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{Title: "셰이딩", ModuleTitle: "3주차", Progress: "30", ContentID: "c"}},
		},
	}

	groups := m.contentGroups(96)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if groups[0].name != "컴퓨터그래픽스" || len(groups[0].lines) != 2 {
		t.Fatalf("first group = %#v", groups[0])
	}
	if groups[1].name != "오픈소스소프트웨어실습" || len(groups[1].lines) != 1 {
		t.Fatalf("second group = %#v", groups[1])
	}
}

func TestCoursePagedCursorAndCourseWrap(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active: screenAssignments,
		width:  96,
		assignmentRows: []app.AssignmentRow{
			{CourseName: "컴퓨터그래픽스", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{CourseName: "컴퓨터그래픽스", Assignment: app.Assignment{Title: "과제2", DueAt: &due}},
			{CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "기말", DueAt: &due}},
		},
	}

	m.moveContentCursor(-1)
	if m.contentCursor != 1 {
		t.Fatalf("cursor after wrapping up = %d, want 1", m.contentCursor)
	}
	m.moveContentCursor(1)
	if m.contentCursor != 0 {
		t.Fatalf("cursor after wrapping down = %d, want 0", m.contentCursor)
	}
	m.moveContentCourse(-1)
	if m.contentCourse != 1 || m.contentCursor != 0 {
		t.Fatalf("course/cursor after wrapping left = %d/%d, want 1/0", m.contentCourse, m.contentCursor)
	}
	m.moveContentCourse(1)
	if m.contentCourse != 0 || m.contentCursor != 0 {
		t.Fatalf("course/cursor after wrapping right = %d/%d, want 0/0", m.contentCourse, m.contentCursor)
	}
}

func TestCoursePagedRenderShowsCurrentCourseOnly(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active:        screenAssignments,
		width:         96,
		height:        24,
		contentCourse: 1,
		assignmentRows: []app.AssignmentRow{
			{CourseName: "컴퓨터그래픽스", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "기말고사 대체 과제", DueAt: &due}},
		},
	}

	view := m.renderCoursePagedPanel(96)
	if !strings.Contains(view, "오픈소스소프트웨어실습") || !strings.Contains(view, "기말고사 대체 과제") {
		t.Fatalf("renderCoursePagedPanel() missing current course: %q", view)
	}
	if strings.Contains(view, "과제1") {
		t.Fatalf("renderCoursePagedPanel() leaked other course: %q", view)
	}
}

func TestCoursePagedSelectionUsesCurrentCourseAndCursor(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active:        screenAssignments,
		width:         96,
		contentCourse: 1,
		contentCursor: 1,
		assignmentRows: []app.AssignmentRow{
			{ID: "1:1", CourseName: "컴퓨터그래픽스", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{ID: "2:1", CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "과제A", DueAt: &due}},
			{ID: "2:2", CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "과제B", DueAt: &due}},
		},
	}

	row, ok := m.selectedAssignmentRow()
	if !ok || row.ID != "2:2" {
		t.Fatalf("selectedAssignmentRow() = %+v, %t", row, ok)
	}
}

func TestKlasShortcutDoesNotMoveListCursor(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active:        screenAssignments,
		width:         96,
		contentCursor: 1,
		assignmentRows: []app.AssignmentRow{
			{ID: "1:1", CourseName: "컴퓨터그래픽스", DetailURL: "https://klas.example/1", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{ID: "1:2", CourseName: "컴퓨터그래픽스", DetailURL: "https://klas.example/2", Assignment: app.Assignment{Title: "과제2", DueAt: &due}},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got := updated.(model)
	if got.contentCursor != 1 {
		t.Fatalf("contentCursor after k = %d, want 1", got.contentCursor)
	}
}
