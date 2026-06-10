package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type confirmModel struct {
	title    string
	message  string
	value    bool
	canceled bool
	width    int
}

func RunConfirm(ctx context.Context, title string, message string, defaultValue bool) (bool, error) {
	_ = ctx
	model := confirmModel{
		title:   title,
		message: message,
		value:   defaultValue,
	}
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return false, err
	}
	result, ok := finalModel.(confirmModel)
	if !ok || result.canceled {
		return false, context.Canceled
	}
	return result.value, nil
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		key := msg.String()
		switch {
		case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
			m.canceled = true
			return m, tea.Quit
		case keyMatches(key, "y", "ㅛ"):
			m.value = true
			return m, tea.Quit
		case keyMatches(key, "n", "ㅜ"):
			m.value = false
			return m, tea.Quit
		case key == "left" || key == "right" || key == "tab":
			m.value = !m.value
		case key == "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	contentWidth := tuiContentWidth(width)
	yes := "Yes"
	no := "No"
	if m.value {
		yes = menuSelectedStyle.Render("› Yes")
		no = mutedStyle.Render("  No")
	} else {
		yes = mutedStyle.Render("  Yes")
		no = menuSelectedStyle.Render("› No")
	}

	var b strings.Builder
	b.WriteString(renderHeaderTitle(contentWidth, "Confirm"))
	b.WriteString("\n")
	b.WriteString(renderRule(contentWidth))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render(m.title))
	b.WriteString("\n")
	b.WriteString(m.message)
	b.WriteString("\n\n")
	b.WriteString(yes)
	b.WriteString("    ")
	b.WriteString(no)
	b.WriteString("\n\n")
	b.WriteString(renderHelpText("←/→ 선택  |  y/n  |  enter 확인  |  q 취소", contentWidth))
	return appStyle.Render(b.String())
}
