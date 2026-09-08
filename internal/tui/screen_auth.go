package tui

import (
	"context"
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

type authScreenModel struct {
	authInputs     []textinput.Model
	authFocus      int
	authSubmitting bool
}

type authService interface {
	Users(context.Context) ([]app.UserRow, error)
	Authenticate(context.Context, string, string) error
}

func (m *authScreenModel) Update(msg tea.Msg, ctx context.Context, service authService) (childAction, bool) {
	switch msg := msg.(type) {
	case authCheckMsg:
		if msg.err != nil || len(msg.users) == 0 {
			m.authInputs = newAuthInputs()
			return childAction{setError: true, err: msg.err, cmd: textinput.Blink}, true
		}
		return childAction{setError: true}, true
	case authSubmitMsg:
		m.authSubmitting = false
		if msg.err == nil {
			m.authInputs = nil
			m.authFocus = 0
		}
		return childAction{setError: true, err: msg.err}, true
	case tea.KeyMsg:
		if len(m.authInputs) == 0 {
			m.authInputs = newAuthInputs()
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			return childAction{cmd: tea.Quit}, true
		case "enter":
			if m.authSubmitting {
				return childAction{}, true
			}
			if m.authFocus < len(m.authInputs)-1 {
				m.authFocus++
				m.updateAuthFocus()
				return childAction{}, true
			}
			studentID := strings.TrimSpace(m.authInputs[0].Value())
			password := strings.TrimSpace(m.authInputs[1].Value())
			if studentID == "" || password == "" {
				return childAction{setError: true, err: errors.New("학번과 비밀번호를 모두 입력하세요")}, true
			}
			m.authSubmitting = true
			return childAction{setError: true, cmd: submitAuth(ctx, service, studentID, password)}, true
		case "up", "shift+tab", "backtab":
			if !m.authSubmitting && m.authFocus > 0 {
				m.authFocus--
			}
			m.updateAuthFocus()
			return childAction{}, true
		case "down", "tab":
			if !m.authSubmitting && m.authFocus < len(m.authInputs)-1 {
				m.authFocus++
			}
			m.updateAuthFocus()
			return childAction{}, true
		}
		if m.authSubmitting {
			return childAction{}, true
		}
		var cmds []tea.Cmd
		for index := range m.authInputs {
			var cmd tea.Cmd
			m.authInputs[index], cmd = m.authInputs[index].Update(msg)
			cmds = append(cmds, cmd)
		}
		return childAction{cmd: tea.Batch(cmds...)}, true
	}
	return childAction{}, false
}

func (m *authScreenModel) updateAuthFocus() {
	for index := range m.authInputs {
		if index == m.authFocus {
			m.authInputs[index].Focus()
			continue
		}
		m.authInputs[index].Blur()
	}
}

func checkAuthUsers(ctx context.Context, service authService) tea.Cmd {
	return func() tea.Msg {
		users, err := service.Users(ctx)
		return authCheckMsg{users: users, err: err}
	}
}

func submitAuth(ctx context.Context, service authService, studentID string, password string) tea.Cmd {
	return func() tea.Msg {
		err := service.Authenticate(ctx, studentID, password)
		return authSubmitMsg{err: err}
	}
}

func (m authScreenModel) View(width int, loading bool, err error) string {
	var b strings.Builder
	b.WriteString(renderHeaderTitle(width, screenTitle(screenAuth)))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Account Setup"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("저장된 계정이 없습니다. KLAS 로그인 검증 후 계정을 저장합니다."))
	b.WriteString("\n\n")
	if loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 계정 상태를 확인하는 중입니다")
		b.WriteString("\n\n")
		b.WriteString(renderHelpText("esc 종료", width))
		return b.String()
	}
	if err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(err.Error())
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
