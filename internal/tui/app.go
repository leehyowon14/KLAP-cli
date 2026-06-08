package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kw-klap/klap-cli/internal/app"
	"github.com/kw-klap/klap-cli/internal/klas"
)

type screen int

const (
	screenHome screen = iota
	screenDashboard
	screenDue
	screenAssignments
	screenNotices
	screenLectures
	screenConfig
)

type menuItem struct {
	title  string
	help   string
	screen screen
}

type model struct {
	ctx      context.Context
	service  *app.Service
	menu     []menuItem
	cursor   int
	active   screen
	loading  bool
	err      error
	content  string
	width    int
	height   int
	loadedAt time.Time
}

type loadMsg struct {
	screen  screen
	content string
	err     error
}

func Run(ctx context.Context, service *app.Service) error {
	if service == nil {
		return errors.New("TUI service가 없습니다")
	}
	initial := model{
		ctx:     ctx,
		service: service,
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
			{title: "Due", help: "다가오는 과제/강의/학사일정", screen: screenDue},
			{title: "Assignments", help: "과제 목록", screen: screenAssignments},
			{title: "Notices", help: "강의 공지", screen: screenNotices},
			{title: "Lectures", help: "온라인 강의 상태", screen: screenLectures},
			{title: "Config", help: "현재 설정", screen: screenConfig},
		},
	}
	_, err := tea.NewProgram(initial).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "b":
			if m.active != screenHome {
				m.active = screenHome
				m.err = nil
				m.content = ""
				m.loading = false
			}
		case "up", "k":
			if m.active == screenHome && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.active == screenHome && m.cursor < len(m.menu)-1 {
				m.cursor++
			}
		case "enter":
			if m.active == screenHome && len(m.menu) > 0 {
				target := m.menu[m.cursor].screen
				m.active = target
				m.loading = true
				m.err = nil
				m.content = ""
				return m, m.load(target, false)
			}
		case "r":
			if m.active != screenHome {
				m.loading = true
				m.err = nil
				m.content = ""
				return m, m.load(m.active, true)
			}
		}
	case loadMsg:
		if msg.screen != m.active {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.content = msg.content
		m.loadedAt = time.Now()
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("KLAP TUI\n")
	b.WriteString("q: 종료  enter: 열기  b/esc: 뒤로  r: 새로고침\n\n")

	if m.active == screenHome {
		for index, item := range m.menu {
			prefix := "  "
			if index == m.cursor {
				prefix = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s - %s\n", prefix, item.title, item.help))
		}
		return b.String()
	}

	b.WriteString(screenTitle(m.active))
	if !m.loadedAt.IsZero() && !m.loading {
		b.WriteString(" | ")
		b.WriteString(m.loadedAt.Format("15:04:05"))
	}
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("불러오는 중...\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString("오류: ")
		b.WriteString(m.err.Error())
		b.WriteString("\n")
		return b.String()
	}
	if strings.TrimSpace(m.content) == "" {
		b.WriteString("표시할 내용이 없습니다\n")
		return b.String()
	}
	b.WriteString(m.content)
	if !strings.HasSuffix(m.content, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) load(target screen, refresh bool) tea.Cmd {
	return func() tea.Msg {
		content, err := m.loadContent(target, refresh)
		return loadMsg{screen: target, content: content, err: err}
	}
}

func (m model) loadContent(target screen, refresh bool) (string, error) {
	switch target {
	case screenDashboard:
		result, err := m.service.Dashboard(m.ctx, app.DashboardOptions{Refresh: refresh})
		if err != nil {
			return "", err
		}
		return formatDashboard(result), nil
	case screenDue:
		result, err := m.service.Due(m.ctx, app.DueOptions{Days: 14, Refresh: refresh})
		if err != nil {
			return "", err
		}
		return formatDue(result), nil
	case screenAssignments:
		rows, err := m.service.AssignmentList(m.ctx, app.AssignmentListOptions{Refresh: refresh})
		if err != nil {
			return "", err
		}
		return formatAssignments(rows), nil
	case screenNotices:
		rows, err := m.service.NoticeList(m.ctx, app.NoticeListOptions{Refresh: refresh})
		if err != nil {
			return "", err
		}
		return formatNotices(rows), nil
	case screenLectures:
		rows, err := m.service.LectureList(m.ctx, app.LectureListOptions{Refresh: refresh})
		if err != nil {
			return "", err
		}
		return formatLectures(rows), nil
	case screenConfig:
		settings, err := m.service.ConfigSettings()
		if err != nil {
			return "", err
		}
		return formatConfig(settings), nil
	default:
		return "", errors.New("지원하지 않는 TUI 화면입니다")
	}
}

func screenTitle(value screen) string {
	switch value {
	case screenDashboard:
		return "Dashboard"
	case screenDue:
		return "Due"
	case screenAssignments:
		return "Assignments"
	case screenNotices:
		return "Notices"
	case screenLectures:
		return "Lectures"
	case screenConfig:
		return "Config"
	default:
		return "Home"
	}
}

func formatDashboard(result app.DashboardResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s (%s)\n", result.Term.Label, result.Term.Value))
	if result.Cached {
		b.WriteString(fmt.Sprintf("캐시 사용: %s\n", result.CacheCreatedAt.Format("2006-01-02 15:04")))
	}
	b.WriteString("\n과제\n")
	if len(result.Assignments) == 0 {
		b.WriteString("  예정된 과제가 없습니다\n")
	} else {
		for _, row := range result.Assignments {
			b.WriteString(fmt.Sprintf("  %s | %s | %s\n", formatTime(row.Assignment.DueAt), row.CourseName, row.Assignment.Title))
		}
	}
	b.WriteString("\n온라인 강의\n")
	if len(result.Lectures) == 0 {
		b.WriteString("  수강할 온라인 강의가 없습니다\n")
	} else {
		for _, row := range result.Lectures {
			b.WriteString(fmt.Sprintf("  %s | %s | %s\n", formatTime(row.Lecture.EndAt), row.CourseName, row.Lecture.Title))
		}
	}
	b.WriteString("\n공지\n")
	if len(result.Notices) == 0 {
		b.WriteString("  공지가 없습니다\n")
	} else {
		for _, row := range result.Notices {
			b.WriteString(fmt.Sprintf("  %s | %s | %s\n", formatTime(row.Notice.Registered), row.CourseName, row.Notice.Title))
		}
	}
	if len(result.SectionErrors) > 0 {
		b.WriteString("\n확인 실패\n")
		for _, sectionError := range result.SectionErrors {
			b.WriteString(fmt.Sprintf("  %s: %v\n", sectionError.Section, sectionError.Err))
		}
	}
	return b.String()
}

func formatDue(result app.DueResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s ~ %s\n\n", result.From.Format("2006-01-02"), result.Until.Format("2006-01-02")))
	if len(result.Items) == 0 {
		b.WriteString("예정된 데드라인이 없습니다\n")
	}
	for _, item := range result.Items {
		course := item.CourseName
		if course == "" {
			course = "-"
		}
		b.WriteString(fmt.Sprintf("%s | %s | %s | %s\n", item.DueAt.Format("2006-01-02 15:04"), item.Kind, course, item.Title))
	}
	if len(result.Errors) > 0 {
		b.WriteString("\n확인 실패\n")
		for _, sectionError := range result.Errors {
			b.WriteString(fmt.Sprintf("  %s: %v\n", sectionError.Section, sectionError.Err))
		}
	}
	return b.String()
}

func formatAssignments(rows []app.AssignmentRow) string {
	if len(rows) == 0 {
		return "과제가 없습니다\n"
	}
	var b strings.Builder
	for _, row := range rows {
		status := "미제출"
		if row.Assignment.Submitted {
			status = "제출"
		}
		b.WriteString(fmt.Sprintf("%s | %s | %s | %s | %s\n", row.ID, formatTime(row.Assignment.DueAt), status, row.CourseName, row.Assignment.Title))
	}
	return b.String()
}

func formatNotices(rows []app.NoticeRow) string {
	if len(rows) == 0 {
		return "강의 공지가 없습니다\n"
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("%s | %s | %s | %s\n", row.ID, formatTime(row.Notice.Registered), row.CourseName, row.Notice.Title))
	}
	return b.String()
}

func formatLectures(rows []app.LectureRow) string {
	if len(rows) == 0 {
		return "온라인 강의가 없습니다\n"
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("%s | %s | %s | %s | %s\n", row.ID, lectureProgress(row.Lecture), row.CourseName, row.Lecture.ModuleTitle, row.Lecture.Title))
	}
	return b.String()
}

func formatConfig(settings app.ConfigSettings) string {
	term := strings.TrimSpace(settings.Term.Value)
	if term == "" {
		term = "자동"
	}
	return fmt.Sprintf("term: %s\nreminder.name: %s\nreminder.use-existing-list: %t\ndownload.dir: %s\n",
		term,
		settings.Reminder.ListName,
		settings.Reminder.UseExistingList,
		settings.Download.Dir,
	)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}

func lectureProgress(lecture klas.Lecture) string {
	if strings.TrimSpace(lecture.ContentID) != "" {
		progress := strings.TrimSpace(lecture.Progress)
		if progress == "" {
			return "진도 확인 필요"
		}
		return progress + "%"
	}
	achieved := strings.TrimSpace(lecture.AchievedTime)
	if achieved == "" {
		achieved = "0"
	}
	required := strings.TrimSpace(lecture.RequiredTime)
	if required == "" {
		required = "?"
	}
	return achieved + "/" + required + "분"
}
