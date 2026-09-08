package cli

import (
	"errors"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AuthCredentials struct {
	StudentID string
	Password  string
}

type authModel struct {
	inputs []textinput.Model
	focus  int
	done   bool
	cancel bool
	err    error
	width  int
	height int
}

func RunAuthForm() (AuthCredentials, error) {
	studentID := textinput.New()
	studentID.Placeholder = "학번"
	studentID.Prompt = "학번 "
	studentID.Focus()
	studentID.CharLimit = 32
	studentID.Width = 32

	password := textinput.New()
	password.Placeholder = "비밀번호"
	password.Prompt = "비밀번호 "
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'
	password.CharLimit = 128
	password.Width = 32

	model := authModel{inputs: []textinput.Model{studentID, password}}
	result, err := tea.NewProgram(model).Run()
	if err != nil {
		return AuthCredentials{}, err
	}

	finalModel, ok := result.(authModel)
	if !ok {
		return AuthCredentials{}, errors.New("auth form returned unexpected model")
	}
	if finalModel.cancel {
		return AuthCredentials{}, errors.New("인증 입력이 취소되었습니다")
	}
	if finalModel.err != nil {
		return AuthCredentials{}, finalModel.err
	}

	return AuthCredentials{
		StudentID: finalModel.inputs[0].Value(),
		Password:  finalModel.inputs[1].Value(),
	}, nil
}

func (m authModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m authModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit
		case "enter":
			if m.focus == len(m.inputs)-1 {
				m.done = true
				return m, tea.Quit
			}
			m.focus++
			return m.updateFocus(), nil
		case "shift+tab", "up":
			if m.focus > 0 {
				m.focus--
			}
			return m.updateFocus(), nil
		case "tab", "down":
			if m.focus < len(m.inputs)-1 {
				m.focus++
			}
			return m.updateFocus(), nil
		}
	}

	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m authModel) View() string {
	if m.done {
		return ""
	}

	return "\nKLAS 계정 인증\n\n" +
		m.inputs[0].View() + "\n" +
		m.inputs[1].View() + "\n\n" +
		"Enter: 다음/저장  Tab: 이동  Esc: 취소\n"
}

func (m authModel) updateFocus() authModel {
	for i := range m.inputs {
		if i == m.focus {
			m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()
	}
	return m
}
