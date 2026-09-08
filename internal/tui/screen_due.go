package tui

import (
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

var duePageLabels = []string{"Summary", "과제", "온라인 강의", "학사일정"}

func (m *model) moveDuePage(delta int) {
	m.duePage += delta
	if m.duePage < 0 {
		m.duePage = len(duePageLabels) - 1
	}
	if m.duePage >= len(duePageLabels) {
		m.duePage = 0
	}
	m.dueCursor = 0
}

func (m *model) moveDueCursor(delta int) {
	lines := duePageLines(m.dueResult, m.duePage, m.width)
	if len(lines) == 0 {
		m.dueCursor = 0
		return
	}
	m.dueCursor += delta
	if m.dueCursor < 0 {
		m.dueCursor = len(lines) - 1
	}
	if m.dueCursor >= len(lines) {
		m.dueCursor = 0
	}
}

func (m model) renderDuePagedPanel(width int) string {
	lines := duePageLines(m.dueResult, m.duePage, width)
	page := m.duePage + 1
	if page < 1 {
		page = 1
	}
	if page > len(duePageLabels) {
		page = len(duePageLabels)
	}

	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%s ~ %s", m.dueResult.From.Format("2006-01-02"), m.dueResult.Until.Format("2006-01-02"))))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%d/%d  %s", page, len(duePageLabels), duePageLabels[page-1])))
	b.WriteString("\n\n")
	if len(lines) == 0 {
		b.WriteString(emptyStyle.Render("표시할 일정이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}

	visibleRows := maxInt(5, m.height-12)
	if m.height <= 0 {
		visibleRows = 16
	}
	if visibleRows > len(lines) {
		visibleRows = len(lines)
	}
	cursor := m.dueCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(lines) {
		cursor = len(lines) - 1
	}
	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(lines) {
		start = maxInt(0, len(lines)-visibleRows)
	}
	end := start + visibleRows
	for index := start; index < end; index++ {
		marker := "  "
		if index == cursor {
			marker = "› "
		}
		line := marker + lines[index]
		if index == cursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < len(lines) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(lines))))
		b.WriteString("\n")
	}
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func duePageLines(result app.DueResult, page int, width int) []string {
	if page == 0 {
		return dueSummaryLines(result)
	}
	if page < 0 || page >= len(duePageLabels) {
		return nil
	}
	kind := duePageLabels[page]
	lines := make([]string, 0)
	for _, item := range result.Items {
		if item.Kind != kind {
			continue
		}
		course := item.CourseName
		if course == "" {
			course = "-"
		}
		titleWidth := maxInt(12, minInt(64, width-38))
		lines = append(lines, fmt.Sprintf("%s  %s  %s  %s",
			mutedStyle.Render(item.DueAt.Format("2006-01-02 15:04")),
			badgeStyle.Render(item.Kind),
			truncateText(course, 18),
			truncateText(item.Title, titleWidth),
		))
	}
	if page >= 0 && page < len(duePageLabels) {
		for _, sectionError := range result.Errors {
			if sectionError.Section == duePageLabels[page] {
				lines = append(lines, errorStyle.Render("ERROR")+" "+sectionError.Err.Error())
			}
		}
	}
	return lines
}

func dueSummaryLines(result app.DueResult) []string {
	counts := map[string]int{}
	for _, item := range result.Items {
		counts[item.Kind]++
	}
	lines := splitRenderedLines(renderSection("OVERVIEW", []string{
		dashboardMetric("Range", fmt.Sprintf("%s ~ %s", result.From.Format("2006-01-02"), result.Until.Format("2006-01-02"))),
		dashboardMetric("Total", fmt.Sprintf("%d items", len(result.Items))),
		dashboardMetric("Assignments", fmt.Sprintf("%d", counts["과제"])),
		dashboardMetric("Lectures", fmt.Sprintf("%d", counts["온라인 강의"])),
		dashboardMetric("Academic", fmt.Sprintf("%d", counts["학사일정"])),
	}))
	lines = append(lines, "")
	lines = append(lines, splitRenderedLines(renderSection("FOCUS", dueFocusLines(result.Items, 6)))...)
	if len(result.Errors) > 0 {
		errorLines := make([]string, 0, len(result.Errors))
		for _, sectionError := range result.Errors {
			errorLines = append(errorLines, errorStyle.Render(sectionError.Section)+" "+sectionError.Err.Error())
		}
		lines = append(lines, "")
		lines = append(lines, splitRenderedLines(renderSection("ERRORS", errorLines))...)
	}
	return lines
}

func dueFocusLines(items []app.DueItem, limit int) []string {
	if len(items) == 0 {
		return []string{emptyStyle.Render("다가오는 일정이 없습니다")}
	}
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	lines := make([]string, 0, limit)
	for _, item := range items[:limit] {
		course := strings.TrimSpace(item.CourseName)
		if course != "" {
			course += " · "
		}
		lines = append(lines, fmt.Sprintf("%s  %s  %s%s",
			mutedStyle.Render(item.DueAt.Format("01-02 15:04")),
			dueKindBadge(item.Kind),
			course,
			item.Title,
		))
	}
	return lines
}

func dueKindBadge(kind string) string {
	switch kind {
	case "과제":
		return warnBadgeStyle.Render(kind)
	case "온라인 강의":
		return badgeStyle.Render("강의")
	case "학사일정":
		return successBadgeStyle.Render("학사")
	default:
		return badgeStyle.Render(kind)
	}
}

func formatDue(result app.DueResult) string {
	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%s ~ %s", result.From.Format("2006-01-02"), result.Until.Format("2006-01-02"))))
	b.WriteString("\n\n")
	if len(result.Items) == 0 {
		b.WriteString(emptyStyle.Render("예정된 데드라인이 없습니다"))
		b.WriteString("\n")
	}
	for _, item := range result.Items {
		course := item.CourseName
		if course == "" {
			course = "-"
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			mutedStyle.Render(item.DueAt.Format("2006-01-02 15:04")),
			badgeStyle.Render(item.Kind),
			course,
			item.Title,
		))
	}
	if len(result.Errors) > 0 {
		b.WriteString("\n확인 실패\n")
		for _, sectionError := range result.Errors {
			b.WriteString(fmt.Sprintf("  %s: %v\n", sectionError.Section, sectionError.Err))
		}
	}
	return b.String()
}
