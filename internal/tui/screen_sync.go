package tui

import (
	"context"
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

func syncForScreen(ctx context.Context, service syncScreenService, source screen, allowDue bool, decisions map[string]app.SyncDecision) tea.Cmd {
	switch source {
	case screenDashboard:
		return syncDashboard(ctx, service, copySyncDecisions(decisions))
	case screenDue:
		if allowDue {
			return syncAcademic(ctx, service, copySyncDecisions(decisions))
		}
		return nil
	case screenAssignments:
		return syncAssignments(ctx, service, copySyncDecisions(decisions))
	case screenLectures:
		return syncLectures(ctx, service, copySyncDecisions(decisions))
	case screenAcademic:
		return syncAcademic(ctx, service, copySyncDecisions(decisions))
	default:
		return nil
	}
}

func (m syncScreenModel) currentSyncConflict() (app.SyncConflict, bool) {
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

func syncDashboard(ctx context.Context, service syncScreenService, decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result := service.SyncDashboard(ctx, app.DashboardSyncOptions{Decisions: decisions})
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

func syncAssignments(ctx context.Context, service syncScreenService, decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result, err := service.SyncAssignmentReminders(ctx, app.AssignmentSyncOptions{Decisions: decisions})
		if conflicts := syncConflictsFromErrors(err); len(conflicts) > 0 {
			return syncMsg{conflicts: conflicts}
		}
		return syncMsg{status: "과제 " + formatReminderSyncStatus("", result), err: err}
	}
}

func syncLectures(ctx context.Context, service syncScreenService, decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result, err := service.SyncLectureReminders(ctx, app.LectureSyncOptions{Decisions: decisions})
		if conflicts := syncConflictsFromErrors(err); len(conflicts) > 0 {
			return syncMsg{conflicts: conflicts}
		}
		return syncMsg{status: "강의 " + formatReminderSyncStatus("", result), err: err}
	}
}

func syncAcademic(ctx context.Context, service syncScreenService, decisions map[string]app.SyncDecision) tea.Cmd {
	return func() tea.Msg {
		result, err := service.SyncAcademicCalendar(ctx, app.AcademicSyncOptions{Decisions: decisions})
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

func (m syncScreenModel) View(width int) string {
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

func (m syncScreenModel) StatusView(statusText string) string {
	switch m.syncPhase {
	case "syncing":
		return warnBadgeStyle.Render("SYNCING") + " " + "캘린더와 미리알림을 동기화하는 중입니다\n"
	case "error":
		status := strings.TrimSpace(statusText)
		if status == "" {
			status = "동기화 중 오류가 발생했습니다"
		}
		return errorStyle.Render("ERROR") + " " + status + "\n"
	default:
		status := strings.TrimSpace(statusText)
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

type syncScreenModel struct {
	syncConflictSource  screen
	syncConflicts       []app.SyncConflict
	syncConflictCursor  int
	syncConflictActions map[string]app.SyncDecision
	syncPhase           string
}

type syncScreenService interface {
	SyncDashboard(context.Context, app.DashboardSyncOptions) app.DashboardSyncResult
	SyncAssignmentReminders(context.Context, app.AssignmentSyncOptions) (app.ReminderSyncResult, error)
	SyncLectureReminders(context.Context, app.LectureSyncOptions) (app.ReminderSyncResult, error)
	SyncAcademicCalendar(context.Context, app.AcademicSyncOptions) (app.CalendarSyncResult, error)
}

func (m *syncScreenModel) Start(source screen) {
	m.syncConflictSource = source
	m.syncConflictActions = map[string]app.SyncDecision{}
	m.syncPhase = "syncing"
}

func (m *syncScreenModel) ClearPhase() { m.syncPhase = "" }

func (m *syncScreenModel) Loaded(msg syncMsg) {
	if len(msg.conflicts) > 0 {
		m.syncPhase = ""
		m.syncConflicts = msg.conflicts
		m.syncConflictCursor = 0
		if m.syncConflictActions == nil {
			m.syncConflictActions = map[string]app.SyncDecision{}
		}
		for _, conflict := range msg.conflicts {
			if m.syncConflictActions[conflict.Key] == "" {
				m.syncConflictActions[conflict.Key] = app.SyncDecisionKeep
			}
		}
		return
	}
	m.syncPhase = "done"
	if msg.err != nil {
		m.syncPhase = "error"
	}
}

func (m *syncScreenModel) Update(msg tea.Msg, ctx context.Context, service syncScreenService, allowDue bool) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	k := key.String()
	switch {
	case k == "ctrl+c" || keyMatches(k, "q", "ㅂ"):
		return childAction{cmd: tea.Quit}, true
	case k == "esc" || keyMatches(k, "b", "ㅠ"):
		target := m.syncConflictSource
		if target == screenSyncConflict || target == 0 {
			target = screenHome
		}
		m.syncConflicts = nil
		m.syncConflictCursor = 0
		m.syncConflictActions = nil
		return childAction{navigate: true, routeOnly: true, target: target}, true
	case k == "left" || k == "right":
		conflict, ok := m.currentSyncConflict()
		if !ok {
			return childAction{}, true
		}
		if m.syncConflictActions == nil {
			m.syncConflictActions = map[string]app.SyncDecision{}
		}
		if m.syncConflictActions[conflict.Key] == app.SyncDecisionApply {
			m.syncConflictActions[conflict.Key] = app.SyncDecisionKeep
		} else {
			m.syncConflictActions[conflict.Key] = app.SyncDecisionApply
		}
	case k == "enter":
		if len(m.syncConflicts) == 0 {
			return childAction{navigate: true, routeOnly: true, target: screenHome}, true
		}
		if m.syncConflictCursor < len(m.syncConflicts)-1 {
			m.syncConflictCursor++
			return childAction{}, true
		}
		source := m.syncConflictSource
		if source == screenSyncConflict || source == 0 {
			source = screenDashboard
		}
		decisions := copySyncDecisions(m.syncConflictActions)
		m.syncConflicts = nil
		m.syncConflictCursor = 0
		m.syncPhase = "syncing"
		return childAction{navigate: true, routeOnly: true, target: source, setStatus: true, setError: true, cmd: syncForScreen(ctx, service, source, allowDue, decisions)}, true
	}
	return childAction{}, true
}

func conflictLabel(conflict app.SyncConflict) string {
	switch conflict.Scope {
	case "assignment":
		return "과제 미리알림"
	case "lecture":
		return "강의 미리알림"
	case "academic":
		return "학사일정 캘린더"
	case "timetable":
		return "시간표 캘린더"
	default:
		return conflict.Scope
	}
}
