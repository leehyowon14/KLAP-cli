package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func batchRow(id, course string, now time.Time) app.LectureRow {
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	return app.LectureRow{ID: id, CourseName: course, Lecture: app.Lecture{ContentID: id, Title: id, StartAt: &start, EndAt: &end}}
}

func TestBatchSelectionAcrossCoursesAndEligibility(t *testing.T) {
	now := time.Now()
	rows := []app.LectureRow{batchRow("a", "A", now), batchRow("b", "B", now), batchRow("expired", "A", now.Add(-2*time.Hour)), batchRow("unknown", "B", now)}
	rows[3].Lecture.StartAt = nil
	m := attendScreenModel{}
	m.StartSelection(rows)
	press := func(k string) childAction {
		return m.updateBatch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}, screenAttendSelect, context.Background(), nil, now)
	}
	press(" ") // current course
	if !m.batch.selected["a"] || m.batch.selected["b"] || m.batch.selected["expired"] {
		t.Fatal(m.batch.selected)
	}
	press("a")
	if !m.batch.selected["b"] || m.batch.selected["unknown"] {
		t.Fatal(m.batch.selected)
	}
	a := m.updateBatch(tea.KeyMsg{Type: tea.KeyEnter}, screenAttendSelect, context.Background(), nil, now)
	if a.target != screenAttendConfirm || a.cmd != nil || len(m.batch.queue) != 2 {
		t.Fatalf("action=%+v queue=%v", a, m.batch.queue)
	}
	press("a")
	a = m.updateBatch(tea.KeyMsg{Type: tea.KeyEnter}, screenAttendSelect, context.Background(), nil, now)
	if a.navigate || m.batch.err == "" {
		t.Fatal("empty selection accepted")
	}
}

type batchAttender struct {
	calls []string
	fail  bool
}

func (s *batchAttender) AttendLecture(ctx context.Context, id string, opts app.LectureAttendOptions) (app.LectureAttendResult, error) {
	s.calls = append(s.calls, id)
	if !opts.RequireEligible {
		return app.LectureAttendResult{}, errors.New("eligibility not requested")
	}
	if s.fail {
		return app.LectureAttendResult{}, errors.New("server failure")
	}
	return app.LectureAttendResult{Lecture: app.LectureRow{ID: id}, Progress: app.LectureProgress{Completed: true, Progress: 100}}, ctx.Err()
}

func TestBatchContinuesAfterFailureAndRejectsLateMessages(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	s := &batchAttender{fail: true}
	m := attendScreenModel{}
	m.StartSelection(nil)
	m.batch.queue = []app.LectureRow{batchRow("a", "A", now), batchRow("b", "B", now)}
	m.nextBatch(ctx, s, now)
	first := m.attendProgress
	done := first.run()()
	m.updateBatch(done, screenAttendProgress, ctx, s, now)
	if m.attendProgress == nil || m.attendProgress.row.ID != "b" || len(s.calls) != 1 {
		t.Fatal("next lecture not queued lazily")
	}
	m.updateBatch(done, screenAttendProgress, ctx, s, now)
	if len(m.batch.queue) != 1 {
		t.Fatal("stale completion changed queue")
	}
	s.fail = false
	m.updateBatch(m.attendProgress.run()(), screenAttendProgress, ctx, s, now)
	if !m.batch.done || len(s.calls) != 2 || !strings.Contains(m.batch.results[0], "실패") || !strings.Contains(m.batch.results[1], "완료") {
		t.Fatalf("%+v %+v", m.batch, s)
	}
}

func TestBatchRechecksWaitingPeriodAndStopsRemaining(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	s := &batchAttender{}
	m := attendScreenModel{}
	m.StartSelection(nil)
	m.batch.queue = []app.LectureRow{batchRow("expired", "A", now.Add(-2*time.Hour)), batchRow("a", "A", now), batchRow("b", "B", now)}
	m.nextBatch(ctx, s, now)
	if len(m.batch.results) != 1 || m.attendProgress.row.ID != "a" {
		t.Fatal("expired queue item started")
	}
	p := m.attendProgress
	m.updateBatch(tea.KeyMsg{Type: tea.KeyEsc}, screenAttendProgress, ctx, s, now)
	if p.ctx.Err() == nil {
		t.Fatal("cancellation not propagated")
	}
	m.updateBatch(lectureAttendDoneMsg{updates: p.updates, err: context.Canceled}, screenAttendProgress, ctx, s, now)
	if !m.batch.done || len(s.calls) != 0 || len(m.batch.results) != 3 {
		t.Fatal("remaining lectures not stopped")
	}
}

func TestMultiAttendRootRoutingAndWidth(t *testing.T) {
	now := time.Now()
	m := model{active: screenLectures, ctx: context.Background(), width: 60, lectures: lectureScreenModel{lectureRows: []app.LectureRow{batchRow(strings.Repeat("긴 강의", 30), "A", now)}}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅡ")})
	got := updated.(model)
	if got.active != screenAttendSelect || cmd != nil || got.attend.batch == nil {
		t.Fatal("multi route")
	}
	for _, line := range strings.Split(got.View(), "\n") {
		if lipgloss.Width(line) > 60 {
			t.Fatalf("overflow: %s", line)
		}
	}
}
