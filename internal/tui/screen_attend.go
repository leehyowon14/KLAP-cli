package tui

import (
	"context"
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"time"
)

type attendScreenModel struct {
	batch          *attendBatch
	attendRow      app.LectureRow
	attendProgress *lectureAttendModel
}

func (m *attendScreenModel) Start(row app.LectureRow, selected bool, now time.Time) error {
	if !selected {
		return errors.New("수강할 강의를 선택할 수 없습니다")
	}
	if err := validateLectureAttend(row, now); err != nil {
		return err
	}
	m.attendRow = row
	m.batch = nil
	return nil
}
func (m *attendScreenModel) Update(msg tea.Msg, route screen, ctx context.Context, service lectureAttender, now time.Time) (childAction, bool) {
	if m.batch != nil {
		return m.updateBatch(msg, route, ctx, service, now), true
	}
	if route == screenAttendProgress {
		return m.updateProgress(msg), true
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	k := key.String()
	switch {
	case k == "ctrl+c" || keyMatches(k, "q", "ㅂ"):
		return childAction{cmd: tea.Quit}, true
	case k == "esc" || keyMatches(k, "b", "ㅠ", "n", "ㅜ"):
		m.attendRow = app.LectureRow{}
		return childAction{navigate: true, routeOnly: true, target: screenLectures}, true
	case k == "enter" || keyMatches(k, "y", "ㅛ"):
		return m.startProgress(ctx, service, now), true
	}
	return childAction{}, true
}
func (m *attendScreenModel) startProgress(ctx context.Context, service lectureAttender, now time.Time) childAction {
	if err := validateLectureAttend(m.attendRow, now); err != nil {
		return childAction{navigate: true, routeOnly: true, target: screenLectures, setError: true, err: err}
	}
	runCtx, cancel := context.WithCancel(ctx)
	progress := &lectureAttendModel{ctx: runCtx, cancel: cancel, service: service, row: m.attendRow, updates: make(chan tea.Msg, 16), progress: initialAttendProgress(m.attendRow)}
	m.attendProgress = progress
	return childAction{navigate: true, routeOnly: true, target: screenAttendProgress, setError: true, cmd: progress.Init()}
}
func (m *attendScreenModel) updateProgress(msg tea.Msg) childAction {
	if m.attendProgress == nil {
		return childAction{navigate: true, routeOnly: true, target: screenLectures}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		k := key.String()
		if k == "ctrl+c" || keyMatches(k, "q", "ㅂ") {
			m.attendProgress.cancel()
			return childAction{cmd: tea.Quit}
		}
		if keyMatches(k, "h", "ㅗ") {
			m.attendProgress.cancel()
			m.attendProgress = nil
			m.attendRow = app.LectureRow{}
			return childAction{navigate: true, routeOnly: true, target: screenHome}
		}
		if m.attendProgress.done && (k == "esc" || keyMatches(k, "b", "ㅠ")) {
			m.attendProgress = nil
			m.attendRow = app.LectureRow{}
			return childAction{navigate: true, reload: true, refresh: true, target: screenLectures}
		}
		if k == "esc" && !m.attendProgress.done {
			m.attendProgress.canceling = true
			m.attendProgress.cancel()
			return childAction{}
		}
	}
	updated, cmd := m.attendProgress.Update(msg)
	if progress, ok := updated.(lectureAttendModel); ok {
		m.attendProgress = &progress
	}
	return childAction{cmd: cmd}
}
func (m attendScreenModel) View(width int, route screen) string {
	if m.batch != nil {
		return m.batchView(width, route)
	}
	if route == screenAttendConfirm {
		return m.renderAttendConfirmView(width)
	}
	if m.attendProgress == nil {
		return appStyle.Render(errorStyle.Render("수강 상태가 없습니다"))
	}
	return m.attendProgress.View()
}

func (m attendScreenModel) renderAttendConfirmView(width int) string {
	var b strings.Builder
	b.WriteString(renderHeaderTitle(width, "Attend"))
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
