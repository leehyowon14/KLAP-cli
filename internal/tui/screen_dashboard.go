package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"time"
)

func (m dashboardScreenModel) currentDashboardCourse() (app.DashboardCourse, bool) {
	index := m.dashboardPage - 1
	if index < 0 || index >= len(m.dashboardResult.Courses) {
		return app.DashboardCourse{}, false
	}
	return m.dashboardResult.Courses[index], true
}

func (m *dashboardScreenModel) moveDashboardPage(delta int) {
	total := dashboardPageCount(m.dashboardResult)
	if total <= 0 {
		m.dashboardPage = 0
		return
	}
	m.dashboardPage += delta
	if m.dashboardPage < 0 {
		m.dashboardPage = total - 1
	}
	if m.dashboardPage >= total {
		m.dashboardPage = 0
	}
	m.dashboardCursor = 0
}

func (m *dashboardScreenModel) moveDashboardCursor(delta, width, height int, lastSyncAt time.Time) {
	lines := m.dashboardLines(width, lastSyncAt)
	if len(lines) == 0 {
		m.dashboardCursor = 0
		return
	}
	visibleRows := visibleBodyRows(height, 0)
	maxCursor := maxInt(0, len(lines)-visibleRows)
	m.dashboardCursor += delta
	if m.dashboardCursor < 0 {
		m.dashboardCursor = 0
	}
	if m.dashboardCursor > maxCursor {
		m.dashboardCursor = maxCursor
	}
}

func (m dashboardScreenModel) View(width, height int, lastSyncAt time.Time) string {
	lines := m.dashboardLines(width, lastSyncAt)
	if len(lines) == 0 {
		return emptyStyle.Render("대시보드 데이터가 없습니다") + "\n"
	}
	return renderWindowedLines(lines, m.dashboardCursor, visibleBodyRows(height, 0))
}

func (m dashboardScreenModel) dashboardLines(width int, lastSyncAt time.Time) []string {
	pages := dashboardPageCount(m.dashboardResult)
	if pages <= 1 || m.dashboardPage == 0 {
		lines := splitRenderedLines(formatDashboard(m.dashboardResult))
		if !lastSyncAt.IsZero() {
			lines = append(lines, "")
			lines = append(lines, splitRenderedLines(renderSection("SYNC", []string{dashboardMetric("Last Syncing", lastSyncAt.Format("2006-01-02 15:04:05"))}))...)
		}
		return lines
	}
	courses := m.dashboardResult.Courses
	index := m.dashboardPage - 1
	if index < 0 {
		index = 0
	}
	if index >= len(courses) {
		index = len(courses) - 1
	}
	return splitRenderedLines(formatDashboardCourse(m.dashboardResult, courses[index]))
}

func formatDashboard(result app.DashboardResult) string {
	var b strings.Builder
	b.WriteString(formatDashboardOverview(result))
	if result.Cached {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("캐시 사용 " + result.CacheCreatedAt.Format("2006-01-02 15:04")))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderSection("FOCUS", formatDashboardFocus(result)))
	b.WriteString("\n")
	b.WriteString(renderSection("LATEST", formatNoticeSummary(result.Notices)))
	if len(result.SectionErrors) > 0 {
		b.WriteString("\n")
		b.WriteString(formatSectionErrors(result.SectionErrors))
	}
	return b.String()
}

func dashboardPageCount(result app.DashboardResult) int {
	return 1 + len(result.Courses)
}

