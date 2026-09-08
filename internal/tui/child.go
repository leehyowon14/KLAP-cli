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
		if action.target == screenRoomDay {
			return m.startRoomFlow()
		}
		return m.enterScreen(action.target)
	}
	return m, action.cmd
}
