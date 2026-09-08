package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeSyncScreenService struct {
	calls     int
	target    screen
	decisions map[string]app.SyncDecision
	err       error
}

func (s *fakeSyncScreenService) SyncDashboard(_ context.Context, o app.DashboardSyncOptions) app.DashboardSyncResult {
	s.calls++
	s.target = screenDashboard
	s.decisions = o.Decisions
	return app.DashboardSyncResult{AssignmentError: s.err}
}
func (s *fakeSyncScreenService) SyncAssignmentReminders(_ context.Context, o app.AssignmentSyncOptions) (app.ReminderSyncResult, error) {
	s.calls++
	s.target = screenAssignments
	s.decisions = o.Decisions
	return app.ReminderSyncResult{}, s.err
}
func (s *fakeSyncScreenService) SyncLectureReminders(_ context.Context, o app.LectureSyncOptions) (app.ReminderSyncResult, error) {
	s.calls++
	s.target = screenLectures
	s.decisions = o.Decisions
	return app.ReminderSyncResult{}, s.err
}
func (s *fakeSyncScreenService) SyncAcademicCalendar(_ context.Context, o app.AcademicSyncOptions) (app.CalendarSyncResult, error) {
	s.calls++
	s.target = screenAcademic
	s.decisions = o.Decisions
	return app.CalendarSyncResult{}, s.err
}
func TestSyncCommandsDeferIOAndSnapshotDecisions(t *testing.T) {
	want := errors.New("sync failed")
	for _, target := range []screen{screenDashboard, screenAssignments, screenLectures, screenAcademic, screenDue} {
		service := &fakeSyncScreenService{err: want}
		decisions := map[string]app.SyncDecision{"id": app.SyncDecisionKeep}
		cmd := syncForScreen(context.Background(), service, target, true, decisions)
		decisions["id"] = app.SyncDecisionApply
		if cmd == nil || service.calls != 0 {
			t.Fatal("eager IO")
		}
		msg := cmd().(syncMsg)
		expected := target
		if target == screenDue {
			expected = screenAcademic
		}
		if service.calls != 1 || service.target != expected || service.decisions["id"] != app.SyncDecisionKeep || !errors.Is(msg.err, want) {
			t.Fatalf("service=%+v msg=%+v", service, msg)
		}
	}
	if syncForScreen(nil, nil, screenDue, false, nil) != nil || syncForScreen(nil, nil, screenNotices, true, nil) != nil {
		t.Fatal("unsupported sync")
	}
}
func TestSyncChildKeepApplyAndSubmit(t *testing.T) {
	service := &fakeSyncScreenService{}
	m := syncScreenModel{}
	m.Start(screenAssignments)
	m.Loaded(syncMsg{conflicts: []app.SyncConflict{{Key: "first"}, {Key: "second"}}})
	if m.syncConflictActions["first"] != app.SyncDecisionKeep || m.syncConflictActions["second"] != app.SyncDecisionKeep || !strings.Contains(m.View(80), "내 수정 유지") {
		t.Fatal("unsafe default")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight}, nil, nil, false)
	if m.syncConflictActions["first"] != app.SyncDecisionApply {
		t.Fatal("toggle apply")
	}
	action, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, nil, nil, false)
	if action.cmd != nil || m.syncConflictCursor != 1 {
		t.Fatal("premature submit")
	}
	action, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}, context.Background(), service, false)
	if !action.navigate || !action.routeOnly || action.target != screenAssignments || action.cmd == nil || m.syncPhase != "syncing" || len(m.syncConflicts) != 0 {
		t.Fatalf("action=%+v state=%+v", action, m)
	}
	m.syncConflictActions["first"] = app.SyncDecisionKeep
	action.cmd()
	if service.decisions["first"] != app.SyncDecisionApply || service.decisions["second"] != app.SyncDecisionKeep {
		t.Fatal("submitted decisions aliased")
	}
}
func TestSyncChildBackAndRepeatedConflicts(t *testing.T) {
	m := syncScreenModel{}
	m.Start(screenAcademic)
	m.Loaded(syncMsg{conflicts: []app.SyncConflict{{Key: "id"}}})
	m.Update(tea.KeyMsg{Type: tea.KeyRight}, nil, nil, false)
	m.Loaded(syncMsg{conflicts: []app.SyncConflict{{Key: "id"}, {Key: "new"}}})
	if m.syncConflictActions["id"] != app.SyncDecisionApply || m.syncConflictActions["new"] != app.SyncDecisionKeep {
		t.Fatal("retry defaults")
	}
	action, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc}, nil, nil, false)
	if action.target != screenAcademic || !action.routeOnly || action.cmd != nil || len(m.syncConflictActions) != 0 {
		t.Fatal("back state")
	}
	root := model{active: screenSyncConflict, sync: m}
	next, cmd := root.applyChildAction(action)
	if next.(model).active != screenAcademic || cmd != nil {
		t.Fatal("back unexpectedly reloads")
	}
	m.syncConflictSource = screenSyncConflict
	action, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}, nil, nil, false)
	if action.target != screenHome {
		t.Fatal("invalid source fallback")
	}
	if _, handled := m.Update(tea.WindowSizeMsg{}, nil, nil, false); handled {
		t.Fatal("global message consumed")
	}
	m.Loaded(syncMsg{err: errors.New("failed")})
	if m.syncPhase != "error" {
		t.Fatal("error phase")
	}
	m.Loaded(syncMsg{})
	if m.syncPhase != "done" {
		t.Fatal("success phase")
	}
	m.ClearPhase()
	if m.syncPhase != "" {
		t.Fatal("timeout phase")
	}
}