func formatDashboardOverview(result app.DashboardResult) string {
	lines := []string{
		fmt.Sprintf("%s  %s", badgeStyle.Render(result.Term.Value), result.Term.Label),
		dashboardMetric("Due", fmt.Sprintf("%d assignments · %d lectures", len(result.Assignments), len(result.Lectures))),
		dashboardMetric("Activity", fmt.Sprintf("%d notices", len(result.Notices))),
	}
	if result.Attendance.TotalCourses > 0 {
		total := result.Attendance.Completed + result.Attendance.Absent + result.Attendance.Late + result.Attendance.LeaveEarly + result.Attendance.Excused + result.Attendance.Unknown
		lines = append(lines, dashboardMetric("Attendance", fmt.Sprintf("출석 %d/%d · 결석 %d · 지각 %d", result.Attendance.Completed, total, result.Attendance.Absent, result.Attendance.Late)))
	}
	if result.Evaluation.Enabled {
		lines = append(lines, dashboardMetric("Evaluation", fmt.Sprintf("완료 %d · 미완료 %d", result.Evaluation.Done, result.Evaluation.Pending)))
	}
	return renderSection("OVERVIEW", lines)
}

func formatDashboardCourse(result app.DashboardResult, course app.DashboardCourse) string {
	var b strings.Builder
	b.WriteString(formatDashboardCourseOverview(result, course))
	b.WriteString("\n")
	b.WriteString(renderSection("FOCUS", formatDashboardCourseFocus(course)))
	b.WriteString("\n")
	b.WriteString(renderSection("LATEST", formatNoticeSummary(course.Notices)))
	status := formatDashboardCourseStatus(course)
	if len(status) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSection("STATUS", status))
	}
	return b.String()
}

func formatDashboardCourseOverview(result app.DashboardResult, course app.DashboardCourse) string {
	lines := []string{
		fmt.Sprintf("%s  %s", badgeStyle.Render(result.Term.Value), result.Term.Label),
		dashboardMetric("Course", fmt.Sprintf("%d. %s", course.Index, course.Name)),
		dashboardMetric("Due", fmt.Sprintf("%d assignments · %d lectures", len(course.Assignments), len(course.Lectures))),
		dashboardMetric("Activity", fmt.Sprintf("%d notices", len(course.Notices))),
	}
	if course.Attendance != nil {
		total := course.Attendance.Completed + course.Attendance.Absent + course.Attendance.Late + course.Attendance.LeaveEarly + course.Attendance.Excused + course.Attendance.Unknown
		lines = append(lines, dashboardMetric("Attendance", fmt.Sprintf("출석 %d/%d · 결석 %d · 지각 %d", course.Attendance.Completed, total, course.Attendance.Absent, course.Attendance.Late)))
	}
	if result.Evaluation.Enabled {
		evaluationStatus := "완료"
		if course.Evaluation != nil {
			evaluationStatus = "미완료"
		}
		lines = append(lines, dashboardMetric("Evaluation", evaluationStatus))
	}
	return renderSection("OVERVIEW", lines)
}

func dashboardMetric(label string, value string) string {
	return mutedStyle.Render(lipgloss.NewStyle().Width(13).Render(label)) + " " + value
}

func formatDashboardFocus(result app.DashboardResult) []string {
	lines := make([]string, 0, len(result.Assignments)+len(result.Lectures))
	for _, row := range result.Assignments {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			warnBadgeStyle.Render("과제"),
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			row.CourseName+" · "+row.Assignment.Title,
		))
	}
	for _, row := range result.Lectures {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			badgeStyle.Render("강의"),
			mutedStyle.Render(formatTime(row.Lecture.EndAt)),
			row.CourseName+" · "+row.Lecture.Title,
		))
	}
	if len(lines) == 0 {
		return []string{emptyStyle.Render("처리할 항목이 없습니다")}
	}
	return lines
}

func formatDashboardCourseFocus(course app.DashboardCourse) []string {
	lines := make([]string, 0, len(course.Assignments)+len(course.Lectures))
	for _, row := range course.Assignments {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			warnBadgeStyle.Render("과제"),
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			row.Assignment.Title,
		))
	}
	for _, row := range course.Lectures {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			badgeStyle.Render("강의"),
			mutedStyle.Render(formatTime(row.Lecture.EndAt)),
			row.Lecture.Title,
		))
	}
	if len(lines) == 0 {
		return []string{emptyStyle.Render("처리할 항목이 없습니다")}
	}
	return lines
}

