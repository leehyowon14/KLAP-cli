package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type childAction struct {
	setLoading  bool
	loading     bool
	courseIndex int
	setStatus   bool
	status      string
	setError    bool
	err         error
	reload      bool
	refresh     bool
	routeOnly   bool
	navigate    bool
	target      screen
	cmd         tea.Cmd
}

func (m model) applyChildAction(action childAction) (tea.Model, tea.Cmd) {
	if action.setLoading {
		m.loading = action.loading
	}
	if action.setStatus {
		m.syncStatus = action.status
	}
	if action.setError {
		m.err = action.err
	}
	if action.navigate {
		if action.reload {
			m.active = action.target
			m.loading = true
			m.markScreenLoading(action.target)
			return m, m.load(action.target, action.refresh)
		}
		if action.routeOnly {
			m.active = action.target
			return m, action.cmd
		}

		if action.target == screenHome {
			m.active = screenHome
			m.loading = false
			m.err = nil
			m.content = ""
			return m, action.cmd
		}

		if (m.active == screenConfig || m.active == screenConfigChoice || m.active == screenConfigInput) && (action.target == screenConfig || action.target == screenConfigChoice || action.target == screenConfigInput) {
			m.active = action.target
			m.loading = false
			return m, action.cmd
		}

		if action.target == screenDownloadSelect {
			m.active = screenDownloadSelect
			m.loading = true
			m.err = nil
			m.content = ""
			return m, loadDownloadRows(m.ctx, m.service)
		}
		if action.target == screenAttendConfirm {
			return m.startAttendConfirm()
		}

		if action.target == screenSyllabus {
			m.active = screenSyllabus
			m.loading = true
			m.err = nil
			m.syllabus.Start(action.courseIndex)
			return m, m.syllabus.Load(m.ctx, m.service, m.dashboard.dashboardResult.Term.Value)
		}
		if action.target == screenAssignmentDetail || action.target == screenNoticeDetail {
			m.detailBack = m.active
			m.active = action.target
			m.loading = true
			m.err = nil
			m.syncStatus = ""
			return m, action.cmd
		}
		if action.target == screenRoomDay || action.target == screenRoomPeriod || action.target == screenRoomResult {
			if action.target == screenRoomDay && m.active != screenRoomPeriod {
				m.room.Start()
			}
			m.active = action.target
			m.err = nil
			m.content = ""
			m.loading = action.target == screenRoomResult && action.cmd != nil
			return m, action.cmd
		}
		return m.enterScreen(action.target)
	}
	return m, action.cmd
}
