package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type childAction struct {
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
			m.syllabusResult = app.SyllabusResult{}
			m.syllabusCourseIndex = action.courseIndex
			m.syllabusCursor = 0
			return m, m.loadSyllabus(action.courseIndex)
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
