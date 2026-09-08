package tui

import tea "github.com/charmbracelet/bubbletea"

type childAction struct {
	navigate bool
	target   screen
	cmd      tea.Cmd
}

func (m model) applyChildAction(action childAction) (tea.Model, tea.Cmd) {
	if action.navigate {
		if action.target == screenRoomDay {
			return m.startRoomFlow()
		}
		return m.enterScreen(action.target)
	}
	return m, action.cmd
}
