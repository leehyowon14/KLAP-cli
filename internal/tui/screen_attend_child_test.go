package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func TestAttendChildStartsOnlyAfterConfirmationAndRevalidates(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local)
	end := now.Add(time.Minute)
	row := app.LectureRow{ID: "id", Lecture: app.Lecture{ContentID: "video", Progress: "25", EndAt: &end}}
	m := attendScreenModel{}
	if err := m.Start(row, false, now); err == nil {
		t.Fatal("missing selection accepted")
	}
	if err := m.Start(row, true, now); err != nil {
		t.Fatal(err)
	}
	service := &fakeLectureAttender{}
	action, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenAttendConfirm, context.Background(), service, now)
	if action.target != screenAttendProgress || action.cmd == nil || m.attendProgress == nil || service.id != "" {
		t.Fatal("start was eager or missing")
	}
	m.attendProgress.cancel()
	m.attendProgress = nil
	action, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenAttendConfirm, context.Background(), service, end.Add(time.Second))
	if action.err == nil || action.target != screenLectures || action.cmd != nil || m.attendProgress != nil {
		t.Fatal("expired confirmation was not revalidated")
	}
}
func TestAttendChildConfirmationBackPreservesLectureList(t *testing.T) {
	m := attendScreenModel{attendRow: app.LectureRow{ID: "id"}}
	action, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}, screenAttendConfirm, nil, nil, time.Time{})
	root := model{active: screenAttendConfirm, attend: m, lectures: lectureScreenModel{pager: coursePager{contentCourse: 2}}}
	next, cmd := root.applyChildAction(action)
	if next.(model).active != screenLectures || next.(model).lectures.pager.contentCourse != 2 || cmd != nil || m.attendRow.ID != "" {
		t.Fatal("confirmation back")
	}
}
func TestAttendChildCancelAndCompletionNavigation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := attendScreenModel{attendProgress: &lectureAttendModel{ctx: ctx, cancel: cancel, updates: make(chan tea.Msg, 1)}}
	action, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc}, screenAttendProgress, nil, nil, time.Time{})
	if action.navigate || !m.attendProgress.canceling || ctx.Err() == nil {
		t.Fatal("escape must cancel before leaving")
	}
	m.attendProgress.done = true
	action, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}, screenAttendProgress, nil, nil, time.Time{})
	if !action.reload || !action.refresh || action.target != screenLectures || m.attendProgress != nil {
		t.Fatal("completed return must refresh")
	}
	ctx, cancel = context.WithCancel(context.Background())
	m.attendProgress = &lectureAttendModel{ctx: ctx, cancel: cancel}
	action, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}, screenAttendProgress, nil, nil, time.Time{})
	if action.target != screenHome || ctx.Err() == nil || m.attendProgress != nil {
		t.Fatal("home must cancel and clear")
	}
}
