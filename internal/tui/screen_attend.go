package tui

import (
	"context"
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"time"
)

func (m model) startAttendConfirm() (tea.Model, tea.Cmd) {
	row, ok := m.selectedLectureRow()
	if !ok {
		m.err = errors.New("수강할 강의를 선택할 수 없습니다")
		return m, nil
	}
	if err := validateLectureAttend(row, time.Now()); err != nil {
		m.err = err
		return m, nil
	}
	m.active = screenAttendConfirm
	m.attendRow = row
	m.err = nil
	return m, nil
}

func (m model) updateAttendConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ", "n", "ㅜ"):
		m.active = screenLectures
		m.attendRow = app.LectureRow{}
		return m, nil
	case key == "enter" || keyMatches(key, "y", "ㅛ"):
		return m.startAttendProgress()
	}
	return m, nil
}

func (m model) startAttendProgress() (tea.Model, tea.Cmd) {
	if err := validateLectureAttend(m.attendRow, time.Now()); err != nil {
		m.err = err
		m.active = screenLectures
		return m, nil
	}
	runCtx, cancel := context.WithCancel(m.ctx)
	progress := lectureAttendModel{
		ctx:      runCtx,
		cancel:   cancel,
		service:  m.service,
		row:      m.attendRow,
		updates:  make(chan tea.Msg, 16),
		progress: initialAttendProgress(m.attendRow),
	}
	m.active = screenAttendProgress
	m.attendProgress = &progress
	m.err = nil
	return m, progress.Init()
}

func (m model) updateAttendProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.attendProgress == nil {
		m.active = screenLectures
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		value := key.String()
		if value == "ctrl+c" || keyMatches(value, "q", "ㅂ") {
			m.attendProgress.cancel()
			return m, tea.Quit
		}
		if keyMatches(value, "h", "ㅗ") {
			m.attendProgress.cancel()
			m.active = screenHome
			m.attendProgress = nil
			m.attendRow = app.LectureRow{}
			return m, nil
		}
		if m.attendProgress.done && (value == "esc" || keyMatches(value, "b", "ㅠ")) {
			m.active = screenLectures
			m.loading = true
			m.attendProgress = nil
			m.attendRow = app.LectureRow{}
			return m, m.load(screenLectures, true)
		}
		if value == "esc" && !m.attendProgress.done {
			m.attendProgress.canceling = true
			m.attendProgress.cancel()
			return m, nil
		}
	}
	updated, cmd := m.attendProgress.Update(msg)
	progress, ok := updated.(lectureAttendModel)
	if ok {
		m.attendProgress = &progress
	}
	return m, cmd
}

func (m model) renderAttendConfirmView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Attend Lecture"))
	b.WriteString("\n")
	b.WriteString(truncateText(m.attendRow.CourseName, maxInt(16, width-4)))
	b.WriteString("\n")
	b.WriteString(truncateText(firstLectureLabel(m.attendRow), maxInt(16, width-4)))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(lectureProgress(m.attendRow.Lecture)))
	b.WriteString("\n\n")
	b.WriteString(warnTextStyle.Render("수강 진도가 완료될 때까지 KLAS에 주기적으로 반영됩니다."))
	b.WriteString("\n")
	b.WriteString("이 강의를 수강할까요?")
	b.WriteString("\n\n")
	b.WriteString(renderHelpText("y/enter 시작  |  n/b 취소  |  q 종료", width))
	return b.String()
}
