package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func fixedColumn(value string, width int) string {
	value = truncateText(value, width)
	return lipgloss.NewStyle().Width(width).Render(value)
}

func tuiContentWidth(width int) int {
	if width <= 0 {
		width = 96
	}
	return maxInt(1, width-appHorizontalPadding)
}

func renderRule(width int) string {
	return mutedStyle.Render(strings.Repeat("─", maxInt(1, minInt(width, 120))))
}

func (m model) renderFooterHelp(width int) string {
	return renderHelpText(m.footerHelp(), width)
}

func renderHelpText(help string, width int) string {
	return footerStyle.Render(wrapHelp(help, maxInt(1, width)))
}

func wrapHelp(help string, width int) string {
	parts := strings.Split(help, "  ")
	lines := make([]string, 0, 2)
	current := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		candidate := part
		if current != "" {
			candidate = current + "  " + part
		}
		if current != "" && lipgloss.Width(candidate) > width {
			lines = append(lines, current)
			current = part
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}

func (m model) renderHeader(width int) string {
	return renderHeaderTitle(width, screenTitle(m.active))
}

func renderHeaderTitle(width int, subtitle string) string {
	if width <= 0 {
		width = 96
	}
	title := headerStyle.Render("KLAP")
	titleWidth := lipgloss.Width("KLAP")
	gap := 2
	available := width - titleWidth - gap
	if available < 0 {
		available = 0
	}
	if available > 0 {
		subtitle = truncateText(subtitle, available)
	} else {
		subtitle = ""
	}
	if strings.TrimSpace(subtitle) == "" {
		return title
	}
	spacer := strings.Repeat(" ", maxInt(1, width-titleWidth-lipgloss.Width(subtitle)))
	line := title + spacer + headerMetaStyle.Render(subtitle)
	if lipgloss.Width(line) > width {
		subtitle = truncateText(subtitle, maxInt(0, width-titleWidth-1))
		spacer = strings.Repeat(" ", maxInt(1, width-titleWidth-lipgloss.Width(subtitle)))
		line = title + spacer + headerMetaStyle.Render(subtitle)
	}
	return line
}

func renderWindowedLines(lines []string, offset int, visibleRows int) string {
	if len(lines) == 0 {
		return ""
	}
	if visibleRows <= 0 {
		visibleRows = 1
	}
	if visibleRows > len(lines) {
		visibleRows = len(lines)
	}
	maxOffset := maxInt(0, len(lines)-visibleRows)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	end := offset + visibleRows
	var b strings.Builder
	for index := offset; index < end; index++ {
		b.WriteString(lines[index])
		b.WriteString("\n")
	}
	if offset > 0 || end < len(lines) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", offset+1, end, len(lines))))
		b.WriteString("\n")
	}
	return b.String()
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= limit {
		return value
	}
	if limit <= 1 {
		return "…"
	}
	var b strings.Builder
	for _, r := range value {
		next := b.String() + string(r)
		if lipgloss.Width(next+"…") > limit {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "…"
}

func padRight(value string, width int) string {
	padding := width - lipgloss.Width(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
