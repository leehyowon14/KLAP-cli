package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"time"
)

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func appendWrappedLines(lines []string, text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return lines
	}
	wrapWidth := maxInt(24, minInt(100, width-6))
	for _, paragraph := range strings.Split(text, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		rendered := lipgloss.NewStyle().Width(wrapWidth).Render(paragraph)
		lines = append(lines, strings.Split(rendered, "\n")...)
	}
	return lines
}

func splitRenderedLines(value string) []string {
	return strings.Split(strings.TrimRight(value, "\n"), "\n")
}

func enabledLabel(value bool) string {
	if value {
		return "켜짐"
	}
	return "꺼짐"
}

func formatMinutes(minutes int) string {
	if minutes%1440 == 0 {
		days := minutes / 1440
		if days == 1 {
			return "1일"
		}
		return fmt.Sprintf("%d일", days)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%d시간", minutes/60)
	}
	return fmt.Sprintf("%d분", minutes)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}

func renderSection(title string, lines []string) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(title))
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func formatSectionErrors(errors []app.DashboardSectionError) string {
	lines := make([]string, 0, len(errors))
	for _, sectionError := range errors {
		lines = append(lines, errorStyle.Render(sectionError.Section)+" "+sectionError.Err.Error())
	}
	return renderSection("확인 실패", lines)
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
