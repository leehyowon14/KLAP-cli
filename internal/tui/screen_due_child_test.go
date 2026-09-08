package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeDueScreenService struct {
	calls   int
	ctx     context.Context
	options app.DueOptions
	err     error
}

func (s *fakeDueScreenService) Due(ctx context.Context, options app.DueOptions) (app.DueResult, error) {
	s.calls++
	s.ctx, s.options = ctx, options
	return app.DueResult{Items: []app.DueItem{{Title: "deadline"}}}, s.err
}

func TestDueLoaderDefersIOAndPreservesContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	want := errors.New("due failed")
	service := &fakeDueScreenService{err: want}
	cmd := loadDue(ctx, service, true, true)
	if service.calls != 0 {
		t.Fatal("eager IO")
	}
	msg := cmd().(loadMsg)
	if service.calls != 1 || service.ctx != ctx || service.options.Days != 14 || !service.options.Refresh {
		t.Fatalf("service = %+v", service)
	}
	if msg.screen != screenDue || !msg.prefetch || !errors.Is(msg.err, want) || len(msg.due.Items) != 1 {
		t.Fatalf("message = %+v", msg)
	}
}

func TestDueChildNavigationAndLoadOwnership(t *testing.T) {
	m := dueScreenModel{duePage: 1, dueCursor: 9}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft}, 80)
	if m.duePage != 0 || m.dueCursor != 0 {
		t.Fatalf("page = %+v", m)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft}, 80)
	if m.duePage != 3 {
		t.Fatal("left did not wrap")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight}, 80)
	if m.duePage != 0 {
		t.Fatal("right did not wrap")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyUp}, 80)
	if m.dueCursor != len(duePageLines(m.dueResult, 0, 80))-1 {
		t.Fatal("cursor did not wrap")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown}, 80)
	if m.dueCursor != 0 {
		t.Fatal("cursor did not wrap back")
	}
	m.duePage, m.dueCursor = 2, 5
	m.Update(tea.KeyMsg{Type: tea.KeyDown}, 80)
	if m.dueCursor != 0 || !strings.Contains(m.View(80, 24, ""), "표시할 일정이 없습니다") {
		t.Fatal("empty page")
	}
	m.dueCursor = 5
	m.Loaded(app.DueResult{}, false)
	if m.duePage != 2 || m.dueCursor != 5 {
		t.Fatal("background load reset selection")
	}
	m.Loaded(app.DueResult{}, true)
	if m.duePage != 0 || m.dueCursor != 0 {
		t.Fatal("foreground load did not reset")
	}
	for _, msg := range []tea.Msg{tea.WindowSizeMsg{}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}} {
		if _, handled := m.Update(msg, 80); handled {
			t.Fatal("consumed global message")
		}
	}
	m.Reset()
	if len(m.dueResult.Items) != 0 || m.duePage != 0 || m.dueCursor != 0 {
		t.Fatal("reset failed")
	}
}

func TestDueRootDoesNotNavigateWhileLoading(t *testing.T) {
	m := model{active: screenDue, loading: true, due: dueScreenModel{duePage: 2}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if next.(model).due.duePage != 2 {
		t.Fatal("loading navigation")
	}
	m.loading = false
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if next.(model).due.duePage != 3 {
		t.Fatal("child navigation not delegated")
	}
}
