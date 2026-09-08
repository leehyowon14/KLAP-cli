package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

type assignmentScreenModel struct {
	assignmentRows   []app.AssignmentRow
	assignmentDetail app.AssignmentDetailResult
	pager            coursePager
	detailCursor     int
}

type assignmentScreenService interface {
	AssignmentList(context.Context, app.AssignmentListOptions) ([]app.AssignmentRow, error)
	AssignmentDetail(context.Context, string, app.UserOption) (app.AssignmentDetailResult, error)
}

func (m *assignmentScreenModel) Loaded(rows []app.AssignmentRow) { m.assignmentRows = rows }
func (m *assignmentScreenModel) DetailLoaded(result app.AssignmentDetailResult) {
	m.assignmentDetail = result
	m.detailCursor = 0
}
func (m *assignmentScreenModel) resetDetailCursor() { m.detailCursor = 0 }

func (m assignmentScreenModel) groups(width int) []contentCourseGroup {
	return assignmentContentGroups(m.assignmentRows, maxInt(24, minInt(92, width-12)))
}
func (m assignmentScreenModel) selectedRow(width int) (app.AssignmentRow, bool) {
	group := m.pager.currentGroup(m.groups(width))
	return selectedRowByCourse(m.assignmentRows, group.name, m.pager.contentCursor, func(row app.AssignmentRow) string { return row.CourseName })
}
func (m model) selectedAssignmentRow() (app.AssignmentRow, bool) {
	return m.assignments.selectedRow(m.width)
}

func (m *assignmentScreenModel) Update(msg tea.Msg, width, height int, detail bool, ctx context.Context, service assignmentScreenService) (childAction, bool) {
	if !detail && m.pager.Update(msg, m.groups(width)) {
		return childAction{}, true
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	if detail {
		switch {
		case key.String() == "up":
			m.moveDetailCursor(-1, width, height)
		case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
			m.moveDetailCursor(1, width, height)
		default:
			return childAction{}, false
		}
		return childAction{}, true
	}
	if key.String() != "enter" {
		return childAction{}, false
	}
	row, ok := m.selectedRow(width)
	if !ok {
		return childAction{setStatus: true, status: "선택된 과제가 없습니다"}, true
	}
	return childAction{navigate: true, target: screenAssignmentDetail, cmd: loadAssignmentDetail(ctx, service, row.ID)}, true
}

func loadAssignmentDetail(ctx context.Context, service assignmentScreenService, id string) tea.Cmd {
	return func() tea.Msg {
		result, err := service.AssignmentDetail(ctx, id, app.UserOption{})
		return detailMsg{screen: screenAssignmentDetail, assignment: result, err: err}
	}
}
func (m *assignmentScreenModel) moveDetailCursor(delta, width, height int) {
	lines := assignmentDetailLines(m.assignmentDetail, width)
	maxCursor := maxInt(0, len(lines)-visibleBodyRows(height, 0))
	m.detailCursor = clampInt(m.detailCursor+delta, 0, maxCursor)
}
func (m assignmentScreenModel) View(width, height int, detail bool, status string) string {
	if !detail {
		return m.pager.View(m.groups(width), height, screenAssignments)
	}
	lines := assignmentDetailLines(m.assignmentDetail, width)
	if len(lines) == 0 {
		return emptyStyle.Render("상세 정보가 없습니다") + "\n"
	}
	rendered := renderWindowedLines(lines, m.detailCursor, visibleBodyRows(height, 0))
	if status != "" {
		rendered += "\n" + footerStyle.Render(status) + "\n"
	}
	return rendered
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

func (m *assignmentScreenModel) Reset() { *m = assignmentScreenModel{} }

func loadAssignments(ctx context.Context, service assignmentScreenService, refresh, prefetch bool) tea.Cmd {
	return func() tea.Msg {
		rows, err := service.AssignmentList(ctx, app.AssignmentListOptions{Refresh: refresh})
		return loadMsg{screen: screenAssignments, prefetch: prefetch, assignments: rows, err: err}
	}
}
