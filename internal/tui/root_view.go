package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m model) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	contentWidth := tuiContentWidth(width)
	switch m.active {
	case screenAuth:
		return appStyle.Render(m.auth.View(contentWidth, m.loading, m.err))
	case screenDownloadSelect:
		return appStyle.Render(m.download.View(contentWidth, m.height, m.active, m.loading, m.err))
	case screenDownloadConfirm:
		return appStyle.Render(m.download.View(contentWidth, m.height, m.active, m.loading, m.err))
	case screenDownloadLanguage:
		return appStyle.Render(m.download.View(contentWidth, m.height, m.active, m.loading, m.err))
	case screenDownloadProgress:
		return m.download.View(contentWidth, m.height, m.active, m.loading, m.err)
	case screenAttendConfirm:
		return appStyle.Render(m.attend.View(contentWidth, m.active))
	case screenAttendProgress:
		return m.attend.View(contentWidth, m.active)
	case screenConfigChoice:
		return appStyle.Render(m.config.View(contentWidth, m.active, m.err))
	case screenConfigInput:
		return appStyle.Render(m.config.View(contentWidth, m.active, m.err))
	case screenRoomDay:
		return appStyle.Render(m.room.View(contentWidth, m.height, m.active, m.err))
	case screenRoomPeriod:
		return appStyle.Render(m.room.View(contentWidth, m.height, m.active, m.err))
	}
	if m.active == screenHome {
		return appStyle.Render(m.home.View(contentWidth))
	}

	header := m.renderHeader(contentWidth)
	rule := renderRule(contentWidth)
	panel := panelStyle.Width(contentWidth).Render(m.renderPanel(contentWidth))
	footer := m.renderFooterHelp(contentWidth)
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, rule, "", panel, "", footer))
}

func (m model) footerHelp() string {
	if m.sync.syncPhase != "" {
		return "동기화 진행 중  q 종료"
	}
	if m.active == screenDashboard {
		if m.dashboard.dashboardPage > 0 {
			return "←/→ 페이지  ↑↓ 스크롤  p 강의계획서  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←/→ 페이지  ↑↓ 스크롤  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenSyllabus {
		return "↑↓ 스크롤  b/esc Dashboard  r 새로고침  q 종료"
	}
	if m.active == screenDue {
		if m.due.duePage == 3 {
			return "←/→ 페이지  ↑↓ 스크롤  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←/→ 페이지  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenAcademic {
		return "←/→ 월 이동  ↑↓ 일정  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenRoomResult {
		return "←/→ 건물  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenSyncConflict {
		return "←/→ 선택  enter 확정/다음  b/esc 취소  q 종료"
	}
	if m.isDetailScreen() {
		return "↑↓ 스크롤  k KLAS  b/esc 목록  q 종료"
	}
	if m.active == screenLectures {
		return "←/→ 과목  ↑↓ 스크롤  a 수강  k KLAS  d 다운로드  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenConfig {
		return "←/→ 페이지  ↑↓ 선택  enter 선택  [] 변경  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.isCoursePagedScreen() {
		if m.active == screenAssignments {
			return "←/→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
		}
		if m.active == screenNotices {
			return "←/→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←/→ 과목  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenRoomResult {
		return "b/esc 뒤로  r 새로고침  q 종료"
	}
	return "b/esc 뒤로  r 새로고침  q 종료"
}

func (m model) renderPanel(width int) string {
	var b strings.Builder
	if subtitle := strings.TrimSpace(screenSubtitle(m.active)); subtitle != "" {
		b.WriteString(mutedStyle.Render(subtitle))
		b.WriteString("\n\n")
	}
	if m.sync.syncPhase != "" {
		b.WriteString(m.sync.StatusView(m.syncStatus))
		return b.String()
	}
	if m.loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 데이터를 불러오는 중입니다\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n")
		return b.String()
	}
	if m.active == screenDashboard {
		b.WriteString(m.renderDashboardPagedPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if m.active == screenSyllabus {
		b.WriteString(m.renderSyllabusPanel(width))
		return b.String()
	}
	if m.active == screenDue {
		b.WriteString(m.renderDuePagedPanel(width))
		return b.String()
	}
	if m.isDetailScreen() {
		b.WriteString(m.renderDetailPanel(width))
		return b.String()
	}
	if m.active == screenAcademic {
		b.WriteString(m.renderAcademicCalendarPanel(width))
		return b.String()
	}
	if m.active == screenRoomResult {
		b.WriteString(m.room.View(width, m.height, m.active, m.err))
		return b.String()
	}
	if m.active == screenSyncConflict {
		b.WriteString(m.sync.View(width))
		return b.String()
	}
	if m.active == screenConfig {
		b.WriteString(m.config.View(width, m.active, m.err))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if m.isCoursePagedScreen() {
		b.WriteString(m.renderCoursePagedPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if strings.TrimSpace(m.content) == "" {
		b.WriteString(emptyStyle.Render("표시할 내용이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(m.content)
	if !strings.HasSuffix(m.content, "\n") {
		b.WriteString("\n")
	}
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) renderDetailPanel(width int) string {
	if m.active == screenAssignmentDetail {
		return m.assignments.View(width, m.height, true, m.syncStatus)
	}
	return m.notices.View(width, m.height, true, m.syncStatus)
}
