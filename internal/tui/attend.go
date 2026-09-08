package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	bubblesprogress "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type lectureAttendModel struct {
	ctx       context.Context
	cancel    context.CancelFunc
	service   lectureAttender
	row       app.LectureRow
	updates   chan tea.Msg
	progress  app.LectureProgress
	width     int
	height    int
	done      bool
	canceling bool
	err       error
}

type lectureAttender interface {
	AttendLecture(context.Context, string, app.LectureAttendOptions) (app.LectureAttendResult, error)
}

type lectureAttendProgressMsg struct {
	progress app.LectureProgress
}

type lectureAttendDoneMsg struct {
	result app.LectureAttendResult
	err    error
}

func (m lectureAttendModel) Init() tea.Cmd {
	return tea.Batch(m.run(), waitLectureAttendProgress(m.updates))
}

func (m lectureAttendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case lectureAttendProgressMsg:
		m.progress = msg.progress
		return m, waitLectureAttendProgress(m.updates)
	case lectureAttendDoneMsg:
		m.done = true
		m.err = msg.err
		if msg.err == nil {
			m.row = msg.result.Lecture
			m.progress = msg.result.Progress
		}
		m.cancel()
	}
	return m, nil
}

func (m lectureAttendModel) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	contentWidth := tuiContentWidth(width)
	labelWidth := maxInt(16, contentWidth-4)

	var b strings.Builder
	b.WriteString(renderHeaderTitle(contentWidth, "Attend"))
	b.WriteString("\n")
	b.WriteString(renderRule(contentWidth))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("LECTURE"))
	b.WriteString("\n")
	b.WriteString(truncateText(m.row.CourseName, labelWidth))
	b.WriteString("\n")
	b.WriteString(truncateText(firstLectureLabel(m.row), labelWidth))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(truncateText(m.row.ID, labelWidth)))
	b.WriteString("\n\n")
	b.WriteString(renderLectureAttendProgress(m.progress, maxInt(10, minInt(42, contentWidth-12))))
	b.WriteString("\n\n")

	switch {
	case m.err != nil && errors.Is(m.err, context.Canceled):
		b.WriteString(warnBadgeStyle.Render("CANCELED"))
		b.WriteString(" 수강을 중단했습니다\n")
	case m.err != nil:
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n")
	case m.done:
		b.WriteString(successBadgeStyle.Render("DONE"))
		b.WriteString(" 수강을 완료했습니다\n")
	case m.canceling:
		b.WriteString(warnBadgeStyle.Render("CANCEL"))
		b.WriteString(" 수강을 중단하는 중입니다\n")
	default:
		b.WriteString(warnBadgeStyle.Render("ATTENDING"))
		b.WriteString(" KLAS에 수강 진도를 반영하는 중입니다\n")
	}

	help := "esc 중단/뒤로  |  h 홈  |  q 종료"
	if m.done {
		help = "b/esc 강의 목록  |  h 홈  |  q 종료"
	}
	b.WriteString("\n")
	b.WriteString(renderHelpText(help, contentWidth))
	return appStyle.Render(b.String())
}

func (m lectureAttendModel) run() tea.Cmd {
	return func() tea.Msg {
		defer close(m.updates)
		result, err := m.service.AttendLecture(m.ctx, m.row.ID, app.LectureAttendOptions{
			OnProgress: func(_ app.LectureRow, progress app.LectureProgress) {
				select {
				case m.updates <- lectureAttendProgressMsg{progress: progress}:
				default:
				}
			},
		})
		return lectureAttendDoneMsg{result: result, err: err}
	}
}

func waitLectureAttendProgress(updates <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-updates
	}
}

func renderLectureAttendProgress(progress app.LectureProgress, width int) string {
	percent := boundedAttendPercent(progress.Progress)
	bar := bubblesprogress.New(
		bubblesprogress.WithWidth(width),
		bubblesprogress.WithFillCharacters('█', '░'),
	)
	total := emptyFallback(strings.TrimSpace(progress.TotalTime), "?")
	required := emptyFallback(strings.TrimSpace(progress.PTime), "?")
	return fmt.Sprintf("%s  %s/%s분", bar.ViewAs(percent/100), total, required)
}

func boundedAttendPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func firstLectureLabel(row app.LectureRow) string {
	for _, value := range []string{row.Lecture.Title, row.Lecture.ModuleTitle, "온라인 강의"} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "온라인 강의"
}

func validateLectureAttend(row app.LectureRow, now time.Time) error {
	if strings.TrimSpace(row.ID) == "" {
		return errors.New("선택한 강의의 ID가 없습니다")
	}
	if strings.TrimSpace(row.Lecture.ContentID) == "" && strings.TrimSpace(row.Lecture.LearningSeq) == "" {
		return errors.New("자동 수강을 지원하지 않는 강의입니다")
	}
	if lectureCompleted(row.Lecture) {
		return errors.New("이미 수강을 완료한 강의입니다")
	}
	if row.Lecture.StartAt != nil && now.Before(*row.Lecture.StartAt) {
		return errors.New("아직 수강 기간이 시작되지 않았습니다")
	}
	if row.Lecture.EndAt != nil && now.After(*row.Lecture.EndAt) {
		return errors.New("수강 기간이 종료되었습니다")
	}
	return nil
}

func initialAttendProgress(row app.LectureRow) app.LectureProgress {
	if strings.TrimSpace(row.Lecture.ContentID) != "" {
		return app.LectureProgress{Progress: parseAttendFloat(row.Lecture.Progress)}
	}
	achieved := parseAttendFloat(row.Lecture.AchievedTime)
	required := parseAttendFloat(row.Lecture.RequiredTime)
	percent := 0.0
	if required > 0 {
		percent = achieved / required * 100
	}
	return app.LectureProgress{
		TotalTime: row.Lecture.AchievedTime,
		PTime:     row.Lecture.RequiredTime,
		Progress:  boundedAttendPercent(percent),
	}
}

func parseAttendFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

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
