package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	case screenAttendConfirm, screenAttendProgress:
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
