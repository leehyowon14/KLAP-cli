package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type noticeScreenStub struct {
	calls   int
	ctx     context.Context
	id      string
	options app.NoticeListOptions
	err     error
}

func (s *noticeScreenStub) NoticeList(ctx context.Context, options app.NoticeListOptions) ([]app.NoticeRow, error) {
	s.calls++
	s.ctx = ctx
	s.options = options
	return []app.NoticeRow{{ID: "id", CourseName: "A"}}, s.err
}
func (s *noticeScreenStub) NoticeDetail(ctx context.Context, id string, _ app.UserOption) (app.NoticeDetailResult, error) {
	s.calls++
	s.ctx = ctx
	s.id = id
	return app.NoticeDetailResult{ID: id}, s.err
}

func TestNoticeChildOwnsSelectionAndDefersDetailIO(t *testing.T) {
	ctx := context.Background()
	service := &noticeScreenStub{err: errors.New("detail error")}
	child := noticeScreenModel{}
	child.Loaded([]app.NoticeRow{{ID: "A", CourseName: "A"}, {ID: "B", CourseName: "B"}})
	child.Update(tea.KeyMsg{Type: tea.KeyRight}, 80, 20, false, ctx, service)
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, 80, 20, false, ctx, service)
	if !handled || !action.navigate || action.target != screenNoticeDetail || action.cmd == nil || service.calls != 0 {
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

func TestNoticeListCommandPreservesRefreshAndPrefetch(t *testing.T) {
	ctx := context.Background()
	service := &noticeScreenStub{}
	cmd := loadNotices(ctx, service, true, true)
	if service.calls != 0 {
		t.Fatal("list IO outside command")
	}
	msg := cmd().(loadMsg)
	if service.calls != 1 || service.ctx != ctx || !service.options.Refresh || !msg.prefetch || msg.screen != screenNotices || len(msg.notices) != 1 {
		t.Fatal("list contract changed")
	}
}

func TestNoticeChildDetailScrollResetAndIsolation(t *testing.T) {
	child := noticeScreenModel{}
	child.DetailLoaded(app.NoticeDetailResult{ID: "id", Detail: app.NoticeDetail{Title: "title", ContentText: strings.Repeat("line\n", 40)}})
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
	other := noticeScreenModel{}
	if other.detailCursor != 0 || other.noticeDetail.ID != "" {
		t.Fatal("child state shared")
	}
	child.Reset()
	if child.noticeRows != nil || child.noticeDetail.ID != "" || child.pager != (coursePager{}) || child.detailCursor != 0 {
		t.Fatal("reset retained account data")
	}
}

func TestNoticeChildLoadingAndGlobalKeys(t *testing.T) {
	m := model{active: screenNotices, loading: true, notices: noticeScreenModel{noticeRows: []app.NoticeRow{{ID: "id", CourseName: "A"}}}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.(model).active != screenNotices || cmd != nil {
		t.Fatal("loading allowed detail entry")
	}
	child := noticeScreenModel{}
	if _, handled := child.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}, 80, 20, false, context.Background(), nil); handled {
		t.Fatal("KLAS key consumed")
	}
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, 80, 20, false, context.Background(), nil)
	if !handled || action.navigate || !action.setStatus {
		t.Fatal("empty selection handled incorrectly")
	}
}

func TestNoticeChildKeepsAssignmentSelectionIndependent(t *testing.T) {
	m := model{active: screenAssignments, loadedScreens: map[screen]bool{}, loadingScreens: map[screen]bool{}, screenErrors: map[screen]error{}}
	m.assignments.Loaded([]app.AssignmentRow{{ID: "A", CourseName: "A"}, {ID: "B", CourseName: "B"}})
	m.assignments.pager.contentCourse = 1
	m.applyLoadMsg(loadMsg{screen: screenNotices, prefetch: true, notices: []app.NoticeRow{{ID: "notice", CourseName: "notice"}}})
	if m.assignments.pager.contentCourse != 1 || m.active != screenAssignments {
		t.Fatal("background notice load clobbered assignment")
	}
	updated, cmd := m.enterScreen(screenNotices)
	got := updated.(model)
	if cmd != nil || got.activePager().contentCourse != 0 || got.assignments.pager.contentCourse != 1 {
		t.Fatal("notice route shared assignment pager")
	}
}
