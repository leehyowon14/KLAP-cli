package tui

import (
	"fmt"
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

func (m model) renderHomeView(width int) string {
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

func (m model) renderHomeMenu(width int) string {
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