func formatDashboardCourseStatus(course app.DashboardCourse) []string {
	lines := make([]string, 0, 3)
	if course.Attendance != nil {
		if course.Attendance.Err != nil {
			lines = append(lines, warnTextStyle.Render("출석 상세를 불러오지 못했습니다"))
		} else {
			lines = append(lines, fmt.Sprintf("출석  %s · %s · %s",
				successTextStyle.Render(fmt.Sprintf("O %d", course.Attendance.Completed)),
				warnTextStyle.Render(fmt.Sprintf("X %d", course.Attendance.Absent)),
				warnTextStyle.Render(fmt.Sprintf("L %d", course.Attendance.Late)),
			))
		}
	}
	if course.Evaluation != nil {
		lines = append(lines, warnTextStyle.Render("수업평가 미완료"))
	}
	if len(lines) == 0 {
		return nil
	}
	return lines
}

func formatAssignmentSummary(rows []app.AssignmentRow) []string {
	if len(rows) == 0 {
		return []string{emptyStyle.Render("예정된 과제가 없습니다")}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", mutedStyle.Render(formatTime(row.Assignment.DueAt)), row.CourseName, row.Assignment.Title))
	}
	return lines
}

func formatLectureSummary(rows []app.LectureRow) []string {
	if len(rows) == 0 {
		return []string{emptyStyle.Render("수강할 온라인 강의가 없습니다")}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", mutedStyle.Render(formatTime(row.Lecture.EndAt)), row.CourseName, row.Lecture.Title))
	}
	return lines
}

func formatNoticeSummary(rows []app.NoticeRow) []string {
	if len(rows) == 0 {
		return []string{emptyStyle.Render("공지 없음")}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			mutedStyle.Render(formatTime(row.Notice.Registered)),
			truncateText(row.CourseName, 22),
			truncateText(row.Notice.Title, 30),
		))
	}
	return lines
}

type dashboardScreenModel struct {
	dashboardResult app.DashboardResult
	dashboardPage   int
	dashboardCursor int
}
type dashboardScreenService interface {
	Dashboard(context.Context, app.DashboardOptions) (app.DashboardResult, error)
}

func (m *dashboardScreenModel) Loaded(result app.DashboardResult, active bool) {
	m.dashboardResult = result
	if active {
		m.dashboardPage = 0
		m.dashboardCursor = 0
	}
}
func (m *dashboardScreenModel) Reset()     { *m = dashboardScreenModel{} }
func (m *dashboardScreenModel) resetPage() { m.dashboardPage = 0 }
func (m *dashboardScreenModel) Update(msg tea.Msg, width, height int, lastSyncAt time.Time) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	switch {
	case key.String() == "up" || keyMatches(key.String(), "k", "ㅏ"):
		m.moveDashboardCursor(-1, width, height, lastSyncAt)
	case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
		m.moveDashboardCursor(1, width, height, lastSyncAt)
	case key.String() == "left":
		m.moveDashboardPage(-1)
	case key.String() == "right":
		m.moveDashboardPage(1)
	case keyMatches(key.String(), "p", "ㅔ"):
		course, ok := m.currentDashboardCourse()
		if !ok {
			return childAction{}, false
		}
		index := course.Index
		if index <= 0 {
			index = m.dashboardPage
		}
		return childAction{navigate: true, target: screenSyllabus, courseIndex: index}, true
	default:
		return childAction{}, false
	}
	return childAction{}, true
}
func loadDashboard(ctx context.Context, service dashboardScreenService, refresh, prefetch bool) tea.Cmd {
	return func() tea.Msg {
		result, err := service.Dashboard(ctx, app.DashboardOptions{Refresh: refresh})
		return loadMsg{screen: screenDashboard, prefetch: prefetch, content: formatDashboard(result), dashboard: result, err: err}
	}
}
func (m model) renderDashboardPagedPanel(width int) string {
	return m.dashboard.View(width, m.height, m.lastSyncAt)
}
func (m *model) moveDashboardPage(delta int) { m.dashboard.moveDashboardPage(delta) }
