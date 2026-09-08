package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

const menuNumberWidth = 3

const klapLogo = ` _  __ _        _    ____
| |/ /| |      / \  |  _ \
| ' / | |     / _ \ | |_) |
| . \ | |___ / ___ \|  __/
|_|\_\|_____/_/   \_\_|`

type menuItem struct {
	title  string
	help   string
	screen screen
}

func (m homeModel) View(width int) string {
	logo := logoStyle.Render(klapLogo)
	meta := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render("Kwangwoon Learning Automation Project"),
		headerStyle.Render("https://github.com/leehyowon14/KLAP-cli"),
		taglineStyle.Render("Beyond KLAS, in your terminal."),
	)
	var top string
	if width >= 82 {
		top = lipgloss.JoinHorizontal(lipgloss.Top, logo, "   ", meta)
	} else {
		top = lipgloss.JoinVertical(lipgloss.Left, logo, meta)
	}

	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n\n")
	b.WriteString(m.renderHomeMenu(width))
	b.WriteString("\n\n")
	b.WriteString(renderHelpText("↑↓ 이동  |  enter 열기  |  r 새로고침  |  q 종료", width))
	return b.String()
}

func (m homeModel) renderHomeMenu(width int) string {
	var b strings.Builder
	for index, item := range m.menu {
		selected := index == m.cursor
		prefix := fmt.Sprintf("%d.", index+1)
		marker := "  "
		if selected {
			marker = "› "
		}
		number := lipgloss.NewStyle().Width(menuNumberWidth).Render(prefix)
		title := lipgloss.NewStyle().Width(16).Render(item.title)
		help := item.help
		line := marker + number + " " + title + " " + help
		if selected {
			line = menuSelectedStyle.Render(line)
		} else {
			line = menuItemStyle.Render(line)
		}
		b.WriteString(line)
		if index < len(m.menu)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

type homeModel struct {
	menu   []menuItem
	cursor int
}

// Update returns navigation intent; the root retains route and prefetch ownership.
func (m *homeModel) Update(msg tea.Msg) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	switch {
	case key.String() == "up" || keyMatches(key.String(), "k", "ㅏ"):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
		if m.cursor < len(m.menu)-1 {
			m.cursor++
		}
	case key.String() == "enter":
		if m.cursor >= 0 && m.cursor < len(m.menu) {
			return childAction{navigate: true, target: m.menu[m.cursor].screen}, true
		}
	default:
		return childAction{}, false
	}
	return childAction{}, true
}

func newHomeModel() homeModel {
	return homeModel{menu: []menuItem{
		{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
		{title: "Due", help: "다가오는 일정", screen: screenDue},
		{title: "Assignments", help: "과제 목록", screen: screenAssignments},
		{title: "Notices", help: "공지 목록", screen: screenNotices},
		{title: "Lectures", help: "강의 상태", screen: screenLectures},
		{title: "Academic", help: "학사일정", screen: screenAcademic},
		{title: "Rooms", help: "빈 강의실 조회", screen: screenRoomDay},
		{title: "Config", help: "설정", screen: screenConfig},
	}}
}
