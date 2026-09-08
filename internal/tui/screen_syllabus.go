package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

type syllabusMsg struct {
	result app.SyllabusResult
	err    error
}

func (m syllabusScreenModel) Load(ctx context.Context, service syllabusScreenService, termValue string) tea.Cmd {
	courseIndex := m.syllabusCourseIndex
	return func() tea.Msg {
		result, err := service.Syllabus(ctx, app.SyllabusOptions{
			Selector:  strconv.Itoa(courseIndex),
			TermValue: termValue,
		})
		return syllabusMsg{result: result, err: err}
	}
}

func (m *syllabusScreenModel) moveCursor(delta, width, height int) {
	lines := syllabusLines(m.syllabusResult, width)
	if len(lines) == 0 {
		m.syllabusCursor = 0
		return
	}
	maxCursor := maxInt(0, len(lines)-visibleBodyRows(height, 0))
	m.syllabusCursor += delta
	if m.syllabusCursor < 0 {
		m.syllabusCursor = 0
	}
	if m.syllabusCursor > maxCursor {
		m.syllabusCursor = maxCursor
	}
}

func (m syllabusScreenModel) View(width, height int) string {
	lines := syllabusLines(m.syllabusResult, width)
	if len(lines) == 0 {
		return emptyStyle.Render("강의계획서 정보가 없습니다") + "\n"
	}
	return renderWindowedLines(lines, m.syllabusCursor, visibleBodyRows(height, 0))
}

func syllabusLines(result app.SyllabusResult, width int) []string {
	syllabus := result.Syllabus
	if strings.TrimSpace(result.SubjectID) == "" && strings.TrimSpace(syllabus.SubjectID) == "" {
		return nil
	}
	title := firstNonEmptyText(syllabus.FullName, syllabus.KoreanName, result.Course.Name, "과목명 확인 필요")
	lines := []string{
		sectionStyle.Render(title),
		mutedStyle.Render(strings.TrimSpace(result.Term.Label) + "  " + strings.TrimSpace(result.Term.Value)),
		"",
		"학정번호  " + emptyFallback(syllabus.CourseCode, "확인 필요"),
		"과목 ID  " + firstNonEmptyText(result.SubjectID, syllabus.SubjectID, "확인 필요"),
	}
	if syllabus.CourseType != "" || syllabus.Credits != "" {
		lines = append(lines, "이수/학점  "+emptyFallback(syllabus.CourseType, "-")+" / "+emptyFallback(syllabus.Credits, "-"))
	}
	if syllabus.Professor != "" {
		professor := syllabus.Professor
		if syllabus.ProfessorTitle != "" {
			professor += " (" + syllabus.ProfessorTitle + ")"
		}
		lines = append(lines, "담당교수  "+professor)
	}
	if len(syllabus.Times) > 0 {
		lines = append(lines, "강의시간  "+formatSyllabusTimes(syllabus.Times))
	}
	if syllabus.Operation != "" {
		lines = append(lines, "운영방식  "+syllabus.Operation)
	}
	if syllabus.Competency != "" {
		lines = append(lines, "대표역량  "+syllabus.Competency)
	}
	lines = append(lines, formatSyllabusEnrollment(syllabus.CurrentNum))
	for _, section := range []struct {
		title string
		text  string
	}{
		{title: "개요", text: syllabus.Summary},
		{title: "학습목표", text: syllabus.Purpose},
		{title: "학습성과", text: syllabus.Outcome},
	} {
		if strings.TrimSpace(section.text) == "" {
			continue
		}
		lines = append(lines, "", sectionStyle.Render(section.title))
		lines = appendWrappedLines(lines, section.text, width)
	}
	if syllabus.BookName != "" {
		lines = append(lines, "", sectionStyle.Render("교재"), syllabus.BookName)
	}
	lines = append(lines, "", sectionStyle.Render("평가"), formatSyllabusEvaluation(syllabus.Evaluation))
	if len(syllabus.Schedule) > 0 {
		lines = append(lines, "", sectionStyle.Render("주차별 계획"))
		for _, week := range syllabus.Schedule {
			label := fmt.Sprintf("%d주차", week.Week)
			lines = appendWrappedLines(lines, label+"  "+week.Topic, width)
			if strings.TrimSpace(week.SubNote) != "" {
				lines = appendWrappedLines(lines, "  "+week.SubNote, width)
			}
		}
	}
	return lines
}

func formatSyllabusTimes(times []app.SyllabusTime) string {
	parts := make([]string, 0, len(times))
	for _, item := range times {
		label := strings.TrimSpace(item.Weekday)
		if len(item.Periods) > 0 {
			periods := make([]string, 0, len(item.Periods))
			for _, period := range item.Periods {
				periods = append(periods, strconv.Itoa(period))
			}
			label += " " + strings.Join(periods, ",") + "교시"
		}
		if item.Room != "" {
			label += " (" + item.Room + ")"
		}
		parts = append(parts, strings.TrimSpace(label))
	}
	return strings.Join(parts, ", ")
}

func formatSyllabusEvaluation(evaluation app.SyllabusEvaluation) string {
	return fmt.Sprintf("출석 %d / 학습 %d / 중간 %d / 기말 %d / 과제 %d / 퀴즈 %d / 기타 %d",
		evaluation.Attendance,
		evaluation.Learning,
		evaluation.Midterm,
		evaluation.Final,
		evaluation.Report,
		evaluation.Quiz,
		evaluation.Other,
	)
}

func formatSyllabusEnrollment(currentNum string) string {
	count, err := strconv.Atoi(strings.TrimSpace(currentNum))
	if err != nil || count < 0 {
		return "수강인원: 확인 필요"
	}
	return fmt.Sprintf("수강인원: %d명 (A: %d명, B: %d명)", count, count*40/100, count*80/100)
}

type syllabusScreenModel struct {
	syllabusResult      app.SyllabusResult
	syllabusCourseIndex int
	syllabusCursor      int
}
type syllabusScreenService interface {
	Syllabus(context.Context, app.SyllabusOptions) (app.SyllabusResult, error)
}

func (m *syllabusScreenModel) Start(index int) { *m = syllabusScreenModel{syllabusCourseIndex: index} }
func (m *syllabusScreenModel) Loaded(result app.SyllabusResult) {
	m.syllabusResult = result
	m.syllabusCursor = 0
}
func (m *syllabusScreenModel) resetCursor() { m.syllabusCursor = 0 }
func (m *syllabusScreenModel) Update(msg tea.Msg, width, height int, loading bool, ctx context.Context, service syllabusScreenService, termValue string) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	if keyMatches(key.String(), "r", "ㄱ") {
		m.resetCursor()
		return childAction{setLoading: true, loading: true, setError: true, cmd: m.Load(ctx, service, termValue)}, true
	}
	if loading {
		return childAction{}, false
	}
	switch {
	case key.String() == "up" || keyMatches(key.String(), "k", "ㅏ"):
		m.moveCursor(-1, width, height)
	case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
		m.moveCursor(1, width, height)
	default:
		return childAction{}, false
	}
	return childAction{}, true
}
func (m model) renderSyllabusPanel(width int) string { return m.syllabus.View(width, m.height) }
