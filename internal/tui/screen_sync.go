package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"time"
)

type syncMsg struct {
	status    string
	err       error
	conflicts []app.SyncConflict
}

type syncDoneTimeoutMsg struct{}

func (m model) syncCurrentScreen() tea.Cmd {
	return m.syncScreenWithDecisions(m.active, nil)
}

func (m model) syncScreenWithDecisions(source screen, decisions map[string]app.SyncDecision) tea.Cmd {
	switch source {
	case screenDashboard:
		return m.syncDashboard(decisions)
	case screenDue:
		if m.due.duePage == 3 {
			return m.syncAcademic(decisions)
		}
		return nil
	case screenAssignments:
		return m.syncAssignments(decisions)
	case screenLectures:
		return m.syncLectures(decisions)
	case screenAcademic:
		return m.syncAcademic(decisions)
	default:
		return nil
	}
}

func (m model) updateSyncConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = m.syncConflictSource
		if m.active == screenSyncConflict || m.active == 0 {
			m.active = screenHome
		}
		m.syncConflicts = nil
		m.syncConflictCursor = 0
		m.syncConflictActions = nil
		return m, nil
	case key == "left" || key == "right":
		conflict, ok := m.currentSyncConflict()
		if !ok {
			return m, nil
		}
		if m.syncConflictActions == nil {
			m.syncConflictActions = map[string]app.SyncDecision{}
		}
		if m.syncConflictActions[conflict.Key] == app.SyncDecisionApply {
			m.syncConflictActions[conflict.Key] = app.SyncDecisionKeep
		} else {
			m.syncConflictActions[conflict.Key] = app.SyncDecisionApply
		}
	case key == "enter":
		if len(m.syncConflicts) == 0 {
			m.active = screenHome
			return m, nil
		}
		if m.syncConflictCursor < len(m.syncConflicts)-1 {
			m.syncConflictCursor++
			return m, nil
		}
		source := m.syncConflictSource
		if source == screenSyncConflict || source == 0 {
			source = screenDashboard
		}
		decisions := copySyncDecisions(m.syncConflictActions)
		m.active = source
		m.syncConflicts = nil
		m.syncConflictCursor = 0
		m.syncPhase = "syncing"
		m.syncStatus = ""
		m.err = nil
		return m, m.syncScreenWithDecisions(source, decisions)
	}
	return m, nil
}

func (m model) currentSyncConflict() (app.SyncConflict, bool) {
	if len(m.syncConflicts) == 0 {
		return app.SyncConflict{}, false
	}
	if m.syncConflictCursor < 0 {
		return m.syncConflicts[0], true
	}
	if m.syncConflictCursor >= len(m.syncConflicts) {
		return m.syncConflicts[len(m.syncConflicts)-1], true
	}
	return m.syncConflicts[m.syncConflictCursor], true
}

func copySyncDecisions(source map[string]app.SyncDecision) map[string]app.SyncDecision {
	if len(source) == 0 {
		return nil
	}
	copied := make(map[string]app.SyncDecision, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func (m model) syncDashboard(decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result := m.service.SyncDashboard(m.ctx, app.DashboardSyncOptions{Decisions: decisions})
		assignments, assignmentErr := result.Assignments, result.AssignmentError
		lectures, lectureErr := result.Lectures, result.LectureError
		academic, academicErr := result.Academic, result.AcademicError
		timetable, timetableErr := result.Timetable, result.TimetableError
		conflicts := syncConflictsFromErrors(assignmentErr, lectureErr, academicErr, timetableErr)
		if len(conflicts) > 0 {
			return syncMsg{conflicts: conflicts}
		}
		parts := []string{
			formatReminderSyncStatus("과제", assignments),
			formatReminderSyncStatus("강의", lectures),
			formatCalendarSyncStatus("학사일정", academic),
			formatCalendarSyncStatus("시간표", timetable),
		}
		if err := firstErr(assignmentErr, lectureErr, academicErr, timetableErr); err != nil {
			return syncMsg{status: strings.Join(parts, " / "), err: err}
		}
		return syncMsg{status: "동기화 완료: " + strings.Join(parts, " / ")}
	}
}

func (m model) syncAssignments(decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.SyncAssignmentReminders(m.ctx, app.AssignmentSyncOptions{Decisions: decisions})
		if conflicts := syncConflictsFromErrors(err); len(conflicts) > 0 {
			return syncMsg{conflicts: conflicts}
		}
		return syncMsg{status: "과제 " + formatReminderSyncStatus("", result), err: err}
	}
}

