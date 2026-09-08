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
	if m.pager.contentCursor != 1 {
		t.Fatalf("cursor after wrapping up = %d, want 1", m.pager.contentCursor)
	}
	m.moveContentCursor(1)
	if m.pager.contentCursor != 0 {
		t.Fatalf("cursor after wrapping down = %d, want 0", m.pager.contentCursor)
	}
	m.moveContentCourse(-1)
	if m.pager.contentCourse != 1 || m.pager.contentCursor != 0 {
		t.Fatalf("course/cursor after wrapping left = %d/%d, want 1/0", m.pager.contentCourse, m.pager.contentCursor)
	}
	m.moveContentCourse(1)
	if m.pager.contentCourse != 0 || m.pager.contentCursor != 0 {
		t.Fatalf("course/cursor after wrapping right = %d/%d, want 0/0", m.pager.contentCourse, m.pager.contentCursor)
	}
}

func TestCoursePagedRenderShowsCurrentCourseOnly(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active: screenAssignments,
		width:  96,
		height: 24,

		assignmentRows: []app.AssignmentRow{
			{CourseName: "컴퓨터그래픽스", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "기말고사 대체 과제", DueAt: &due}},
		},
		pager: coursePager{contentCourse: 1},
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
		active: screenAssignments,
		width:  96,

		assignmentRows: []app.AssignmentRow{
			{ID: "1:1", CourseName: "컴퓨터그래픽스", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{ID: "2:1", CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "과제A", DueAt: &due}},
			{ID: "2:2", CourseName: "오픈소스소프트웨어실습", Assignment: app.Assignment{Title: "과제B", DueAt: &due}},
		},
		pager: coursePager{contentCourse: 1,
			contentCursor: 1},
	}

	row, ok := m.selectedAssignmentRow()
	if !ok || row.ID != "2:2" {
		t.Fatalf("selectedAssignmentRow() = %+v, %t", row, ok)
	}
}

func TestKlasShortcutDoesNotMoveListCursor(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active: screenAssignments,
		width:  96,

		assignmentRows: []app.AssignmentRow{
			{ID: "1:1", CourseName: "컴퓨터그래픽스", DetailURL: "https://klas.example/1", Assignment: app.Assignment{Title: "과제1", DueAt: &due}},
			{ID: "1:2", CourseName: "컴퓨터그래픽스", DetailURL: "https://klas.example/2", Assignment: app.Assignment{Title: "과제2", DueAt: &due}},
		},
		pager: coursePager{contentCursor: 1},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got := updated.(model)
	if got.pager.contentCursor != 1 {
		t.Fatalf("contentCursor after k = %d, want 1", got.pager.contentCursor)
	}
}

func TestCoursePagerOwnsNavigationAndEmptyState(t *testing.T) {
	groups := []contentCourseGroup{{name: "A", lines: []string{"one", "two"}}, {name: "B", lines: []string{"three"}}}
	pager := coursePager{}
	pager.Update(tea.KeyMsg{Type: tea.KeyUp}, groups)
	if pager.contentCursor != 1 {
		t.Fatal("cursor wrap changed")
	}
	pager.Update(tea.KeyMsg{Type: tea.KeyRight}, groups)
	if pager.contentCourse != 1 || pager.contentCursor != 0 {
		t.Fatal("course change did not reset cursor")
	}
	pager.Update(tea.KeyMsg{Type: tea.KeyRight}, groups)
	if pager.contentCourse != 0 {
		t.Fatal("course wrap changed")
	}
	if pager.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, groups) {
		t.Fatal("pager consumed KLAS shortcut")
	}
	if pager.Update(tea.WindowSizeMsg{}, groups) {
		t.Fatal("pager consumed global window")
	}
	pager.Update(tea.KeyMsg{Type: tea.KeyLeft}, nil)
	if pager != (coursePager{}) {
		t.Fatal("empty state not reset")
	}
	if !strings.Contains(pager.View(nil, 20, screenAssignments), "과제가 없습니다") {
		t.Fatal("empty label changed")
	}
	pager.contentCourse = 99
	pager.contentCursor = 99
	if !strings.Contains(pager.View(groups, 20, screenNotices), "three") {
		t.Fatal("out of range view not clamped")
	}
}
