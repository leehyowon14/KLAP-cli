package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type assignmentScreenStub struct {
	calls   int
	ctx     context.Context
	id      string
	options app.AssignmentListOptions
	err     error
}

func (s *assignmentScreenStub) AssignmentList(ctx context.Context, options app.AssignmentListOptions) ([]app.AssignmentRow, error) {
	s.calls++
	s.ctx = ctx
	s.options = options
	return []app.AssignmentRow{{ID: "id", CourseName: "A"}}, s.err
}
func (s *assignmentScreenStub) AssignmentDetail(ctx context.Context, id string, _ app.UserOption) (app.AssignmentDetailResult, error) {
	s.calls++
	s.ctx = ctx
	s.id = id
	return app.AssignmentDetailResult{ID: id}, s.err
}

func TestAssignmentChildOwnsSelectionAndDefersDetailIO(t *testing.T) {
	ctx := context.Background()
	service := &assignmentScreenStub{err: errors.New("detail error")}
	child := assignmentScreenModel{}
	child.Loaded([]app.AssignmentRow{{ID: "A", CourseName: "A"}, {ID: "B", CourseName: "B"}})
	child.Update(tea.KeyMsg{Type: tea.KeyRight}, 80, 20, false, ctx, service)
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, 80, 20, false, ctx, service)
	if !handled || !action.navigate || action.target != screenAssignmentDetail || action.cmd == nil || service.calls != 0 {
		t.Fatal("detail intent did not defer IO")
	}
	msg := action.cmd().(detailMsg)
	if service.calls != 1 || service.ctx != ctx || service.id != "B" || !errors.Is(msg.err, service.err) {
		t.Fatal("selected detail request changed")
	}
	if action, _ := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, 80, 20, true, ctx, service); action.cmd != nil {
		t.Fatal("detail enter repeated request")
	}
}

func TestAssignmentListCommandPreservesRefreshAndPrefetch(t *testing.T) {
	ctx := context.Background()
	service := &assignmentScreenStub{}
	cmd := loadAssignments(ctx, service, true, true)
	if service.calls != 0 {
		t.Fatal("list IO outside command")
	}
	msg := cmd().(loadMsg)
	if service.calls != 1 || service.ctx != ctx || !service.options.Refresh || !msg.prefetch || msg.screen != screenAssignments || len(msg.assignments) != 1 {
		t.Fatal("list contract changed")
	}
}

func TestAssignmentChildDetailScrollResetAndIsolation(t *testing.T) {
	child := assignmentScreenModel{}
	child.DetailLoaded(app.AssignmentDetailResult{ID: "id", Detail: app.AssignmentDetail{Title: "title", ContentText: strings.Repeat("line\n", 40)}})
	for i := 0; i < 100; i++ {
		child.moveDetailCursor(1, 80, 14)
	}
	if child.detailCursor <= 0 {
		t.Fatal("detail did not scroll")
	}
	for i := 0; i < 100; i++ {
		child.moveDetailCursor(-1, 80, 14)
	}
	if child.detailCursor != 0 {
		t.Fatal("detail lower clamp changed")
	}
	if !strings.Contains(child.View(80, 14, true, "status"), "status") {
		t.Fatal("detail status missing")
	}
	other := assignmentScreenModel{}
	if other.detailCursor != 0 || other.assignmentDetail.ID != "" {
		t.Fatal("child state shared")
	}
	child.Reset()
	if child.assignmentRows != nil || child.assignmentDetail.ID != "" || child.pager != (coursePager{}) || child.detailCursor != 0 {
		t.Fatal("reset retained account data")
	}
}

func TestAssignmentChildLoadingAndGlobalKeys(t *testing.T) {
	m := model{active: screenAssignments, loading: true, assignments: assignmentScreenModel{assignmentRows: []app.AssignmentRow{{ID: "id", CourseName: "A"}}}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.(model).active != screenAssignments || cmd != nil {
		t.Fatal("loading allowed detail entry")
	}
	child := assignmentScreenModel{}
	if _, handled := child.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, 80, 20, false, context.Background(), nil); handled {
		t.Fatal("KLAS key consumed")
	}
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, 80, 20, false, context.Background(), nil)
	if !handled || action.navigate || !action.setStatus {
		t.Fatal("empty selection handled incorrectly")
	}
}
