package tui

import (
	"context"
	"errors"
	"fmt"
	bubblesprogress "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
	"time"
)

type lectureAttendModel struct {
	ctx             context.Context
	cancel          context.CancelFunc
	service         lectureAttender
	row             app.LectureRow
	updates         chan tea.Msg
	progress        app.LectureProgress
	width           int
	height          int
	done            bool
	canceling       bool
	err             error
	requireEligible bool
}

type lectureAttender interface {
	AttendLecture(context.Context, string, app.LectureAttendOptions) (app.LectureAttendResult, error)
}

type lectureAttendProgressMsg struct {
	updates  chan tea.Msg
	progress app.LectureProgress
}

type lectureAttendDoneMsg struct {
	updates chan tea.Msg
	result  app.LectureAttendResult
	err     error
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
		if (msg.updates != nil && msg.updates != m.updates) || m.done {
			return m, nil
		}
		m.progress = msg.progress
		return m, waitLectureAttendProgress(m.updates)
	case lectureAttendDoneMsg:
		if msg.updates != nil && msg.updates != m.updates {
			return m, nil
		}
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
			RequireEligible: m.requireEligible,
			OnProgress: func(_ app.LectureRow, progress app.LectureProgress) {
				select {
				case m.updates <- lectureAttendProgressMsg{updates: m.updates, progress: progress}:
				default:
				}
			},
		})
		return lectureAttendDoneMsg{updates: m.updates, result: result, err: err}
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
