package tui

import (
	"errors"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

type authCheckMsg struct {
	users []app.UserRow
	err   error
}

type authSubmitMsg struct {
	err error
}

func newAuthInputs() []textinput.Model {
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

	return []textinput.Model{studentID, password}
}

func (m model) updateAuth(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.authInputs) == 0 {
		m.authInputs = newAuthInputs()
	}
	key := msg.String()
	switch key {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "enter":
		if m.authSubmitting {
			return m, nil
		}
		if m.authFocus < len(m.authInputs)-1 {
			m.authFocus++
			return m.updateAuthFocus(), nil
		}
		studentID := strings.TrimSpace(m.authInputs[0].Value())
		password := strings.TrimSpace(m.authInputs[1].Value())
		if studentID == "" || password == "" {
			m.err = errors.New("학번과 비밀번호를 모두 입력하세요")
			return m, nil
		}
		m.err = nil
		m.authSubmitting = true
		return m, m.submitAuth(studentID, password)
	case "up", "shift+tab", "backtab":
		if !m.authSubmitting && m.authFocus > 0 {
			m.authFocus--
		}
		return m.updateAuthFocus(), nil
	case "down", "tab":
		if !m.authSubmitting && m.authFocus < len(m.authInputs)-1 {
			m.authFocus++
		}
		return m.updateAuthFocus(), nil
	}
	if m.authSubmitting {
		return m, nil
	}
	var cmds []tea.Cmd
	for index := range m.authInputs {
		var cmd tea.Cmd
		m.authInputs[index], cmd = m.authInputs[index].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) updateAuthFocus() model {
	for index := range m.authInputs {
		if index == m.authFocus {
			m.authInputs[index].Focus()
			continue
		}
		m.authInputs[index].Blur()
	}
	return m
}

func (m model) checkAuthUsers() tea.Cmd {
	return func() tea.Msg {
		users, err := m.service.Users(m.ctx)
		return authCheckMsg{users: users, err: err}
	}
}

func (m model) submitAuth(studentID string, password string) tea.Cmd {
	return func() tea.Msg {
		err := m.service.Authenticate(m.ctx, studentID, password)
		return authSubmitMsg{err: err}
	}
}

func (m model) renderAuthView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Account Setup"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("저장된 계정이 없습니다. KLAS 로그인 검증 후 계정을 저장합니다."))
	b.WriteString("\n\n")
	if m.loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 계정 상태를 확인하는 중입니다")
		b.WriteString("\n\n")
		b.WriteString(renderHelpText("esc 종료", width))
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	if len(m.authInputs) == 0 {
		m.authInputs = newAuthInputs()
	}
	for index, input := range m.authInputs {
		b.WriteString(input.View())
		if index < len(m.authInputs)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")
	if m.authSubmitting {
		b.WriteString(warnBadgeStyle.Render("AUTHENTICATING"))
		b.WriteString(" KLAS에 로그인 요청을 보내는 중입니다")
		b.WriteString("\n\n")
	}
	b.WriteString(renderHelpText("enter 다음/저장  tab 이동  esc 종료", width))
	return b.String()
}