func (m model) syncLectures(decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.SyncLectureReminders(m.ctx, app.LectureSyncOptions{Decisions: decisions})
		if conflicts := syncConflictsFromErrors(err); len(conflicts) > 0 {
			return syncMsg{conflicts: conflicts}
		}
		return syncMsg{status: "강의 " + formatReminderSyncStatus("", result), err: err}
	}
}

func (m model) syncAcademic(decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.SyncAcademicCalendar(m.ctx, app.AcademicSyncOptions{Decisions: decisions})
		if conflicts := syncConflictsFromErrors(err); len(conflicts) > 0 {
			return syncMsg{conflicts: conflicts}
		}
		return syncMsg{status: "학사일정 " + formatCalendarSyncStatus("", result), err: err}
	}
}

func syncConflictsFromErrors(errs ...error) []app.SyncConflict {
	var conflicts []app.SyncConflict
	for _, err := range errs {
		var conflictErr app.SyncConflictError
		if errors.As(err, &conflictErr) {
			conflicts = append(conflicts, conflictErr.Conflicts...)
		}
	}
	return conflicts
}

func syncDoneTimeout() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return syncDoneTimeoutMsg{}
	})
}

func formatReminderSyncStatus(label string, result app.ReminderSyncResult) string {
	prefix := ""
	if strings.TrimSpace(label) != "" {
		prefix = label + " "
	}
	if result.EligibleCount == 0 {
		return prefix + "대상 없음"
	}
	return fmt.Sprintf("%s생성 %d, 갱신 %d, 완료 %d, 제외 %d",
		prefix,
		result.Result.Created,
		result.Result.Updated,
		result.Result.Completed,
		result.Result.Skipped,
	)
}

func formatCalendarSyncStatus(label string, result app.CalendarSyncResult) string {
	prefix := ""
	if strings.TrimSpace(label) != "" {
		prefix = label + " "
	}
	if result.EligibleCount == 0 {
		return prefix + "대상 없음"
	}
	return fmt.Sprintf("%s생성 %d, 갱신 %d, 제외 %d",
		prefix,
		result.Result.Created,
		result.Result.Updated,
		result.Result.Skipped,
	)
}

func (m model) renderSyncConflictPanel(width int) string {
	var b strings.Builder
	conflict, ok := m.currentSyncConflict()
	if !ok {
		b.WriteString(emptyStyle.Render("확인할 동기화 변경사항이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	total := len(m.syncConflicts)
	current := m.syncConflictCursor + 1
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	decision := m.syncConflictActions[conflict.Key]
	if decision == "" {
		decision = app.SyncDecisionKeep
	}
	b.WriteString(sectionStyle.Render("KLAS 원본 변경 확인"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%d/%d  %s", current, total, conflictLabel(conflict))))
	b.WriteString("\n\n")
	b.WriteString("항목: ")
	b.WriteString(truncateText(conflict.Title, maxInt(12, width-8)))
	b.WriteString("\n")
	if strings.TrimSpace(conflict.Summary) != "" {
		b.WriteString("KLAS: ")
		b.WriteString(truncateText(conflict.Summary, maxInt(12, width-8)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	var keep string
	var apply string
	if decision == app.SyncDecisionApply {
		keep = mutedStyle.Render("  내 수정 유지")
		apply = menuSelectedStyle.Render("› KLAS 갱신 반영")
	} else {
		keep = menuSelectedStyle.Render("› 내 수정 유지")
		apply = mutedStyle.Render("  KLAS 갱신 반영")
	}
	b.WriteString(keep)
	b.WriteString("    ")
	b.WriteString(apply)
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("이 선택은 해당 KLAS 변경값에 대해 한 번만 저장됩니다."))
	b.WriteString("\n")
	return b.String()
}

func (m model) renderSyncPanel() string {
	switch m.syncPhase {
	case "syncing":
		return warnBadgeStyle.Render("SYNCING") + " " + "캘린더와 미리알림을 동기화하는 중입니다\n"
	case "error":
		status := strings.TrimSpace(m.syncStatus)
		if status == "" {
			status = "동기화 중 오류가 발생했습니다"
		}
		return errorStyle.Render("ERROR") + " " + status + "\n"
	default:
		status := strings.TrimSpace(m.syncStatus)
		if status == "" {
			status = "동기화 완료"
		}
		return successBadgeStyle.Render("DONE") + " " + formatSyncPanelStatus(status) + "\n"
	}
}

func formatSyncPanelStatus(status string) string {
	status = strings.TrimSpace(status)
	status = strings.TrimPrefix(status, "동기화 완료:")
	status = strings.TrimSpace(status)
	if status == "" || !strings.Contains(status, " / ") {
		return status
	}
	parts := strings.Split(status, " / ")
	lines := []string{"동기화 완료", ""}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			lines = append(lines, part)
		}
	}
	return strings.Join(lines, "\n")
}
