package tui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeRoomScreenService struct {
	calls   int
	ctx     context.Context
	options app.RoomAvailabilityForDaysOptions
	err     error
}

func (s *fakeRoomScreenService) RoomAvailabilityForDays(ctx context.Context, options app.RoomAvailabilityForDaysOptions) ([]app.RoomAvailableResult, error) {
	s.calls++
	s.ctx, s.options = ctx, options
	return []app.RoomAvailableResult{{Weekday: 2}}, s.err
}
func TestRoomChildLoaderSnapshotsSelectionAndDefersIO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	want := errors.New("room failed")
	service := &fakeRoomScreenService{err: want}
	m := roomScreenModel{roomDaysSelected: map[int]bool{5: true, 2: true}, roomPeriodsSelected: map[int]bool{8: true, 1: true}}
	cmd := m.Load(ctx, service, true)
	m.roomDaysSelected[2] = false
	m.roomPeriodsSelected[1] = false
	if service.calls != 0 {
		t.Fatal("eager IO")
	}
	msg := cmd().(roomAvailableResultsMsg)
	if service.calls != 1 || service.ctx != ctx || !service.options.Refresh || !reflect.DeepEqual(service.options.Days, []int{2, 5}) || !reflect.DeepEqual(service.options.Periods, []int{1, 8}) || !errors.Is(msg.err, want) || len(msg.results) != 1 {
		t.Fatalf("service=%+v msg=%+v", service, msg)
	}
}
func TestRoomChildValidationAndBackNavigation(t *testing.T) {
	root := model{active: screenHome}
	next, _ := root.applyChildAction(childAction{navigate: true, target: screenRoomDay})
	root = next.(model)
	next, cmd := root.Update(tea.KeyMsg{Type: tea.KeyEnter})
	root = next.(model)
	if root.err == nil || root.active != screenRoomDay || cmd != nil {
		t.Fatal("missing day validation")
	}
	next, cmd = root.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(model).active != screenHome || next.(model).loading || cmd != nil {
		t.Fatal("Home back must not load")
	}
	root.room.roomDaysSelected[1] = true
	next, _ = root.Update(tea.KeyMsg{Type: tea.KeyEnter})
	root = next.(model)
	next, cmd = root.Update(tea.KeyMsg{Type: tea.KeyEnter})
	root = next.(model)
	if root.err == nil || root.active != screenRoomPeriod || cmd != nil {
		t.Fatal("missing period validation")
	}
	root.room.roomPeriodsSelected[3] = true
	next, cmd = root.Update(tea.KeyMsg{Type: tea.KeyEnter})
	root = next.(model)
	if root.active != screenRoomResult || !root.loading || cmd == nil {
		t.Fatal("result entry")
	}
	next, _ = root.Update(tea.KeyMsg{Type: tea.KeyEsc})
	root = next.(model)
	if root.active != screenRoomPeriod || root.loading || !root.room.roomDaysSelected[1] || !root.room.roomPeriodsSelected[3] {
		t.Fatal("result back selection lost")
	}
	next, _ = root.Update(roomAvailableResultsMsg{results: []app.RoomAvailableResult{{Weekday: 5}}})
	if len(next.(model).room.roomResults) != 0 {
		t.Fatal("inactive stale result applied")
	}
	next, _ = root.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(model).active != screenRoomDay || !next.(model).room.roomPeriodsSelected[3] {
		t.Fatal("period back reset flow")
	}
}
func TestRoomChildSelectionBoundsAndReset(t *testing.T) {
	m := roomScreenModel{}
	m.Start()
	for i := 0; i < 20; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown}, screenRoomDay, false, nil, nil)
	}
	if m.roomDayCursor != len(roomDayOptions)-1 {
		t.Fatal("day cursor bounds")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, screenRoomDay, false, nil, nil)
	if len(m.selectedRoomDays()) != len(roomDayOptions) {
		t.Fatal("select all days")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, screenRoomDay, false, nil, nil)
	if len(m.selectedRoomDays()) != 0 {
		t.Fatal("clear all days")
	}
	m.roomPeriodCursor = 7
	m.Update(tea.KeyMsg{Type: tea.KeyDown}, screenRoomPeriod, false, nil, nil)
	if m.roomPeriodCursor != 7 {
		t.Fatal("period cursor bounds")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, screenRoomPeriod, false, nil, nil)
	if len(m.selectedRoomPeriods()) != 8 {
		t.Fatal("all periods")
	}
	m.Start()
	if len(m.selectedRoomPeriods()) != 0 || m.roomDayCursor != 0 {
		t.Fatal("restart reset")
	}
	if _, handled := m.Update(tea.WindowSizeMsg{}, screenRoomDay, false, nil, nil); handled {
		t.Fatal("global message")
	}
}
