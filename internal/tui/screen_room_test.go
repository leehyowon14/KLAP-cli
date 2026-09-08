package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

func TestRoomFlowSelectsDaysAndPeriods(t *testing.T) {
	m := model{active: screenHome}
	updated, _ := m.startRoomFlow()
	got := updated.(model)
	if got.active != screenRoomDay || len(got.roomDaysSelected) != 0 {
		t.Fatalf("startRoomFlow() active=%v selected=%v", got.active, got.roomDaysSelected)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	if days := got.selectedRoomDays(); len(days) != 2 || days[0] != 1 || days[1] != 2 {
		t.Fatalf("selectedRoomDays() = %v, want [1 2]", days)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(model)
	if got.active != screenRoomPeriod {
		t.Fatalf("active after day enter = %v", got.active)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	if periods := got.selectedRoomPeriods(); len(periods) != 2 || periods[0] != 1 || periods[1] != 3 {
		t.Fatalf("selectedRoomPeriods() = %v, want [1 3]", periods)
	}
}

func TestFormatRoomAvailableResults(t *testing.T) {
	view := formatRoomAvailableResults([]app.RoomAvailableResult{{
		Weekday: 5,
		Periods: []int{1, 3},
		Rooms: []app.RoomAvailableRoom{
			{Room: "새빛관102"},
			{Room: "새빛관103"},
		},
	}})
	if !strings.Contains(view, "금 1, 3교시 비어있음") || !strings.Contains(view, "새빛관102") || !strings.Contains(view, "새빛관103") {
		t.Fatalf("formatRoomAvailableResults() = %q", view)
	}

	empty := formatRoomAvailableResults([]app.RoomAvailableResult{{Weekday: 1, Periods: []int{6, 7, 8}}})
	if !strings.Contains(empty, "조건에 맞는 빈 강의실이 없습니다") {
		t.Fatalf("formatRoomAvailableResults(empty) = %q", empty)
	}
}

func TestRoomAvailableGroupsByBuilding(t *testing.T) {
	results := []app.RoomAvailableResult{{
		Weekday: 2,
		Periods: []int{4, 5, 6},
		Rooms: []app.RoomAvailableRoom{
			{Room: "새빛관103"},
			{Room: "비마관502"},
			{Room: "새빛관101"},
		},
	}}
	groups := roomAvailableBuildingGroups(results)
	if len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
	if groups[0].Building != "비마관" || groups[1].Building != "새빛관" {
		t.Fatalf("building order = %+v", groups)
	}
	if groups[1].Rows[0].Room != "새빛관101" || groups[1].Rows[1].Room != "새빛관103" {
		t.Fatalf("room sort = %+v", groups[1].Rows)
	}
}

func TestRoomResultNavigationWrapsByBuildingAndRows(t *testing.T) {
	m := model{roomResults: []app.RoomAvailableResult{{
		Weekday: 2,
		Periods: []int{4},
		Rooms: []app.RoomAvailableRoom{
			{Room: "비마관502"},
			{Room: "새빛관101"},
			{Room: "새빛관103"},
		},
	}}}
	m.moveRoomResultPage(1)
	if m.roomResultPage != 1 || m.roomResultCursor != 0 {
		t.Fatalf("page=%d cursor=%d", m.roomResultPage, m.roomResultCursor)
	}
	m.moveRoomResultCursor(-1)
	if m.roomResultCursor != 1 {
		t.Fatalf("cursor wrap = %d, want 1", m.roomResultCursor)
	}
	m.moveRoomResultPage(1)
	if m.roomResultPage != 0 || m.roomResultCursor != 0 {
		t.Fatalf("page wrap=%d cursor=%d", m.roomResultPage, m.roomResultCursor)
	}
}

func TestRoomResultViewFitsHeightAndKeepsHeader(t *testing.T) {
	rooms := make([]app.RoomAvailableRoom, 0, 30)
	for index := 1; index <= 30; index++ {
		rooms = append(rooms, app.RoomAvailableRoom{Room: fmt.Sprintf("새빛관%03d", index)})
	}
	m := model{
		active: screenRoomResult,
		width:  96,
		height: 18,
		roomResults: []app.RoomAvailableResult{{
			Weekday: 2,
			Periods: []int{4, 5, 6},
			Rooms:   rooms,
		}},
	}
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > m.height {
		t.Fatalf("room result view lines = %d > %d\n%s", len(lines), m.height, view)
	}
	if !strings.Contains(lines[0], "KLAP") || !strings.Contains(view, "1-7 / 30") {
		t.Fatalf("room result view = %q", view)
	}
}
