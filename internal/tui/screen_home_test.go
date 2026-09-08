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
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
			{title: "Due", help: "다가오는 데드라인", screen: screenDue},
		},
	}

	view := m.View()
	if !strings.Contains(view, "https://github.com/leehyowon14/KLAP-cli") || !strings.Contains(view, "Beyond KLAS, in your terminal.") || !strings.Contains(view, "Kwangwoon Learning Automation Project") || !strings.Contains(view, "› 1.") || !strings.Contains(view, "Dashboard") || !strings.Contains(view, "_  __") {
		t.Fatalf("View() = %q", view)
	}
}

func TestHomeViewLinesFitWidth(t *testing.T) {
	m := model{
		width: 96,
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
		},
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

func TestHomeNavigationAcceptsKoreanKeyboardKeys(t *testing.T) {
	m := model{
		ctx: context.Background(),
		menu: []menuItem{
			{title: "Dashboard", screen: screenDashboard},
			{title: "Due", screen: screenDue},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅓ")})
	got := updated.(model)
	if got.cursor != 1 {
		t.Fatalf("cursor after korean j key = %d", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅏ")})
	got = updated.(model)
	if got.cursor != 0 {
		t.Fatalf("cursor after korean k key = %d", got.cursor)
	}
}
