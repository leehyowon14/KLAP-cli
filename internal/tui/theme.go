package tui

import (
	"github.com/charmbracelet/lipgloss"
)

const appHorizontalPadding = 4

var (
	appStyle = lipgloss.NewStyle().
			Padding(0, 2)
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))
	headerMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781"))
	panelStyle = lipgloss.NewStyle().
			Padding(0, 0)
	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8C959F")).
			Padding(0, 0)
	menuSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#56D4DD"))
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781"))
	successTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9ACD32"))
	warnTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4A72C"))
	selectedDayStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#0969DA"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CF222E")).
			Bold(true)
	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781")).
			Italic(true)
	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#30363D")).
			Padding(0, 1)
	successBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#238636")).
				Padding(0, 1)
	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#24292F")).
			Background(lipgloss.Color("#D4A72C")).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781"))
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))
	taglineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ACD32"))
)
