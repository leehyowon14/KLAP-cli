package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

// updateActiveChild adapts root-owned route and global state into each child's
// inputs. Children return navigation intent without owning the root model.
func (m *model) updateActiveChild(msg tea.Msg) (childAction, bool) {
	switch m.active {
	case screenHome:
		return m.home.Update(msg)
	case screenAuth:
		return m.auth.Update(msg, m.ctx, m.service)
	case screenDownloadSelect, screenDownloadConfirm, screenDownloadLanguage, screenDownloadProgress:
		return m.download.Update(msg, m.active, m.loading, m.ctx, m.service)
	case screenAttendSelect, screenAttendConfirm, screenAttendProgress:
		return m.attend.Update(msg, m.active, m.ctx, m.service, time.Now())
	case screenConfig, screenConfigChoice, screenConfigInput:
		return m.config.Update(msg, m.active, m.loading, m.ctx, m.service)
	case screenRoomDay, screenRoomPeriod, screenRoomResult:
		return m.room.Update(msg, m.active, m.loading, m.ctx, m.service)
	case screenSyncConflict:
		return m.sync.Update(msg, m.ctx, m.service, m.due.duePage == 3)
	case screenSyllabus:
		return m.syllabus.Update(msg, m.width, m.height, m.loading, m.ctx, m.service, m.dashboard.dashboardResult.Term.Value)
	case screenLectures:
		return m.lectures.Update(msg, m.width, m.loading)
	}
	if m.loading {
		return childAction{}, false
	}
	switch m.active {
	case screenAssignments, screenAssignmentDetail:
		return m.assignments.Update(msg, m.width, m.height, m.active == screenAssignmentDetail, m.ctx, m.service)
	case screenNotices, screenNoticeDetail:
		return m.notices.Update(msg, m.width, m.height, m.active == screenNoticeDetail, m.ctx, m.service)
	case screenDashboard:
		return m.dashboard.Update(msg, m.width, m.height, m.lastSyncAt)
	case screenAcademic:
		return m.academic.Update(msg)
	case screenDue:
		return m.due.Update(msg, m.width)
	}
	return childAction{}, false
}

type screen int

const (
	screenHome screen = iota
	screenAuth
	screenDashboard
	screenDue
	screenAssignments
	screenNotices
	screenLectures
	screenAssignmentDetail
	screenNoticeDetail
	screenSyllabus
	screenAcademic
	screenConfig
	screenRoomDay
	screenRoomPeriod
	screenRoomResult
	screenSyncConflict
	screenConfigChoice
	screenConfigInput
	screenDownloadSelect
	screenDownloadConfirm
	screenDownloadLanguage
	screenDownloadProgress
	screenAttendConfirm
	screenAttendProgress
	screenAttendSelect
)

func (m model) updateGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		if m.active == screenSyllabus {
			m.active = screenDashboard
			m.err = nil
			m.loading = false
			m.syllabus.resetCursor()
		} else if m.isDetailScreen() {
			if m.active == screenAssignmentDetail {
				m.assignments.resetDetailCursor()
			} else {
				m.notices.resetDetailCursor()
			}
			m.active = m.detailBack
			m.err = nil
			m.loading = false
		} else if m.active != screenHome {
			m.active = screenHome
			m.err = nil
			m.content = ""
			m.loading = false
		}
	case keyMatches(key, "r", "ㄱ"):
		if m.active != screenHome {
			m.loading = true
			m.err = nil
			m.content = ""
			m.syncStatus = ""
			m.markScreenLoading(m.active)
			return m, m.load(m.active, true)
		}
	case keyMatches(key, "s", "ㄴ"):
		if cmd := syncForScreen(m.ctx, m.service, m.active, m.due.duePage == 3, nil); cmd != nil {
			m.loading = false
			m.err = nil
			m.sync.Start(m.active)
			m.syncStatus = ""
			return m, cmd
		}
	case keyMatches(key, "k", "ㅏ"):
		if cmd := m.openCurrentKlasURL(); cmd != nil {
			m.err = nil
			return m, cmd
		}
	}
	return m, nil
}

func (m model) enterScreen(target screen) (tea.Model, tea.Cmd) {
	m.active = target
	m.err = nil
	m.content = ""
	if m.isScreenLoaded(target) {
		m.loading = false
		if err := m.screenError(target); err != nil {
			m.err = err
		}
		return m, nil
	}
	m.loading = true
	if m.isScreenLoading(target) {
		return m, nil
	}
	m.markScreenLoading(target)
	return m, m.load(target, false)
}

func (m model) isDetailScreen() bool {
	return m.active == screenAssignmentDetail || m.active == screenNoticeDetail
}

func (m *model) moveDetailCursor(delta int) {
	if m.active == screenAssignmentDetail {
		m.assignments.moveDetailCursor(delta, m.width, m.height)
	} else {
		m.notices.moveDetailCursor(delta, m.width, m.height)
	}
}

func (m model) detailLines(width int) []string {
	switch m.active {
	case screenAssignmentDetail:
		return assignmentDetailLines(m.assignments.assignmentDetail, width)
	case screenNoticeDetail:
		return noticeDetailLines(m.notices.noticeDetail, width)
	default:
		return nil
	}
}

func screenTitle(value screen) string {
	switch value {
	case screenAuth:
		return "Account"
	case screenDashboard:
		return "Dashboard"
	case screenDue:
		return "Due"
	case screenAssignments:
		return "Assignments"
	case screenNotices:
		return "Notices"
	case screenLectures:
		return "Lectures"
	case screenAssignmentDetail:
		return "Assignment"
	case screenNoticeDetail:
		return "Notice"
	case screenSyllabus:
		return "Syllabus"
	case screenAcademic:
		return "Academic"
	case screenConfig:
		return "Config"
	case screenConfigChoice:
		return "Config"
	case screenConfigInput:
		return "Config"
	case screenRoomDay, screenRoomPeriod, screenRoomResult:
		return "Rooms"
	case screenSyncConflict:
		return "Sync"
	case screenDownloadSelect:
		return "Download"
	case screenDownloadConfirm:
		return "Transcript"
	case screenDownloadLanguage:
		return "Transcript"
	case screenDownloadProgress:
		return "Download"
	case screenAttendConfirm, screenAttendProgress:
		return "Attend"
	default:
		return "Home"
	}
}

func screenSubtitle(value screen) string {
	switch value {
	case screenDashboard:
		return "과제, 온라인 강의, 공지, 출석, 수업평가 요약"
	case screenDue:
		return "다가오는 과제, 온라인 강의, 학사일정"
	case screenAssignments:
		return "현재 학기 과제 목록"
	case screenNotices:
		return "최근 강의 공지"
	case screenLectures:
		return "온라인 강의와 학습활동 상태"
	case screenAssignmentDetail:
		return "과제 상세"
	case screenNoticeDetail:
		return "공지 상세"
	case screenSyllabus:
		return "선택한 과목의 강의계획서"
	case screenAcademic:
		return "학사일정 달력"
	case screenConfig:
		return "현재 유저 설정"
	case screenConfigChoice:
		return "설정 항목 선택"
	case screenConfigInput:
		return "설정 값 입력"
	case screenRoomResult:
		return "선택한 요일과 교시에 비어 있는 강의실"
	case screenSyncConflict:
		return "KLAS 변경사항 반영 여부 선택"
	default:
		return ""
	}
}
