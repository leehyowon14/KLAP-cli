package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

func TestHomeViewShowsMenu(t *testing.T) {
	m := model{
		width: 96,
		ctx:   context.Background(),
		home: homeModel{menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
			{title: "Due", help: "다가오는 데드라인", screen: screenDue},
		}},
	}

	view := m.View()
	if !strings.Contains(view, "https://github.com/leehyowon14/KLAP-cli") || !strings.Contains(view, "Beyond KLAS, in your terminal.") || !strings.Contains(view, "Kwangwoon Learning Automation Project") || !strings.Contains(view, "› 1.") || !strings.Contains(view, "Dashboard") || !strings.Contains(view, "_  __") {
		t.Fatalf("View() = %q", view)
	}
}

func TestHomeViewLinesFitWidth(t *testing.T) {
	m := model{
		width: 96,
		home: homeModel{menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
		}},
	}
	for index, line := range strings.Split(m.View(), "\n") {
		if width := lipgloss.Width(line); width > m.width {
			t.Fatalf("home line %d width = %d > %d: %q", index, width, m.width, line)
		}
	}
}

func TestHomeNavigation(t *testing.T) {
	m := model{
		ctx: context.Background(),
		home: homeModel{menu: []menuItem{
			{title: "Dashboard", screen: screenDashboard},
			{title: "Due", screen: screenDue},
		}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(model)
	if got.home.cursor != 1 {
		t.Fatalf("cursor after down = %d", got.home.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyUp})
	got = updated.(model)
	if got.home.cursor != 0 {
		t.Fatalf("cursor after up = %d", got.home.cursor)
	}
}

func TestHomeNavigationAcceptsKoreanKeyboardKeys(t *testing.T) {
	m := model{
		ctx: context.Background(),
		home: homeModel{menu: []menuItem{
			{title: "Dashboard", screen: screenDashboard},
			{title: "Due", screen: screenDue},
		}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅓ")})
	got := updated.(model)
	if got.home.cursor != 1 {
		t.Fatalf("cursor after korean j key = %d", got.home.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅏ")})
	got = updated.(model)
	if got.home.cursor != 0 {
		t.Fatalf("cursor after korean k key = %d", got.home.cursor)
	}
}

func TestHomeChildNavigationIntentAndBounds(t *testing.T) {
	child := newHomeModel()
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled || !action.navigate || action.target != screenDashboard {
		t.Fatalf("action=%+v", action)
	}
	child.cursor = 0
	child.Update(tea.KeyMsg{Type: tea.KeyUp})
	if child.cursor != 0 {
		t.Fatal("upper bound changed")
	}
	child.cursor = len(child.menu) - 1
	child.Update(tea.KeyMsg{Type: tea.KeyDown})
	if child.cursor != len(child.menu)-1 {
		t.Fatal("lower bound changed")
	}
	empty := homeModel{}
	if action, handled = empty.Update(tea.KeyMsg{Type: tea.KeyEnter}); !handled || action.navigate {
		t.Fatal("empty menu navigated")
	}
	if _, handled = child.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); handled {
		t.Fatal("child consumed global quit")
	}
	if _, handled = child.Update(tea.WindowSizeMsg{Width: 80}); handled {
		t.Fatal("child consumed global window size")
	}
}

func TestHomeChildRoutePreservesPrefetchAndRoomEntry(t *testing.T) {
	m := model{active: screenHome, home: newHomeModel(), loadedScreens: map[screen]bool{screenDashboard: true}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if got.active != screenDashboard || got.loading || cmd != nil {
		t.Fatal("prefetched route changed")
	}
	m.home.cursor = 6
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(model)
	if got.active != screenRoomDay || got.room.roomDaysSelected == nil {
		t.Fatal("room flow entry changed")
	}
}
