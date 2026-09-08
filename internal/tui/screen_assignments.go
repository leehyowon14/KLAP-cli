package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

func (m model) openAssignmentDetail() (tea.Model, tea.Cmd) {
	row, ok := m.selectedAssignmentRow()
	if !ok {
		m.syncStatus = "선택된 과제가 없습니다"
		return m, nil
	}
	m.active = screenAssignmentDetail
	m.detailBack = screenAssignments
	m.loading = true
	m.err = nil
	m.syncStatus = ""
	return m, m.loadAssignmentDetail(row.ID)
}

func (m model) loadAssignmentDetail(id string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.AssignmentDetail(m.ctx, id, app.UserOption{})
		return detailMsg{screen: screenAssignmentDetail, assignment: result, err: err}
	}
}

func (m model) selectedAssignmentRow() (app.AssignmentRow, bool) {
	group := m.currentContentGroup(m.width)
	return selectedRowByCourse(m.assignmentRows, group.name, m.pager.contentCursor, func(row app.AssignmentRow) string {
		return row.CourseName
	})
}

func assignmentContentGroups(rows []app.AssignmentRow, width int) []contentCourseGroup {
	groups := make([]contentCourseGroup, 0)
	indexByName := make(map[string]int)
	for _, row := range rows {
		groupIndex := contentGroupIndex(&groups, indexByName, row.CourseName)
		status := warnBadgeStyle.Render("미제출")
		if row.Assignment.Submitted {
			status = successBadgeStyle.Render("제출")
		}
		title := truncateText(row.Assignment.Title, maxInt(12, width-24))
		groups[groupIndex].lines = append(groups[groupIndex].lines, fmt.Sprintf("%s  %s  %s",
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			status,
			title,
		))
	}
	return groups
}

func assignmentDetailLines(result app.AssignmentDetailResult, width int) []string {
	if strings.TrimSpace(result.ID) == "" {
		return nil
	}
	detail := result.Detail
	status := "미제출"
	if detail.Submitted {
		status = "제출"
	}
	lines := []string{
		sectionStyle.Render(detail.Title),
		mutedStyle.Render(result.CourseName),
		"",
		"마감  " + formatTime(detail.DueAt),
		"상태  " + status,
	}
	if detail.ReportType != "" {
		lines = append(lines, "제출 방식  "+detail.ReportType)
	}
	if detail.SubmitFileType != "" {
		lines = append(lines, "파일 형식  "+detail.SubmitFileType)
	}
	if detail.FileLimitMB != "" {
		lines = append(lines, "파일 제한  "+detail.FileLimitMB+"MB")
	}
	lines = append(lines, "", mutedStyle.Render("KLAS  "+result.DetailURL))
	if strings.TrimSpace(detail.ContentText) != "" {
		lines = append(lines, "", sectionStyle.Render("본문"))
		lines = appendWrappedLines(lines, detail.ContentText, width)
	}
	if strings.TrimSpace(detail.SubmittedTitle+detail.SubmittedText) != "" {
		lines = append(lines, "", sectionStyle.Render("내 제출"))
		if detail.SubmittedTitle != "" {
			lines = append(lines, "제목  "+detail.SubmittedTitle)
		}
		lines = appendWrappedLines(lines, detail.SubmittedText, width)
	}
	if detail.FinalScore != "" && detail.FinalScore != "<nil>" {
		lines = append(lines, "", "점수  "+detail.FinalScore)
	}
	if strings.TrimSpace(detail.TutorText) != "" {
		lines = append(lines, "", sectionStyle.Render("피드백"))
		lines = appendWrappedLines(lines, detail.TutorText, width)
	}
	return lines
}

func formatAssignments(rows []app.AssignmentRow) string {
	if len(rows) == 0 {
		return emptyStyle.Render("과제가 없습니다") + "\n"
	}
	var b strings.Builder
	for _, row := range rows {
		status := warnBadgeStyle.Render("미제출")
		if row.Assignment.Submitted {
			status = successBadgeStyle.Render("제출")
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			status,
			row.CourseName,
			row.Assignment.Title,
		))
	}
	return b.String()
}
