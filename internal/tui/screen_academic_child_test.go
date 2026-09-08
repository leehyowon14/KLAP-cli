package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeAcademicScreenService struct {
	calls   int
	ctx     context.Context
	options app.AcademicListOptions
	err     error
}

func (s *fakeAcademicScreenService) AcademicList(ctx context.Context, options app.AcademicListOptions) (app.AcademicListResult, error) {
	s.calls++
	s.ctx, s.options = ctx, options
	return app.AcademicListResult{Year: "2026"}, s.err
}
func TestAcademicLoaderDefersIOAndPreservesContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	want := errors.New("academic failed")
	service := &fakeAcademicScreenService{err: want}
	cmd := loadAcademic(ctx, service, true, true)
	if service.calls != 0 {
		t.Fatal("eager IO")
	}
	msg := cmd().(loadMsg)
	if service.calls != 1 || service.ctx != ctx || !service.options.Refresh || msg.screen != screenAcademic || !msg.prefetch || msg.academic.Year != "2026" || !errors.Is(msg.err, want) {
		t.Fatalf("service=%+v msg=%+v", service, msg)
	}
}
func TestAcademicChildNavigationAndLoadOwnership(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
	result := app.AcademicListResult{Year: "2026", Events: []app.AcademicEvent{
		{Year: "2026", Month: "6월", Date: "2(화)", Title: "first"},
		{Year: "2026", Month: "6월", Date: "3(수)", Title: "second"},
	}}
	m := academicScreenModel{}
	m.Loaded(result, false, now)
	if m.academicMonth != 6 {
		t.Fatal("initial background month")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.academicCursor != 1 {
		t.Fatal("cursor must clamp")
	}
	m.Loaded(result, false, now)
	if m.academicCursor != 1 {
		t.Fatal("background reset")
	}
	m.Loaded(result, true, now)
	if m.academicCursor != 0 {
		t.Fatal("foreground reset")
	}
	m.academicMonth, m.academicCursor = 1, 1
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.academicMonth != 12 || m.academicCursor != 0 {
		t.Fatal("left wrap/reset")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.academicMonth != 1 {
		t.Fatal("right wrap")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.academicCursor != 0 {
		t.Fatal("empty month")
	}
	m.academicMonth = 6
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.academicCursor != 0 {
		t.Fatal("negative cursor")
	}
	for _, msg := range []tea.Msg{tea.WindowSizeMsg{}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}} {
		if _, handled := m.Update(msg); handled {
			t.Fatal("global message consumed")
		}
	}
	m.Reset()
	if m.academicMonth != 0 || len(m.academicResult.Events) != 0 {
		t.Fatal("reset")
	}
}
func TestAcademicRootDoesNotNavigateWhileLoading(t *testing.T) {
	m := model{active: screenAcademic, loading: true, academic: academicScreenModel{academicMonth: 6}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if next.(model).academic.academicMonth != 6 {
		t.Fatal("loading navigation")
	}
	m.loading = false
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if next.(model).academic.academicMonth != 7 {
		t.Fatal("child delegation")
	}
}
