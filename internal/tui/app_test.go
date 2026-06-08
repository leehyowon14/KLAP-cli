package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHomeViewShowsMenu(t *testing.T) {
	m := model{
		ctx: context.Background(),
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
			{title: "Due", help: "다가오는 데드라인", screen: screenDue},
		},
	}

	view := m.View()
	if !strings.Contains(view, "KLAP TUI") || !strings.Contains(view, "> Dashboard") {
		t.Fatalf("View() = %q", view)
	}
}

func TestHomeNavigation(t *testing.T) {
	m := model{
		ctx: context.Background(),
		menu: []menuItem{
			{title: "Dashboard", screen: screenDashboard},
			{title: "Due", screen: screenDue},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(model)
	if got.cursor != 1 {
		t.Fatalf("cursor after down = %d", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyUp})
	got = updated.(model)
	if got.cursor != 0 {
		t.Fatalf("cursor after up = %d", got.cursor)
	}
}
