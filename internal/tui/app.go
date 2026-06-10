package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kw-klap/klap-cli/internal/app"
	"github.com/kw-klap/klap-cli/internal/klas"
	settingspkg "github.com/kw-klap/klap-cli/internal/settings"
)

const menuNumberWidth = 3

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))
	headerMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781"))
	panelStyle = lipgloss.NewStyle().
			Padding(0, 0)
	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8C959F")).
			Padding(0, 0)
	menuSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#56D4DD"))
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781"))
	successTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9ACD32"))
	warnTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4A72C"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CF222E")).
			Bold(true)
	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781")).
			Italic(true)
	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#30363D")).
			Padding(0, 1)
	successBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#238636")).
				Padding(0, 1)
	warnBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#24292F")).
			Background(lipgloss.Color("#D4A72C")).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E7781"))
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#58A6FF"))
	taglineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ACD32"))
)

const klapLogo = ` _  __ _      _    ____
| |/ /| |    / \  |  _ \
| ' / | |   / _ \ | |_) |
| . \ | |__/ ___ \|  __/
|_|\_\|____/_/   \_\_|`

type screen int

const (
	screenHome screen = iota
	screenDashboard
	screenDue
	screenAssignments
	screenNotices
	screenLectures
	screenAssignmentDetail
	screenNoticeDetail
	screenAcademic
	screenConfig
	screenRoomDay
	screenRoomPeriod
	screenRoomResult
	screenDownloadSelect
	screenDownloadConfirm
	screenDownloadLanguage
	screenDownloadProgress
)

type menuItem struct {
	title  string
	help   string
	screen screen
}

type model struct {
	ctx                 context.Context
	service             *app.Service
	menu                []menuItem
	cursor              int
	active              screen
	loading             bool
	err                 error
	content             string
	dashboardResult     app.DashboardResult
	dashboardPage       int
	assignmentRows      []app.AssignmentRow
	noticeRows          []app.NoticeRow
	lectureRows         []app.LectureRow
	dueResult           app.DueResult
	duePage             int
	dueCursor           int
	academicResult      app.AcademicListResult
	academicMonth       int
	assignmentDetail    app.AssignmentDetailResult
	noticeDetail        app.NoticeDetailResult
	detailBack          screen
	detailCursor        int
	contentCourse       int
	contentCursor       int
	syncStatus          string
	width               int
	height              int
	loadedAt            time.Time
	configEditing       string
	configInput         textinput.Model
	configSettings      app.ConfigSettings
	configOptions       app.CategoryOptions
	configCursor        int
	downloadRows        []app.LectureRow
	downloadSelected    map[string]bool
	downloadCourse      int
	downloadCursor      int
	downloadTranscribe  bool
	downloadLanguage    int
	downloadProgress    *lectureDownloadModel
	roomDayCursor       int
	roomDaysSelected    map[int]bool
	roomPeriodCursor    int
	roomPeriodsSelected map[int]bool
}

type transcriptLanguage struct {
	label  string
	locale string
}

type configRow struct {
	key      string
	section  string
	label    string
	value    string
	hint     string
	editable bool
	cycle    bool
	reset    bool
}

var transcriptLanguages = []transcriptLanguage{
	{label: "한국어", locale: "ko-KR"},
	{label: "English", locale: "en-US"},
	{label: "日本語", locale: "ja-JP"},
	{label: "中文", locale: "zh-CN"},
	{label: "Deutsch", locale: "de-DE"},
	{label: "Français", locale: "fr-FR"},
	{label: "Español", locale: "es-ES"},
}

type roomDayOption struct {
	weekday int
	label   string
}

var roomDayOptions = []roomDayOption{
	{weekday: 1, label: "월요일"},
	{weekday: 2, label: "화요일"},
	{weekday: 3, label: "수요일"},
	{weekday: 4, label: "목요일"},
	{weekday: 5, label: "금요일"},
}

type loadMsg struct {
	screen      screen
	content     string
	err         error
	assignments []app.AssignmentRow
	notices     []app.NoticeRow
	lectures    []app.LectureRow
	due         app.DueResult
	academic    app.AcademicListResult
	dashboard   app.DashboardResult
	config      app.ConfigSettings
	categories  app.CategoryOptions
}

type downloadRowsMsg struct {
	rows []app.LectureRow
	err  error
}

type syncMsg struct {
	status string
	err    error
}

type detailMsg struct {
	screen     screen
	assignment app.AssignmentDetailResult
	notice     app.NoticeDetailResult
	err        error
}

type roomAvailableResultsMsg struct {
	results []app.RoomAvailableResult
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
			{title: "Due", help: "다가오는 일정", screen: screenDue},
			{title: "Assignments", help: "과제 목록", screen: screenAssignments},
			{title: "Notices", help: "공지 목록", screen: screenNotices},
			{title: "Lectures", help: "강의 상태", screen: screenLectures},
			{title: "Academic", help: "학사일정", screen: screenAcademic},
			{title: "Rooms", help: "빈 강의실 조회", screen: screenRoomDay},
			{title: "Config", help: "설정", screen: screenConfig},
		},
	}
	_, err := tea.NewProgram(initial, tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.active == screenDownloadProgress {
		return m.updateDownloadProgress(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.active == screenDownloadSelect {
			return m.updateDownloadSelect(msg)
		}
		if m.active == screenDownloadConfirm {
			return m.updateDownloadConfirm(msg)
		}
		if m.active == screenDownloadLanguage {
			return m.updateDownloadLanguage(msg)
		}
		if m.active == screenRoomDay {
			return m.updateRoomDay(msg)
		}
		if m.active == screenRoomPeriod {
			return m.updateRoomPeriod(msg)
		}
		if m.configEditing != "" {
			return m.updateConfigInput(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
		key := msg.String()
		switch {
		case keyMatches(key, "q", "ㅂ"):
			return m, tea.Quit
		case key == "esc" || keyMatches(key, "b", "ㅠ"):
			if m.active == screenRoomResult {
				m.active = screenRoomPeriod
				m.err = nil
				m.content = ""
				m.loading = false
			} else if m.isDetailScreen() {
				m.active = m.detailBack
				m.err = nil
				m.loading = false
				m.detailCursor = 0
			} else if m.active != screenHome {
				m.active = screenHome
				m.err = nil
				m.content = ""
				m.loading = false
			}
		case key == "up" || (keyMatches(key, "k", "ㅏ") && !m.canOpenKlasURL()):
			if m.active == screenHome && m.cursor > 0 {
				m.cursor--
			} else if m.active == screenConfig && !m.loading {
				m.moveConfigCursor(-1)
			} else if m.isDetailScreen() && !m.loading {
				m.moveDetailCursor(-1)
			} else if m.active == screenDue && !m.loading {
				m.moveDueCursor(-1)
			} else if m.isCoursePagedScreen() && !m.loading {
				m.moveContentCursor(-1)
			}
		case key == "down" || keyMatches(key, "j", "ㅓ"):
			if m.active == screenHome && m.cursor < len(m.menu)-1 {
				m.cursor++
			} else if m.active == screenConfig && !m.loading {
				m.moveConfigCursor(1)
			} else if m.isDetailScreen() && !m.loading {
				m.moveDetailCursor(1)
			} else if m.active == screenDue && !m.loading {
				m.moveDueCursor(1)
			} else if m.isCoursePagedScreen() && !m.loading {
				m.moveContentCursor(1)
			}
		case key == "left":
			if m.active == screenConfig && !m.loading {
				return m.adjustConfigCurrent(-1)
			} else if m.active == screenDashboard && !m.loading {
				m.moveDashboardPage(-1)
			} else if m.active == screenDue && !m.loading {
				m.moveDuePage(-1)
			} else if m.active == screenAcademic && !m.loading {
				m.moveAcademicMonth(-1)
			} else if m.isCoursePagedScreen() && !m.loading {
				m.moveContentCourse(-1)
			}
		case key == "right":
			if m.active == screenConfig && !m.loading {
				return m.adjustConfigCurrent(1)
			} else if m.active == screenDashboard && !m.loading {
				m.moveDashboardPage(1)
			} else if m.active == screenDue && !m.loading {
				m.moveDuePage(1)
			} else if m.active == screenAcademic && !m.loading {
				m.moveAcademicMonth(1)
			} else if m.isCoursePagedScreen() && !m.loading {
				m.moveContentCourse(1)
			}
		case key == "enter":
			if m.active == screenHome && len(m.menu) > 0 {
				target := m.menu[m.cursor].screen
				if target == screenRoomDay {
					return m.startRoomFlow()
				}
				m.active = target
				m.loading = true
				m.err = nil
				m.content = ""
				return m, m.load(target, false)
			}
			if m.active == screenAssignments && !m.loading {
				return m.openAssignmentDetail()
			}
			if m.active == screenNotices && !m.loading {
				return m.openNoticeDetail()
			}
			if m.active == screenConfig && !m.loading {
				return m.activateConfigCurrent()
			}
		case keyMatches(key, "r", "ㄱ"):
			if m.active == screenRoomResult {
				m.loading = true
				m.err = nil
				m.content = ""
				return m, m.loadRoomAvailableResults(true)
			}
			if m.active != screenHome {
				m.loading = true
				m.err = nil
				m.content = ""
				m.syncStatus = ""
				return m, m.load(m.active, true)
			}
		case keyMatches(key, "s", "ㄴ"):
			if cmd := m.syncCurrentScreen(); cmd != nil {
				m.loading = true
				m.err = nil
				m.syncStatus = "동기화 중..."
				return m, cmd
			}
		case keyMatches(key, "k", "ㅏ"):
			if cmd := m.openCurrentKlasURL(); cmd != nil {
				m.syncStatus = "KLAS 원문을 여는 중..."
				return m, cmd
			}
		case m.active == screenLectures && keyMatches(key, "d", "ㅇ"):
			m.active = screenDownloadSelect
			m.loading = true
			m.err = nil
			m.content = ""
			return m, m.loadDownloadRows()
		}
	case loadMsg:
		if msg.screen != m.active {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.content = msg.content
		m.dashboardResult = msg.dashboard
		m.assignmentRows = msg.assignments
		m.noticeRows = msg.notices
		m.lectureRows = msg.lectures
		m.dueResult = msg.due
		m.academicResult = msg.academic
		m.configSettings = msg.config
		m.configOptions = msg.categories
		if msg.screen == screenDashboard {
			m.dashboardPage = 0
		}
		if msg.screen == screenDue {
			m.duePage = 0
			m.dueCursor = 0
		}
		if msg.screen == screenAcademic {
			m.academicMonth = defaultAcademicMonth(msg.academic.Events, time.Now())
		}
		if msg.screen == screenConfig && m.configCursor >= len(configRows(msg.config, msg.categories)) {
			m.configCursor = maxInt(0, len(configRows(msg.config, msg.categories))-1)
		}
		m.contentCourse = 0
		m.contentCursor = 0
		m.syncStatus = ""
		m.loadedAt = time.Now()
	case syncMsg:
		m.loading = false
		m.err = msg.err
		m.syncStatus = msg.status
		m.loadedAt = time.Now()
	case detailMsg:
		if msg.screen != m.active {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.assignmentDetail = msg.assignment
		m.noticeDetail = msg.notice
		m.detailCursor = 0
		m.syncStatus = ""
		m.loadedAt = time.Now()
	case downloadRowsMsg:
		if m.active != screenDownloadSelect {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.downloadRows = msg.rows
		m.downloadSelected = make(map[string]bool, len(msg.rows))
		m.downloadCourse = 0
		m.downloadCursor = 0
	case roomAvailableResultsMsg:
		if m.active != screenRoomResult {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.content = formatRoomAvailableResults(msg.results)
		m.loadedAt = time.Now()
	}
	return m, nil
}

func keyMatches(value string, keys ...string) bool {
	for _, key := range keys {
		if value == key {
			return true
		}
	}
	return false
}

func (m model) isDetailScreen() bool {
	return m.active == screenAssignmentDetail || m.active == screenNoticeDetail
}

func (m model) canOpenKlasURL() bool {
	switch m.active {
	case screenAssignments, screenNotices, screenLectures, screenAssignmentDetail, screenNoticeDetail:
		return true
	default:
		return false
	}
}

func (m model) openAssignmentDetail() (tea.Model, tea.Cmd) {
	row, ok := m.selectedAssignmentRow()
	if !ok {
		m.syncStatus = "선택된 과제가 없습니다"
		return m, nil
	}
	m.active = screenAssignmentDetail
	m.detailBack = screenAssignments
	m.loading = true
	m.err = nil
	m.syncStatus = ""
	return m, m.loadAssignmentDetail(row.ID)
}

func (m model) openNoticeDetail() (tea.Model, tea.Cmd) {
	row, ok := m.selectedNoticeRow()
	if !ok {
		m.syncStatus = "선택된 공지가 없습니다"
		return m, nil
	}
	m.active = screenNoticeDetail
	m.detailBack = screenNotices
	m.loading = true
	m.err = nil
	m.syncStatus = ""
	return m, m.loadNoticeDetail(row.ID)
}

func (m model) loadAssignmentDetail(id string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.AssignmentDetail(m.ctx, id, app.UserOption{})
		return detailMsg{screen: screenAssignmentDetail, assignment: result, err: err}
	}
}

func (m model) loadNoticeDetail(id string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.NoticeDetail(m.ctx, id, app.UserOption{})
		return detailMsg{screen: screenNoticeDetail, notice: result, err: err}
	}
}

func (m model) openCurrentKlasURL() tea.Cmd {
	url := ""
	switch m.active {
	case screenAssignments:
		if row, ok := m.selectedAssignmentRow(); ok {
			url = row.DetailURL
		}
	case screenNotices:
		if row, ok := m.selectedNoticeRow(); ok {
			url = row.DetailURL
		}
	case screenLectures:
		if row, ok := m.selectedLectureRow(); ok {
			if strings.TrimSpace(row.Lecture.PlayURL) == "" {
				id := row.ID
				return func() tea.Msg {
					result, err := m.service.LectureOpenURL(m.ctx, id, app.UserOption{})
					if err != nil {
						return syncMsg{status: "KLAS 원문 열기 실패", err: err}
					}
					if err := openExternalURL(result.URL); err != nil {
						return syncMsg{status: "KLAS 원문 열기 실패", err: err}
					}
					return syncMsg{status: "KLAS 원문을 열었습니다"}
				}
			}
			url = row.Lecture.PlayURL
		}
	case screenAssignmentDetail:
		url = m.assignmentDetail.DetailURL
	case screenNoticeDetail:
		url = m.noticeDetail.DetailURL
	}
	if strings.TrimSpace(url) == "" {
		return nil
	}
	return func() tea.Msg {
		if err := openExternalURL(url); err != nil {
			return syncMsg{status: "KLAS 원문 열기 실패", err: err}
		}
		return syncMsg{status: "KLAS 원문을 열었습니다"}
	}
}

func openExternalURL(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("열 URL이 없습니다")
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("KLAS URL 열기 실패: %w", err)
	}
	return nil
}

func (m model) selectedAssignmentRow() (app.AssignmentRow, bool) {
	group := m.currentContentGroup(m.width)
	return selectedRowByCourse(m.assignmentRows, group.name, m.contentCursor, func(row app.AssignmentRow) string {
		return row.CourseName
	})
}

func (m model) selectedNoticeRow() (app.NoticeRow, bool) {
	group := m.currentContentGroup(m.width)
	return selectedRowByCourse(m.noticeRows, group.name, m.contentCursor, func(row app.NoticeRow) string {
		return row.CourseName
	})
}

func (m model) selectedLectureRow() (app.LectureRow, bool) {
	group := m.currentContentGroup(m.width)
	return selectedRowByCourse(m.lectureRows, group.name, m.contentCursor, func(row app.LectureRow) string {
		return row.CourseName
	})
}

func selectedRowByCourse[T any](rows []T, courseName string, cursor int, course func(T) string) (T, bool) {
	var zero T
	if strings.TrimSpace(courseName) == "" || cursor < 0 {
		return zero, false
	}
	seen := 0
	for _, row := range rows {
		name := strings.TrimSpace(course(row))
		if name == "" {
			name = "과목 확인 필요"
		}
		if name != courseName {
			continue
		}
		if seen == cursor {
			return row, true
		}
		seen++
	}
	return zero, false
}

func (m model) syncCurrentScreen() tea.Cmd {
	switch m.active {
	case screenDashboard:
		return m.syncDashboard()
	case screenDue:
		if m.duePage == 3 {
			return m.syncAcademic()
		}
		return nil
	case screenAssignments:
		return m.syncAssignments()
	case screenLectures:
		return m.syncLectures()
	case screenAcademic:
		return m.syncAcademic()
	default:
		return nil
	}
}

func (m model) syncDashboard() tea.Cmd {
	return func() tea.Msg {
		assignments, assignmentErr := m.service.SyncAssignmentReminders(m.ctx, app.AssignmentListOptions{})
		lectures, lectureErr := m.service.SyncLectureReminders(m.ctx, app.LectureListOptions{})
		academic, academicErr := m.service.SyncAcademicCalendar(m.ctx, app.AcademicListOptions{})
		parts := []string{
			formatReminderSyncStatus("과제", assignments),
			formatReminderSyncStatus("강의", lectures),
			formatCalendarSyncStatus("학사일정", academic),
		}
		if err := firstErr(assignmentErr, lectureErr, academicErr); err != nil {
			return syncMsg{status: strings.Join(parts, " / "), err: err}
		}
		return syncMsg{status: "동기화 완료: " + strings.Join(parts, " / ")}
	}
}

func (m model) syncAssignments() tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.SyncAssignmentReminders(m.ctx, app.AssignmentListOptions{})
		return syncMsg{status: "과제 " + formatReminderSyncStatus("", result), err: err}
	}
}

func (m model) syncLectures() tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.SyncLectureReminders(m.ctx, app.LectureListOptions{})
		return syncMsg{status: "강의 " + formatReminderSyncStatus("", result), err: err}
	}
}

func (m model) syncAcademic() tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.SyncAcademicCalendar(m.ctx, app.AcademicListOptions{})
		return syncMsg{status: "학사일정 " + formatCalendarSyncStatus("", result), err: err}
	}
}

func formatReminderSyncStatus(label string, result app.ReminderSyncResult) string {
	prefix := ""
	if strings.TrimSpace(label) != "" {
		prefix = label + " "
	}
	if result.EligibleCount == 0 {
		return prefix + "대상 없음"
	}
	return fmt.Sprintf("%s생성 %d, 갱신 %d, 완료 %d, 제외 %d",
		prefix,
		result.Result.Created,
		result.Result.Updated,
		result.Result.Completed,
		result.Result.Skipped,
	)
}

func formatCalendarSyncStatus(label string, result app.CalendarSyncResult) string {
	prefix := ""
	if strings.TrimSpace(label) != "" {
		prefix = label + " "
	}
	if result.EligibleCount == 0 {
		return prefix + "대상 없음"
	}
	return fmt.Sprintf("%s생성 %d, 갱신 %d, 제외 %d",
		prefix,
		result.Result.Created,
		result.Result.Updated,
		result.Result.Skipped,
	)
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (m model) loadDownloadRows() tea.Cmd {
	return func() tea.Msg {
		rows, err := m.service.LectureList(m.ctx, app.LectureListOptions{Refresh: true})
		return downloadRowsMsg{rows: rows, err: err}
	}
}

func (m model) startRoomFlow() (tea.Model, tea.Cmd) {
	m.active = screenRoomDay
	m.loading = false
	m.err = nil
	m.content = ""
	m.roomDayCursor = 0
	m.roomPeriodCursor = 0
	m.roomDaysSelected = map[int]bool{}
	m.roomPeriodsSelected = map[int]bool{}
	return m, nil
}

func (m model) updateRoomDay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenHome
		m.err = nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if m.roomDayCursor > 0 {
			m.roomDayCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if m.roomDayCursor < len(roomDayOptions)-1 {
			m.roomDayCursor++
		}
	case key == " ":
		m.toggleRoomDayCurrent()
	case keyMatches(key, "a", "ㅁ"):
		m.toggleRoomAllDays()
	case key == "enter":
		if len(m.selectedRoomDays()) == 0 {
			m.err = errors.New("요일을 하나 이상 선택하세요")
			return m, nil
		}
		m.err = nil
		m.active = screenRoomPeriod
		m.roomPeriodCursor = 0
		if m.roomPeriodsSelected == nil {
			m.roomPeriodsSelected = map[int]bool{}
		}
	}
	return m, nil
}

func (m model) updateRoomPeriod(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenRoomDay
		m.err = nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if m.roomPeriodCursor > 0 {
			m.roomPeriodCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if m.roomPeriodCursor < 7 {
			m.roomPeriodCursor++
		}
	case key == " ":
		m.toggleRoomPeriodCurrent()
	case keyMatches(key, "a", "ㅁ"):
		m.toggleRoomAllPeriods()
	case key == "enter":
		if len(m.selectedRoomPeriods()) == 0 {
			m.err = errors.New("교시를 하나 이상 선택하세요")
			return m, nil
		}
		m.err = nil
		m.active = screenRoomResult
		m.loading = true
		m.content = ""
		return m, m.loadRoomAvailableResults(false)
	}
	return m, nil
}

func (m *model) toggleRoomDayCurrent() {
	if m.roomDaysSelected == nil {
		m.roomDaysSelected = map[int]bool{}
	}
	day := roomDayOptions[m.roomDayCursor].weekday
	m.roomDaysSelected[day] = !m.roomDaysSelected[day]
}

func (m *model) toggleRoomAllDays() {
	if m.roomDaysSelected == nil {
		m.roomDaysSelected = map[int]bool{}
	}
	allSelected := true
	for _, day := range roomDayOptions {
		if !m.roomDaysSelected[day.weekday] {
			allSelected = false
			break
		}
	}
	for _, day := range roomDayOptions {
		m.roomDaysSelected[day.weekday] = !allSelected
	}
}

func (m *model) toggleRoomPeriodCurrent() {
	if m.roomPeriodsSelected == nil {
		m.roomPeriodsSelected = map[int]bool{}
	}
	period := m.roomPeriodCursor + 1
	m.roomPeriodsSelected[period] = !m.roomPeriodsSelected[period]
}

func (m *model) toggleRoomAllPeriods() {
	if m.roomPeriodsSelected == nil {
		m.roomPeriodsSelected = map[int]bool{}
	}
	allSelected := true
	for period := 1; period <= 8; period++ {
		if !m.roomPeriodsSelected[period] {
			allSelected = false
			break
		}
	}
	for period := 1; period <= 8; period++ {
		m.roomPeriodsSelected[period] = !allSelected
	}
}

func (m model) selectedRoomDays() []int {
	days := make([]int, 0, len(roomDayOptions))
	for _, day := range roomDayOptions {
		if m.roomDaysSelected[day.weekday] {
			days = append(days, day.weekday)
		}
	}
	return days
}

func (m model) selectedRoomPeriods() []int {
	periods := make([]int, 0, 8)
	for period := 1; period <= 8; period++ {
		if m.roomPeriodsSelected[period] {
			periods = append(periods, period)
		}
	}
	return periods
}

func (m model) loadRoomAvailableResults(refresh bool) tea.Cmd {
	days := m.selectedRoomDays()
	periods := m.selectedRoomPeriods()
	return func() tea.Msg {
		results := make([]app.RoomAvailableResult, 0, len(days))
		for index, weekday := range days {
			result, err := m.service.RoomAvailable(m.ctx, app.RoomAvailableOptions{
				Refresh: refresh && index == 0,
				Day:     app.RoomWeekdayLabel(weekday),
				Periods: periods,
			})
			if err != nil {
				return roomAvailableResultsMsg{err: err}
			}
			results = append(results, result)
		}
		return roomAvailableResultsMsg{results: results}
	}
}

func (m model) updateDownloadSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenLectures
		m.loading = true
		m.err = nil
		return m, m.load(screenLectures, false)
	case m.loading:
		return m, nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		maxCursor := len(m.currentDownloadRows())
		if maxCursor <= 0 {
			m.downloadCursor = 0
		} else if m.downloadCursor <= 0 {
			m.downloadCursor = maxCursor
		} else {
			m.downloadCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		maxCursor := len(m.currentDownloadRows())
		if maxCursor <= 0 || m.downloadCursor >= maxCursor {
			m.downloadCursor = 0
		} else {
			m.downloadCursor++
		}
	case key == "left":
		if m.downloadCourse > 0 {
			m.downloadCourse--
			m.downloadCursor = 0
		}
	case key == "right":
		if m.downloadCourse < len(m.downloadGroups())-1 {
			m.downloadCourse++
			m.downloadCursor = 0
		}
	case key == " ":
		m.toggleDownloadCurrent()
	case keyMatches(key, "a", "ㅁ"):
		m.toggleDownloadAll()
	case key == "enter":
		if len(m.selectedDownloadIDs()) == 0 {
			m.err = errors.New("선택된 강의가 없습니다")
			return m, nil
		}
		m.err = nil
		m.active = screenDownloadConfirm
		m.downloadTranscribe = false
	}
	return m, nil
}

func (m model) updateDownloadConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenDownloadSelect
	case keyMatches(key, "y", "ㅛ"):
		m.downloadTranscribe = true
		m.active = screenDownloadLanguage
		m.downloadLanguage = 0
	case keyMatches(key, "n", "ㅜ"):
		m.downloadTranscribe = false
		return m.startDownloadProgress()
	case key == "left" || key == "right" || key == "tab":
		m.downloadTranscribe = !m.downloadTranscribe
	case key == "enter":
		if m.downloadTranscribe {
			m.active = screenDownloadLanguage
			m.downloadLanguage = 0
			return m, nil
		}
		return m.startDownloadProgress()
	}
	return m, nil
}

func (m model) updateDownloadLanguage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenDownloadConfirm
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if len(transcriptLanguages) == 0 {
			m.downloadLanguage = 0
		} else if m.downloadLanguage <= 0 {
			m.downloadLanguage = len(transcriptLanguages) - 1
		} else {
			m.downloadLanguage--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if len(transcriptLanguages) == 0 || m.downloadLanguage >= len(transcriptLanguages)-1 {
			m.downloadLanguage = 0
		} else {
			m.downloadLanguage++
		}
	case key == "enter":
		return m.startDownloadProgress()
	}
	return m, nil
}

func (m model) startDownloadProgress() (tea.Model, tea.Cmd) {
	settings, err := m.service.DownloadSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	transcriptSettings, err := m.service.TranscriptSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	selectedRows := m.selectedDownloadRows()
	runCtx, cancel := context.WithCancel(m.ctx)
	progress := lectureDownloadModel{
		ctx:     runCtx,
		cancel:  cancel,
		service: m.service,
		request: LectureDownloadRequest{
			All:                   true,
			Rows:                  selectedRows,
			LectureIDs:            m.selectedDownloadIDs(),
			Concurrency:           settings.Concurrency,
			Transcribe:            m.downloadTranscribe,
			TranscriptLocale:      m.selectedTranscriptLocale(),
			TranscriptConcurrency: transcriptSettings.Concurrency,
		},
		updates:           make(chan tea.Msg, 64),
		items:             initialDownloadStatusLines(selectedRows),
		startedAt:         time.Now(),
		transcriptStarted: make(map[string]bool),
		transcriptRunning: make(map[string]bool),
	}
	m.active = screenDownloadProgress
	m.err = nil
	m.downloadProgress = &progress
	return m, progress.Init()
}

func (m model) updateDownloadProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.downloadProgress == nil {
		m.active = screenLectures
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "ctrl+c" || keyMatches(key.String(), "q", "ㅂ") {
			m.downloadProgress.cancel()
			return m, tea.Quit
		}
		if keyMatches(key.String(), "h", "ㅗ") {
			if !m.downloadProgress.done {
				m.downloadProgress.cancel()
			}
			m.active = screenHome
			m.loading = false
			m.err = nil
			m.content = ""
			m.downloadProgress = nil
			return m, nil
		}
		if m.downloadProgress.done && (key.String() == "esc" || keyMatches(key.String(), "b", "ㅠ")) {
			m.active = screenLectures
			m.loading = true
			m.downloadProgress = nil
			return m, m.load(screenLectures, true)
		}
		if key.String() == "esc" && !m.downloadProgress.done {
			m.downloadProgress.cancelAndCleanup()
			m.active = screenLectures
			m.loading = true
			m.downloadProgress = nil
			return m, m.load(screenLectures, true)
		}
	}
	updated, cmd := m.downloadProgress.Update(msg)
	progress, ok := updated.(lectureDownloadModel)
	if !ok {
		return m, cmd
	}
	m.downloadProgress = &progress
	return m, cmd
}

func (m *model) toggleDownloadCurrent() {
	rows := m.currentDownloadRows()
	if m.downloadCursor == 0 {
		m.toggleDownloadCourse()
		return
	}
	rowIndex := m.downloadCursor - 1
	if rowIndex < 0 || rowIndex >= len(rows) {
		return
	}
	row := rows[rowIndex]
	if !lectureDownloadable(row) {
		return
	}
	m.downloadSelected[row.ID] = !m.downloadSelected[row.ID]
}

func (m *model) toggleDownloadCourse() {
	rows := m.currentDownloadRows()
	allSelected := true
	for _, row := range rows {
		if lectureDownloadable(row) && !m.downloadSelected[row.ID] {
			allSelected = false
			break
		}
	}
	for _, row := range rows {
		if lectureDownloadable(row) {
			m.downloadSelected[row.ID] = !allSelected
		}
	}
}

func (m *model) toggleDownloadAll() {
	allSelected := true
	hasDownloadable := false
	for _, row := range m.downloadRows {
		if !lectureDownloadable(row) {
			continue
		}
		hasDownloadable = true
		if !m.downloadSelected[row.ID] {
			allSelected = false
			break
		}
	}
	if !hasDownloadable {
		return
	}
	for _, row := range m.downloadRows {
		if lectureDownloadable(row) {
			m.downloadSelected[row.ID] = !allSelected
		}
	}
}

func (m model) selectedDownloadIDs() []string {
	ids := make([]string, 0)
	for _, row := range m.downloadRows {
		if m.downloadSelected[row.ID] {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

func (m model) selectedDownloadRows() []app.LectureRow {
	rows := make([]app.LectureRow, 0)
	for _, row := range m.downloadRows {
		if m.downloadSelected[row.ID] {
			rows = append(rows, row)
		}
	}
	return rows
}

func (m model) selectedTranscriptLocale() string {
	if m.downloadLanguage < 0 || m.downloadLanguage >= len(transcriptLanguages) {
		return transcriptLanguages[0].locale
	}
	return transcriptLanguages[m.downloadLanguage].locale
}

type downloadCourseGroup struct {
	name string
	rows []app.LectureRow
}

type contentCourseGroup struct {
	name  string
	lines []string
}

func (m model) downloadGroups() []downloadCourseGroup {
	groups := make([]downloadCourseGroup, 0)
	indexByName := make(map[string]int)
	for _, row := range m.downloadRows {
		name := strings.TrimSpace(row.CourseName)
		if name == "" {
			name = "과목 확인 필요"
		}
		index, ok := indexByName[name]
		if !ok {
			index = len(groups)
			indexByName[name] = index
			groups = append(groups, downloadCourseGroup{name: name})
		}
		groups[index].rows = append(groups[index].rows, row)
	}
	return groups
}

func (m model) currentDownloadGroup() downloadCourseGroup {
	groups := m.downloadGroups()
	if len(groups) == 0 {
		return downloadCourseGroup{}
	}
	index := m.downloadCourse
	if index < 0 {
		index = 0
	}
	if index >= len(groups) {
		index = len(groups) - 1
	}
	return groups[index]
}

func (m model) currentDownloadRows() []app.LectureRow {
	return m.currentDownloadGroup().rows
}

func (m model) currentDownloadCourseSelected() bool {
	rows := m.currentDownloadRows()
	hasDownloadable := false
	for _, row := range rows {
		if !lectureDownloadable(row) {
			continue
		}
		hasDownloadable = true
		if !m.downloadSelected[row.ID] {
			return false
		}
	}
	return hasDownloadable
}

func (m model) isCoursePagedScreen() bool {
	switch m.active {
	case screenAssignments, screenNotices, screenLectures:
		return true
	default:
		return false
	}
}

func (m model) contentGroups(width int) []contentCourseGroup {
	lineWidth := maxInt(24, minInt(92, width-12))
	switch m.active {
	case screenAssignments:
		return assignmentContentGroups(m.assignmentRows, lineWidth)
	case screenNotices:
		return noticeContentGroups(m.noticeRows, lineWidth)
	case screenLectures:
		return lectureContentGroups(m.lectureRows, lineWidth)
	default:
		return nil
	}
}

func (m model) currentContentGroup(width int) contentCourseGroup {
	groups := m.contentGroups(width)
	if len(groups) == 0 {
		return contentCourseGroup{}
	}
	index := m.contentCourse
	if index < 0 {
		index = 0
	}
	if index >= len(groups) {
		index = len(groups) - 1
	}
	return groups[index]
}

func (m *model) moveContentCourse(delta int) {
	groups := m.contentGroups(m.width)
	if len(groups) == 0 {
		m.contentCourse = 0
		m.contentCursor = 0
		return
	}
	m.contentCourse += delta
	if m.contentCourse < 0 {
		m.contentCourse = len(groups) - 1
	}
	if m.contentCourse >= len(groups) {
		m.contentCourse = 0
	}
	m.contentCursor = 0
}

func (m *model) moveContentCursor(delta int) {
	group := m.currentContentGroup(m.width)
	if len(group.lines) == 0 {
		m.contentCursor = 0
		return
	}
	m.contentCursor += delta
	if m.contentCursor < 0 {
		m.contentCursor = len(group.lines) - 1
	}
	if m.contentCursor >= len(group.lines) {
		m.contentCursor = 0
	}
}

func (m *model) moveDetailCursor(delta int) {
	lines := m.detailLines(m.width)
	if len(lines) == 0 {
		m.detailCursor = 0
		return
	}
	m.detailCursor += delta
	if m.detailCursor < 0 {
		m.detailCursor = len(lines) - 1
	}
	if m.detailCursor >= len(lines) {
		m.detailCursor = 0
	}
}

var duePageLabels = []string{"Summary", "과제", "온라인 강의", "학사일정"}

func (m *model) moveDashboardPage(delta int) {
	total := dashboardPageCount(m.dashboardResult)
	if total <= 0 {
		m.dashboardPage = 0
		return
	}
	m.dashboardPage += delta
	if m.dashboardPage < 0 {
		m.dashboardPage = total - 1
	}
	if m.dashboardPage >= total {
		m.dashboardPage = 0
	}
}

func (m *model) moveDuePage(delta int) {
	m.duePage += delta
	if m.duePage < 0 {
		m.duePage = len(duePageLabels) - 1
	}
	if m.duePage >= len(duePageLabels) {
		m.duePage = 0
	}
	m.dueCursor = 0
}

func (m *model) moveDueCursor(delta int) {
	lines := duePageLines(m.dueResult, m.duePage, m.width)
	if len(lines) == 0 {
		m.dueCursor = 0
		return
	}
	m.dueCursor += delta
	if m.dueCursor < 0 {
		m.dueCursor = len(lines) - 1
	}
	if m.dueCursor >= len(lines) {
		m.dueCursor = 0
	}
}

func (m *model) moveAcademicMonth(delta int) {
	m.academicMonth += delta
	if m.academicMonth < 1 {
		m.academicMonth = 12
	}
	if m.academicMonth > 12 {
		m.academicMonth = 1
	}
}

func assignmentContentGroups(rows []app.AssignmentRow, width int) []contentCourseGroup {
	groups := make([]contentCourseGroup, 0)
	indexByName := make(map[string]int)
	for _, row := range rows {
		groupIndex := contentGroupIndex(&groups, indexByName, row.CourseName)
		status := warnBadgeStyle.Render("미제출")
		if row.Assignment.Submitted {
			status = successBadgeStyle.Render("제출")
		}
		title := truncateText(row.Assignment.Title, maxInt(12, width-24))
		groups[groupIndex].lines = append(groups[groupIndex].lines, fmt.Sprintf("%s  %s  %s",
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			status,
			title,
		))
	}
	return groups
}

func noticeContentGroups(rows []app.NoticeRow, width int) []contentCourseGroup {
	groups := make([]contentCourseGroup, 0)
	indexByName := make(map[string]int)
	for _, row := range rows {
		groupIndex := contentGroupIndex(&groups, indexByName, row.CourseName)
		date := mutedStyle.Render(formatTime(row.Notice.Registered))
		badge := ""
		titleWidth := maxInt(12, width-19)
		if row.Notice.Top {
			date = warnTextStyle.Render(formatTime(row.Notice.Registered))
			badge = " " + warnBadgeStyle.Render("Pinned")
			titleWidth = maxInt(12, width-28)
		}
		title := truncateText(row.Notice.Title, titleWidth)
		groups[groupIndex].lines = append(groups[groupIndex].lines, fmt.Sprintf("%s  %s%s",
			date,
			title,
			badge,
		))
	}
	return groups
}

func lectureContentGroups(rows []app.LectureRow, width int) []contentCourseGroup {
	groups := make([]contentCourseGroup, 0)
	indexByName := make(map[string]int)
	progressWidth := 9
	moduleWidth := maxInt(18, minInt(34, width/3))
	for _, row := range rows {
		groupIndex := contentGroupIndex(&groups, indexByName, row.CourseName)
		progressText := lectureProgress(row.Lecture)
		progress := lectureProgressStyle(row.Lecture).Render(fixedColumn(progressText, progressWidth))
		module := emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요")
		module = fixedColumn(module, moduleWidth)
		titleWidth := maxInt(12, width-progressWidth-moduleWidth-6)
		title := truncateText(row.Lecture.Title, titleWidth)
		groups[groupIndex].lines = append(groups[groupIndex].lines, fmt.Sprintf("%s  %s  %s",
			progress,
			module,
			title,
		))
	}
	return groups
}

func contentGroupIndex(groups *[]contentCourseGroup, indexByName map[string]int, courseName string) int {
	name := strings.TrimSpace(courseName)
	if name == "" {
		name = "과목 확인 필요"
	}
	index, ok := indexByName[name]
	if !ok {
		index = len(*groups)
		indexByName[name] = index
		*groups = append(*groups, contentCourseGroup{name: name})
	}
	return index
}

func fixedColumn(value string, width int) string {
	value = truncateText(value, width)
	return lipgloss.NewStyle().Width(width).Render(value)
}

func lectureProgressStyle(lecture klas.Lecture) lipgloss.Style {
	if lectureCompleted(lecture) {
		return successTextStyle
	}
	return warnTextStyle
}

func lectureCompleted(lecture klas.Lecture) bool {
	if strings.TrimSpace(lecture.ContentID) != "" {
		progress, err := strconv.ParseFloat(strings.TrimSpace(lecture.Progress), 64)
		return err == nil && progress >= 100
	}
	achieved, achievedErr := strconv.ParseFloat(strings.TrimSpace(lecture.AchievedTime), 64)
	required, requiredErr := strconv.ParseFloat(strings.TrimSpace(lecture.RequiredTime), 64)
	return achievedErr == nil && requiredErr == nil && required > 0 && achieved >= required
}

func (m model) updateConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.configEditing = ""
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.configInput.Value())
		var err error
		switch m.configEditing {
		case "download.dir":
			_, err = m.service.SetDownloadConfig(value, 0, nil, nil)
		case "reminder.name":
			_, err = m.service.SetReminderConfig(value, false)
		case "calendar.name":
			_, err = m.service.SetCalendarConfig(value, false)
		case "reminder.alarm-before-min":
			_, err = m.service.SetConfigValue("reminder.alarm-before-min", value)
		}
		if err != nil {
			m.err = err
		} else {
			m.err = nil
			m.configEditing = ""
			m.refreshConfigContent()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.configInput, cmd = m.configInput.Update(msg)
	return m, cmd
}

func (m *model) moveConfigCursor(delta int) {
	rows := configRows(m.configSettings, m.configOptions)
	if len(rows) == 0 {
		m.configCursor = 0
		return
	}
	m.configCursor = (m.configCursor + delta + len(rows)) % len(rows)
}

func (m model) activateConfigCurrent() (tea.Model, tea.Cmd) {
	row, ok := m.currentConfigRow()
	if !ok {
		return m, nil
	}
	if row.reset {
		return m.resetConfigSettings()
	}
	if row.editable {
		return m.startConfigEdit(row)
	}
	return m.adjustConfigCurrent(1)
}

func (m model) adjustConfigCurrent(delta int) (tea.Model, tea.Cmd) {
	row, ok := m.currentConfigRow()
	if !ok {
		return m, nil
	}
	switch row.key {
	case "reminder.name":
		next, ok := cycleStringOption(m.configOptions.Reminders, m.configSettings.Reminder.ListName, delta)
		if !ok {
			return m.startConfigEdit(row)
		}
		if _, err := m.service.SetReminderConfig(next, true); err != nil {
			m.err = err
			return m, nil
		}
	case "calendar.name":
		next, ok := cycleStringOption(m.configOptions.Calendars, m.configSettings.Calendar.Name, delta)
		if !ok {
			return m.startConfigEdit(row)
		}
		if _, err := m.service.SetCalendarConfig(next, true); err != nil {
			m.err = err
			return m, nil
		}
	case "reminder.alarm-before-min":
		next := m.configSettings.Reminder.AlarmBeforeMin + delta*60
		if next < 1 {
			next = 1
		}
		if _, err := m.service.SetConfigValue("reminder.alarm-before-min", strconv.Itoa(next)); err != nil {
			m.err = err
			return m, nil
		}
	case "download.concurrency":
		return m.adjustDownloadConcurrency(delta)
	case "download.caffeinate":
		return m.toggleDownloadCaffeinate()
	case "download.keep-partial":
		return m.toggleDownloadKeepPartial()
	case "transcript.concurrency":
		return m.adjustTranscriptConcurrency(delta)
	default:
		return m, nil
	}
	m.err = nil
	m.refreshConfigContent()
	return m, nil
}

func (m model) startConfigEdit(row configRow) (tea.Model, tea.Cmd) {
	input := textinput.New()
	switch row.key {
	case "download.dir":
		input.SetValue(m.configSettings.Download.Dir)
		input.Placeholder = "다운로드 폴더"
	case "reminder.name":
		input.SetValue(m.configSettings.Reminder.ListName)
		input.Placeholder = "미리알림 목록"
	case "calendar.name":
		input.SetValue(m.configSettings.Calendar.Name)
		input.Placeholder = "캘린더"
	case "reminder.alarm-before-min":
		input.SetValue(strconv.Itoa(m.configSettings.Reminder.AlarmBeforeMin))
		input.Placeholder = "분 단위 알림 시간"
	default:
		input.SetValue(row.value)
		input.Placeholder = row.label
	}
	input.Prompt = row.key + " "
	input.Focus()
	m.configEditing = row.key
	m.configInput = input
	return m, nil
}

func (m model) startDownloadDirEdit() (tea.Model, tea.Cmd) {
	row := configRow{key: "download.dir", label: "다운로드 폴더", editable: true}
	return m.startConfigEdit(row)
}

func (m model) adjustDownloadConcurrency(delta int) (tea.Model, tea.Cmd) {
	settings, err := m.service.DownloadSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	next := settings.Concurrency + delta
	if next < 1 {
		next = 1
	}
	if _, err := m.service.SetDownloadConfig("", next, nil, nil); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil
	m.refreshConfigContent()
	return m, nil
}

func (m model) adjustTranscriptConcurrency(delta int) (tea.Model, tea.Cmd) {
	settings, err := m.service.TranscriptSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	next := settings.Concurrency + delta
	if next < 1 {
		next = 1
	}
	if next > settingspkg.MaxTranscriptConcurrency {
		next = settingspkg.MaxTranscriptConcurrency
	}
	if _, err := m.service.SetTranscriptConfig(next); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil
	m.refreshConfigContent()
	return m, nil
}

func (m model) toggleDownloadCaffeinate() (tea.Model, tea.Cmd) {
	settings, err := m.service.DownloadSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	next := !settings.Caffeinate
	if _, err := m.service.SetDownloadConfig("", 0, &next, nil); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil
	m.refreshConfigContent()
	return m, nil
}

func (m model) toggleDownloadKeepPartial() (tea.Model, tea.Cmd) {
	settings, err := m.service.DownloadSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	next := !settings.KeepPartial
	if _, err := m.service.SetDownloadConfig("", 0, nil, &next); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil
	m.refreshConfigContent()
	return m, nil
}

func (m model) resetConfigSettings() (tea.Model, tea.Cmd) {
	if _, err := m.service.ResetConfigSettings(); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil
	m.refreshConfigContent()
	return m, nil
}

func (m *model) refreshConfigContent() {
	settings, err := m.service.ConfigSettings()
	if err != nil {
		m.err = err
		return
	}
	options, err := m.service.CategoryOptions()
	if err != nil {
		m.err = err
		return
	}
	m.configSettings = settings
	m.configOptions = options
	rows := configRows(settings, options)
	if m.configCursor >= len(rows) {
		m.configCursor = maxInt(0, len(rows)-1)
	}
	m.content = formatConfig(settings)
	m.loadedAt = time.Now()
}

func (m model) currentConfigRow() (configRow, bool) {
	rows := configRows(m.configSettings, m.configOptions)
	if len(rows) == 0 {
		return configRow{}, false
	}
	if m.configCursor < 0 {
		return rows[0], true
	}
	if m.configCursor >= len(rows) {
		return rows[len(rows)-1], true
	}
	return rows[m.configCursor], true
}

func (m model) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	switch m.active {
	case screenDownloadSelect:
		return appStyle.Render(m.renderDownloadSelectView(width))
	case screenDownloadConfirm:
		return appStyle.Render(m.renderDownloadConfirmView(width))
	case screenDownloadLanguage:
		return appStyle.Render(m.renderDownloadLanguageView(width))
	case screenDownloadProgress:
		if m.downloadProgress == nil {
			return appStyle.Render(errorStyle.Render("다운로드 상태가 없습니다"))
		}
		return m.downloadProgress.View()
	case screenRoomDay:
		return appStyle.Render(m.renderRoomDayView(width))
	case screenRoomPeriod:
		return appStyle.Render(m.renderRoomPeriodView(width))
	}
	if m.active == screenHome {
		return appStyle.Render(m.renderHomeView(width))
	}

	header := m.renderHeader(width)
	rule := mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120))))
	panel := panelStyle.Width(maxInt(48, width-4)).Render(m.renderPanel(width))
	footer := footerStyle.Render(m.footerHelp())
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, rule, "", panel, "", footer))
}

func (m model) footerHelp() string {
	if m.active == screenDashboard {
		return "←→ 페이지  s sync  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenDue {
		if m.duePage == 3 {
			return "←→ 페이지  ↑↓ 스크롤  s sync  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←→ 페이지  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenAcademic {
		return "←→ 월 이동  s sync  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.isDetailScreen() {
		return "↑↓ 스크롤  k KLAS  b/esc 목록  q 종료"
	}
	if m.active == screenLectures {
		return "←→ 과목  ↑↓ 스크롤  k KLAS  d 다운로드  s sync  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenConfig {
		if m.configEditing != "" {
			return "enter 저장  esc 취소  q 종료"
		}
		return "↑↓ 선택  ←→ 변경  enter 입력/실행  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.isCoursePagedScreen() {
		if m.active == screenAssignments {
			return "←→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  s sync  b/esc 뒤로  r 새로고침  q 종료"
		}
		if m.active == screenNotices {
			return "←→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←→ 과목  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenRoomResult {
		return "b/esc 뒤로  r 새로고침  q 종료"
	}
	return "b/esc 뒤로  r 새로고침  q 종료"
}

func (m model) renderHeader(width int) string {
	title := "KLAP"
	subtitle := screenTitle(m.active)
	if m.active == screenHome {
		subtitle = "Home"
	}
	left := headerStyle.Render(title) + mutedStyle.Render(" tui")
	meta := subtitle
	if !m.loadedAt.IsZero() && !m.loading && m.active != screenHome {
		meta += " · " + m.loadedAt.Format("15:04:05")
	}
	right := headerMetaStyle.Render(meta)
	spacerWidth := width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if spacerWidth < 1 {
		spacerWidth = 1
	}
	return left + strings.Repeat(" ", spacerWidth) + right
}

func (m model) renderHomeView(width int) string {
	logo := logoStyle.Render(klapLogo)
	meta := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render("https://github.com/leehyowon14/KLAP-GoLang"),
		taglineStyle.Render("Kwangwoon KLAS in your terminal."),
	)
	top := logo
	if width >= 82 {
		top = lipgloss.JoinHorizontal(lipgloss.Top, logo, "   ", meta)
	} else {
		top = lipgloss.JoinVertical(lipgloss.Left, logo, meta)
	}

	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n\n")
	b.WriteString(m.renderHomeMenu(width))
	b.WriteString("\n\n")
	b.WriteString(footerStyle.Render("↑↓ 이동  |  enter 열기  |  r 새로고침  |  q 종료"))
	return b.String()
}

func (m model) renderHomeMenu(width int) string {
	var b strings.Builder
	for index, item := range m.menu {
		selected := index == m.cursor
		prefix := fmt.Sprintf("%d.", index+1)
		marker := "  "
		if selected {
			marker = "› "
		}
		number := lipgloss.NewStyle().Width(menuNumberWidth).Render(prefix)
		title := lipgloss.NewStyle().Width(16).Render(item.title)
		help := item.help
		line := marker + number + " " + title + " " + help
		if selected {
			line = menuSelectedStyle.Render(line)
		} else {
			line = menuItemStyle.Render(line)
		}
		b.WriteString(line)
		if index < len(m.menu)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m model) renderDownloadSelectView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120)))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Download"))
	b.WriteString("\n")
	if m.loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 강의 목록을 불러오는 중입니다\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	if len(m.downloadRows) == 0 {
		b.WriteString(emptyStyle.Render("다운로드할 온라인 강의가 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	groups := m.downloadGroups()
	group := m.currentDownloadGroup()
	rows := group.rows
	page := m.downloadCourse + 1
	if page < 1 {
		page = 1
	}
	if page > len(groups) {
		page = len(groups)
	}
	b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 과목 이동  %d/%d  %s", page, len(groups), group.name)))
	b.WriteString("\n\n")

	visibleRows := maxInt(5, m.height-10)
	if m.height <= 0 {
		visibleRows = 16
	}
	totalItems := len(rows) + 1
	if visibleRows > totalItems {
		visibleRows = totalItems
	}
	start := m.downloadCursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > totalItems {
		start = maxInt(0, totalItems-visibleRows)
	}
	end := start + visibleRows
	for index := start; index < end; index++ {
		marker := "  "
		if index == m.downloadCursor {
			marker = "› "
		}
		if index == 0 {
			check := "[ ]"
			if m.currentDownloadCourseSelected() {
				check = "[x]"
			}
			line := fmt.Sprintf("%s%s  모두 선택", marker, check)
			if index == m.downloadCursor {
				line = menuSelectedStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		row := rows[index-1]
		check := "[ ]"
		if m.downloadSelected[row.ID] {
			check = "[x]"
		}
		if !lectureDownloadable(row) {
			check = "[-]"
		}
		line := fmt.Sprintf("%s%s  %s", marker, check, truncateText(lectureDownloadLabel(row), maxInt(24, minInt(70, width-16))))
		if index == m.downloadCursor {
			line = menuSelectedStyle.Render(line)
		} else if !lectureDownloadable(row) {
			line = mutedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < totalItems {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, totalItems)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("←→ 과목  |  space 선택  |  a 전체 과목  |  enter 다음  |  b 뒤로  |  q 종료"))
	return b.String()
}

func (m model) renderDownloadConfirmView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120)))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Transcript"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("선택한 강의 %d개를 다운로드한 뒤 전사할까요?", len(m.selectedDownloadIDs())))
	b.WriteString("\n\n")
	yes := mutedStyle.Render("  Yes")
	no := menuSelectedStyle.Render("› No")
	if m.downloadTranscribe {
		yes = menuSelectedStyle.Render("› Yes")
		no = mutedStyle.Render("  No")
	}
	b.WriteString(yes)
	b.WriteString("    ")
	b.WriteString(no)
	b.WriteString("\n\n")
	b.WriteString(footerStyle.Render("←→ 선택  |  y/n  |  enter 다운로드  |  b 뒤로  |  q 종료"))
	return b.String()
}

func (m model) renderDownloadLanguageView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120)))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Transcript Language"))
	b.WriteString("\n")
	b.WriteString("전사 주 언어를 선택하세요.")
	b.WriteString("\n\n")
	for index, language := range transcriptLanguages {
		marker := "  "
		if index == m.downloadLanguage {
			marker = "› "
		}
		line := fmt.Sprintf("%s%s  %s", marker, language.label, mutedStyle.Render(language.locale))
		if index == m.downloadLanguage {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("↑↓ 선택  |  enter 다운로드  |  b 뒤로  |  q 종료"))
	return b.String()
}

func (m model) renderRoomDayView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120)))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Rooms"))
	b.WriteString("\n")
	b.WriteString("빈 강의실을 조회할 요일을 선택하세요.")
	b.WriteString("\n\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	for index, option := range roomDayOptions {
		marker := "  "
		if index == m.roomDayCursor {
			marker = "› "
		}
		check := "[ ]"
		if m.roomDaysSelected[option.weekday] {
			check = "[x]"
		}
		line := fmt.Sprintf("%s%s  %s", marker, check, option.label)
		if index == m.roomDayCursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("↑↓ 이동  |  space 선택  |  a 전체  |  enter 다음  |  b 뒤로  |  q 종료"))
	return b.String()
}

func (m model) renderRoomPeriodView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120)))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Rooms"))
	b.WriteString("\n")
	b.WriteString("빈 강의실을 조회할 교시를 선택하세요.")
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("요일 " + roomSelectedDaysLabel(m.selectedRoomDays())))
	b.WriteString("\n\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	for period := 1; period <= 8; period++ {
		index := period - 1
		marker := "  "
		if index == m.roomPeriodCursor {
			marker = "› "
		}
		check := "[ ]"
		if m.roomPeriodsSelected[period] {
			check = "[x]"
		}
		line := fmt.Sprintf("%s%s  %d교시", marker, check, period)
		if index == m.roomPeriodCursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("↑↓ 이동  |  space 선택  |  a 전체  |  enter 조회  |  b 뒤로  |  q 종료"))
	return b.String()
}

func (m model) renderPanel(width int) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(screenTitle(m.active)))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(screenSubtitle(m.active)))
	b.WriteString("\n\n")
	if m.loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 데이터를 불러오는 중입니다\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n")
		return b.String()
	}
	if m.active == screenDashboard {
		b.WriteString(m.renderDashboardPagedPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if m.active == screenDue {
		b.WriteString(m.renderDuePagedPanel(width))
		return b.String()
	}
	if m.isDetailScreen() {
		b.WriteString(m.renderDetailPanel(width))
		return b.String()
	}
	if m.active == screenAcademic {
		b.WriteString(m.renderAcademicCalendarPanel(width))
		return b.String()
	}
	if m.active == screenConfig {
		b.WriteString(m.renderConfigPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if m.isCoursePagedScreen() {
		b.WriteString(m.renderCoursePagedPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if strings.TrimSpace(m.content) == "" {
		b.WriteString(emptyStyle.Render("표시할 내용이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(m.content)
	if !strings.HasSuffix(m.content, "\n") {
		b.WriteString("\n")
	}
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) renderConfigPanel(width int) string {
	rows := configRows(m.configSettings, m.configOptions)
	if len(rows) == 0 {
		return emptyStyle.Render("설정 항목이 없습니다") + "\n"
	}
	valueWidth := maxInt(16, minInt(42, width-44))
	var b strings.Builder
	lastSection := ""
	for index, row := range rows {
		if row.section != lastSection {
			if index > 0 {
				b.WriteString("\n")
			}
			b.WriteString(mutedStyle.Render(row.section))
			b.WriteString("\n")
			lastSection = row.section
		}
		marker := "  "
		if index == m.configCursor {
			marker = "› "
		}
		label := lipgloss.NewStyle().Width(18).Render(row.label)
		value := truncateText(row.value, valueWidth)
		valueText := lipgloss.NewStyle().Width(valueWidth).Render(value)
		hint := mutedStyle.Render(row.hint)
		line := fmt.Sprintf("%s%s  %s  %s", marker, label, valueText, hint)
		if index == m.configCursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.configEditing != "" {
		b.WriteString("\n")
		b.WriteString(m.configInput.View())
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("enter 저장  esc 취소"))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) renderCoursePagedPanel(width int) string {
	groups := m.contentGroups(width)
	if len(groups) == 0 {
		switch m.active {
		case screenAssignments:
			return emptyStyle.Render("과제가 없습니다") + "\n"
		case screenNotices:
			return emptyStyle.Render("강의 공지가 없습니다") + "\n"
		case screenLectures:
			return emptyStyle.Render("온라인 강의가 없습니다") + "\n"
		default:
			return emptyStyle.Render("표시할 내용이 없습니다") + "\n"
		}
	}

	group := m.currentContentGroup(width)
	page := m.contentCourse + 1
	if page < 1 {
		page = 1
	}
	if page > len(groups) {
		page = len(groups)
	}

	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 과목 이동  %d/%d  %s", page, len(groups), group.name)))
	b.WriteString("\n\n")

	visibleRows := maxInt(5, m.height-10)
	if m.height <= 0 {
		visibleRows = 16
	}
	if visibleRows > len(group.lines) {
		visibleRows = len(group.lines)
	}

	cursor := m.contentCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(group.lines) {
		cursor = len(group.lines) - 1
	}

	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(group.lines) {
		start = maxInt(0, len(group.lines)-visibleRows)
	}
	end := start + visibleRows

	for index := start; index < end; index++ {
		marker := "  "
		if index == cursor {
			marker = "› "
		}
		line := marker + group.lines[index]
		if index == cursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < len(group.lines) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(group.lines))))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) renderDashboardPagedPanel(width int) string {
	pages := dashboardPageCount(m.dashboardResult)
	if pages <= 1 || m.dashboardPage == 0 {
		var b strings.Builder
		b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 페이지 이동  1/%d  전체", maxInt(1, pages))))
		b.WriteString("\n\n")
		b.WriteString(formatDashboard(m.dashboardResult))
		return b.String()
	}
	courses := m.dashboardResult.Courses
	index := m.dashboardPage - 1
	if index < 0 {
		index = 0
	}
	if index >= len(courses) {
		index = len(courses) - 1
	}
	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 페이지 이동  %d/%d  %s", m.dashboardPage+1, pages, courses[index].Name)))
	b.WriteString("\n\n")
	b.WriteString(formatDashboardCourse(m.dashboardResult, courses[index]))
	return b.String()
}

func (m model) renderDetailPanel(width int) string {
	lines := m.detailLines(width)
	if len(lines) == 0 {
		return emptyStyle.Render("상세 정보가 없습니다") + "\n"
	}
	visibleRows := maxInt(5, m.height-10)
	if m.height <= 0 {
		visibleRows = 18
	}
	if visibleRows > len(lines) {
		visibleRows = len(lines)
	}
	cursor := m.detailCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(lines) {
		cursor = len(lines) - 1
	}
	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(lines) {
		start = maxInt(0, len(lines)-visibleRows)
	}
	end := start + visibleRows

	var b strings.Builder
	for index := start; index < end; index++ {
		b.WriteString(lines[index])
		b.WriteString("\n")
	}
	if start > 0 || end < len(lines) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(lines))))
		b.WriteString("\n")
	}
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) detailLines(width int) []string {
	switch m.active {
	case screenAssignmentDetail:
		return assignmentDetailLines(m.assignmentDetail, width)
	case screenNoticeDetail:
		return noticeDetailLines(m.noticeDetail, width)
	default:
		return nil
	}
}

func assignmentDetailLines(result app.AssignmentDetailResult, width int) []string {
	if strings.TrimSpace(result.ID) == "" {
		return nil
	}
	detail := result.Detail
	status := "미제출"
	if detail.Submitted {
		status = "제출"
	}
	lines := []string{
		sectionStyle.Render(detail.Title),
		mutedStyle.Render(result.CourseName),
		"",
		"마감  " + formatTime(detail.DueAt),
		"상태  " + status,
	}
	if detail.ReportType != "" {
		lines = append(lines, "제출 방식  "+detail.ReportType)
	}
	if detail.SubmitFileType != "" {
		lines = append(lines, "파일 형식  "+detail.SubmitFileType)
	}
	if detail.FileLimitMB != "" {
		lines = append(lines, "파일 제한  "+detail.FileLimitMB+"MB")
	}
	lines = append(lines, "", mutedStyle.Render("KLAS  "+result.DetailURL))
	if strings.TrimSpace(detail.ContentText) != "" {
		lines = append(lines, "", sectionStyle.Render("본문"))
		lines = appendWrappedLines(lines, detail.ContentText, width)
	}
	if strings.TrimSpace(detail.SubmittedTitle+detail.SubmittedText) != "" {
		lines = append(lines, "", sectionStyle.Render("내 제출"))
		if detail.SubmittedTitle != "" {
			lines = append(lines, "제목  "+detail.SubmittedTitle)
		}
		lines = appendWrappedLines(lines, detail.SubmittedText, width)
	}
	if detail.FinalScore != "" && detail.FinalScore != "<nil>" {
		lines = append(lines, "", "점수  "+detail.FinalScore)
	}
	if strings.TrimSpace(detail.TutorText) != "" {
		lines = append(lines, "", sectionStyle.Render("피드백"))
		lines = appendWrappedLines(lines, detail.TutorText, width)
	}
	return lines
}

func noticeDetailLines(result app.NoticeDetailResult, width int) []string {
	if strings.TrimSpace(result.ID) == "" {
		return nil
	}
	detail := result.Detail
	lines := []string{
		sectionStyle.Render(detail.Title),
		mutedStyle.Render(result.CourseName),
		"",
		"작성일  " + formatTime(detail.Registered),
	}
	if detail.Author != "" {
		lines = append(lines, "작성자  "+detail.Author)
	}
	if detail.Top {
		lines = append(lines, "중요  예")
	}
	if detail.ReadCount != "" {
		lines = append(lines, "조회수  "+detail.ReadCount)
	}
	if detail.Attachment != "" {
		lines = append(lines, "첨부 묶음  "+detail.Attachment)
	}
	lines = append(lines, "", mutedStyle.Render("KLAS  "+result.DetailURL))
	if strings.TrimSpace(detail.ContentText) != "" {
		lines = append(lines, "", sectionStyle.Render("본문"))
		lines = appendWrappedLines(lines, detail.ContentText, width)
	}
	return lines
}

func appendWrappedLines(lines []string, text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return lines
	}
	wrapWidth := maxInt(24, minInt(100, width-6))
	for _, paragraph := range strings.Split(text, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		rendered := lipgloss.NewStyle().Width(wrapWidth).Render(paragraph)
		lines = append(lines, strings.Split(rendered, "\n")...)
	}
	return lines
}

func (m model) renderDuePagedPanel(width int) string {
	lines := duePageLines(m.dueResult, m.duePage, width)
	page := m.duePage + 1
	if page < 1 {
		page = 1
	}
	if page > len(duePageLabels) {
		page = len(duePageLabels)
	}

	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%s ~ %s", m.dueResult.From.Format("2006-01-02"), m.dueResult.Until.Format("2006-01-02"))))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 페이지 이동  %d/%d  %s", page, len(duePageLabels), duePageLabels[page-1])))
	b.WriteString("\n\n")
	if len(lines) == 0 {
		b.WriteString(emptyStyle.Render("표시할 일정이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}

	visibleRows := maxInt(5, m.height-12)
	if m.height <= 0 {
		visibleRows = 16
	}
	if visibleRows > len(lines) {
		visibleRows = len(lines)
	}
	cursor := m.dueCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(lines) {
		cursor = len(lines) - 1
	}
	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(lines) {
		start = maxInt(0, len(lines)-visibleRows)
	}
	end := start + visibleRows
	for index := start; index < end; index++ {
		marker := "  "
		if index == cursor {
			marker = "› "
		}
		line := marker + lines[index]
		if index == cursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < len(lines) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(lines))))
		b.WriteString("\n")
	}
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func duePageLines(result app.DueResult, page int, width int) []string {
	if page == 0 {
		return dueSummaryLines(result)
	}
	if page < 0 || page >= len(duePageLabels) {
		return nil
	}
	kind := duePageLabels[page]
	lines := make([]string, 0)
	for _, item := range result.Items {
		if item.Kind != kind {
			continue
		}
		course := item.CourseName
		if course == "" {
			course = "-"
		}
		titleWidth := maxInt(12, minInt(64, width-38))
		lines = append(lines, fmt.Sprintf("%s  %s  %s  %s",
			mutedStyle.Render(item.DueAt.Format("2006-01-02 15:04")),
			badgeStyle.Render(item.Kind),
			truncateText(course, 18),
			truncateText(item.Title, titleWidth),
		))
	}
	if page >= 0 && page < len(duePageLabels) {
		for _, sectionError := range result.Errors {
			if sectionError.Section == duePageLabels[page] {
				lines = append(lines, errorStyle.Render("ERROR")+" "+sectionError.Err.Error())
			}
		}
	}
	return lines
}

func dueSummaryLines(result app.DueResult) []string {
	counts := map[string]int{}
	for _, item := range result.Items {
		counts[item.Kind]++
	}
	lines := []string{
		fmt.Sprintf("%s  %d", badgeStyle.Render("전체"), len(result.Items)),
		fmt.Sprintf("%s  %d", badgeStyle.Render("과제"), counts["과제"]),
		fmt.Sprintf("%s  %d", badgeStyle.Render("온라인 강의"), counts["온라인 강의"]),
		fmt.Sprintf("%s  %d", badgeStyle.Render("학사일정"), counts["학사일정"]),
	}
	if len(result.Errors) > 0 {
		lines = append(lines, "")
		for _, sectionError := range result.Errors {
			lines = append(lines, errorStyle.Render(sectionError.Section)+" "+sectionError.Err.Error())
		}
	}
	return lines
}

func (m model) renderAcademicCalendarPanel(width int) string {
	month := m.academicMonth
	if month < 1 || month > 12 {
		month = defaultAcademicMonth(m.academicResult.Events, time.Now())
	}
	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 월 이동  %s년 %d월", m.academicResult.Year, month)))
	b.WriteString("\n\n")
	b.WriteString(renderAcademicMonthCalendar(m.academicResult, month, width))
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func renderAcademicMonthCalendar(result app.AcademicListResult, month int, width int) string {
	year, err := strconv.Atoi(strings.TrimSpace(result.Year))
	if err != nil || month < 1 || month > 12 {
		return emptyStyle.Render("학사일정 연도를 확인할 수 없습니다") + "\n"
	}
	eventsByDay := make(map[int][]app.AcademicEvent)
	for _, event := range result.Events {
		dueAt, ok := academicEventDate(event)
		if !ok || dueAt.Year() != year || int(dueAt.Month()) != month {
			continue
		}
		eventsByDay[dueAt.Day()] = append(eventsByDay[dueAt.Day()], event)
	}

	var b strings.Builder
	b.WriteString("월  화  수  목  금  토  일\n")
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	offset := int(first.Weekday()+6) % 7
	days := daysInMonth(year, month)
	for index := 0; index < offset; index++ {
		b.WriteString("    ")
	}
	for day := 1; day <= days; day++ {
		label := fmt.Sprintf("%2d", day)
		if len(eventsByDay[day]) > 0 {
			label = warnTextStyle.Render(label)
		}
		b.WriteString(label)
		if (offset+day)%7 == 0 {
			b.WriteString("\n")
		} else {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n\n")
	eventLines := make([]string, 0)
	for day := 1; day <= days; day++ {
		for _, event := range eventsByDay[day] {
			eventLines = append(eventLines, fmt.Sprintf("%02d일  %s", day, truncateText(event.Title, maxInt(16, minInt(72, width-12)))))
		}
	}
	if len(eventLines) == 0 {
		b.WriteString(emptyStyle.Render("이 달의 학사일정이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	for _, line := range eventLines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func defaultAcademicMonth(events []app.AcademicEvent, now time.Time) int {
	if len(events) == 0 {
		return int(now.Month())
	}
	for _, event := range events {
		dueAt, ok := academicEventDate(event)
		if ok && !dueAt.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)) {
			return int(dueAt.Month())
		}
	}
	if dueAt, ok := academicEventDate(events[0]); ok {
		return int(dueAt.Month())
	}
	return int(now.Month())
}

var tuiAcademicDatePattern = regexp.MustCompile(`([0-9]{1,2})\s*[./]\s*([0-9]{1,2})|([0-9]{1,2})\s*일|([0-9]{1,2})\s*\(`)

func academicEventDate(event app.AcademicEvent) (time.Time, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(event.Year))
	if err != nil {
		return time.Time{}, false
	}
	month, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(event.Month), "월"))
	if err != nil {
		return time.Time{}, false
	}
	match := tuiAcademicDatePattern.FindStringSubmatch(strings.TrimSpace(event.Date))
	if len(match) == 0 {
		return time.Time{}, false
	}
	dayText := firstNonEmptyString(match[2], match[3], match[4])
	if match[1] != "" && match[2] != "" {
		if parsedMonth, monthErr := strconv.Atoi(match[1]); monthErr == nil {
			month = parsedMonth
		}
	}
	day, err := strconv.Atoi(dayText)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), true
}

func daysInMonth(year int, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func (m model) load(target screen, refresh bool) tea.Cmd {
	return func() tea.Msg {
		switch target {
		case screenDashboard:
			result, err := m.service.Dashboard(m.ctx, app.DashboardOptions{Refresh: refresh})
			return loadMsg{screen: target, content: formatDashboard(result), dashboard: result, err: err}
		case screenDue:
			result, err := m.service.Due(m.ctx, app.DueOptions{Days: 14, Refresh: refresh})
			return loadMsg{screen: target, due: result, err: err}
		case screenAssignments:
			rows, err := m.service.AssignmentList(m.ctx, app.AssignmentListOptions{Refresh: refresh})
			return loadMsg{screen: target, assignments: rows, err: err}
		case screenNotices:
			rows, err := m.service.NoticeList(m.ctx, app.NoticeListOptions{Refresh: refresh})
			return loadMsg{screen: target, notices: rows, err: err}
		case screenLectures:
			rows, err := m.service.LectureList(m.ctx, app.LectureListOptions{Refresh: refresh})
			return loadMsg{screen: target, lectures: rows, err: err}
		case screenAcademic:
			result, err := m.service.AcademicList(m.ctx, app.AcademicListOptions{Refresh: refresh})
			return loadMsg{screen: target, academic: result, err: err}
		case screenConfig:
			settings, err := m.service.ConfigSettings()
			if err != nil {
				return loadMsg{screen: target, err: err}
			}
			categories, err := m.service.CategoryOptions()
			return loadMsg{screen: target, content: formatConfig(settings), config: settings, categories: categories, err: err}
		}
		content, err := m.loadContent(target, refresh)
		return loadMsg{screen: target, content: content, err: err}
	}
}

func (m model) loadContent(target screen, refresh bool) (string, error) {
	switch target {
	case screenDashboard:
		return "", errors.New("Dashboard는 structured loader를 사용해야 합니다")
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
	case screenAssignmentDetail:
		return "Assignment"
	case screenNoticeDetail:
		return "Notice"
	case screenAcademic:
		return "Academic"
	case screenConfig:
		return "Config"
	case screenRoomDay, screenRoomPeriod, screenRoomResult:
		return "Rooms"
	case screenDownloadSelect:
		return "Download"
	case screenDownloadConfirm:
		return "Transcript"
	case screenDownloadLanguage:
		return "Transcript"
	case screenDownloadProgress:
		return "Download"
	default:
		return "Home"
	}
}

func screenSubtitle(value screen) string {
	switch value {
	case screenDashboard:
		return "과제, 온라인 강의, 공지, 출석, 수업평가 요약"
	case screenDue:
		return "다가오는 과제, 온라인 강의, 학사일정"
	case screenAssignments:
		return "현재 학기 과제 목록"
	case screenNotices:
		return "최근 강의 공지"
	case screenLectures:
		return "온라인 강의와 학습활동 상태"
	case screenAssignmentDetail:
		return "과제 상세"
	case screenNoticeDetail:
		return "공지 상세"
	case screenAcademic:
		return "학사일정 달력"
	case screenConfig:
		return "현재 유저 설정"
	case screenRoomResult:
		return "선택한 요일과 교시에 비어 있는 강의실"
	default:
		return ""
	}
}

func formatDashboard(result app.DashboardResult) string {
	var b strings.Builder
	b.WriteString(formatDashboardOverview(result))
	if result.Cached {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("캐시 사용 " + result.CacheCreatedAt.Format("2006-01-02 15:04")))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderSection("FOCUS", formatDashboardFocus(result)))
	b.WriteString("\n")
	b.WriteString(renderSection("LATEST", formatNoticeSummary(result.Notices)))
	if len(result.SectionErrors) > 0 {
		b.WriteString("\n")
		b.WriteString(formatSectionErrors(result.SectionErrors))
	}
	return b.String()
}

func dashboardPageCount(result app.DashboardResult) int {
	return 1 + len(result.Courses)
}

func formatDashboardOverview(result app.DashboardResult) string {
	lines := []string{
		fmt.Sprintf("%s  %s", badgeStyle.Render(result.Term.Value), result.Term.Label),
		dashboardMetric("Due", fmt.Sprintf("%d assignments · %d lectures", len(result.Assignments), len(result.Lectures))),
		dashboardMetric("Activity", fmt.Sprintf("%d notices", len(result.Notices))),
	}
	if result.Attendance.TotalCourses > 0 {
		total := result.Attendance.Completed + result.Attendance.Absent + result.Attendance.Late + result.Attendance.LeaveEarly + result.Attendance.Excused + result.Attendance.Unknown
		lines = append(lines, dashboardMetric("Attendance", fmt.Sprintf("출석 %d/%d · 결석 %d · 지각 %d", result.Attendance.Completed, total, result.Attendance.Absent, result.Attendance.Late)))
	}
	if result.Evaluation.Enabled {
		lines = append(lines, dashboardMetric("Evaluation", fmt.Sprintf("완료 %d · 미완료 %d", result.Evaluation.Done, result.Evaluation.Pending)))
	}
	return renderSection("OVERVIEW", lines)
}

func formatDashboardCourse(result app.DashboardResult, course app.DashboardCourse) string {
	var b strings.Builder
	b.WriteString(formatDashboardCourseOverview(result, course))
	b.WriteString("\n")
	b.WriteString(renderSection("FOCUS", formatDashboardCourseFocus(course)))
	b.WriteString("\n")
	b.WriteString(renderSection("LATEST", formatNoticeSummary(course.Notices)))
	status := formatDashboardCourseStatus(course)
	if len(status) > 0 {
		b.WriteString("\n")
		b.WriteString(renderSection("STATUS", status))
	}
	return b.String()
}

func formatDashboardCourseOverview(result app.DashboardResult, course app.DashboardCourse) string {
	lines := []string{
		fmt.Sprintf("%s  %s", badgeStyle.Render(result.Term.Value), result.Term.Label),
		dashboardMetric("Course", fmt.Sprintf("%d. %s", course.Index, course.Name)),
		dashboardMetric("Due", fmt.Sprintf("%d assignments · %d lectures", len(course.Assignments), len(course.Lectures))),
		dashboardMetric("Activity", fmt.Sprintf("%d notices", len(course.Notices))),
	}
	if course.Attendance != nil {
		total := course.Attendance.Completed + course.Attendance.Absent + course.Attendance.Late + course.Attendance.LeaveEarly + course.Attendance.Excused + course.Attendance.Unknown
		lines = append(lines, dashboardMetric("Attendance", fmt.Sprintf("출석 %d/%d · 결석 %d · 지각 %d", course.Attendance.Completed, total, course.Attendance.Absent, course.Attendance.Late)))
	}
	if result.Evaluation.Enabled {
		evaluationStatus := "완료"
		if course.Evaluation != nil {
			evaluationStatus = "미완료"
		}
		lines = append(lines, dashboardMetric("Evaluation", evaluationStatus))
	}
	return renderSection("OVERVIEW", lines)
}

func dashboardMetric(label string, value string) string {
	return mutedStyle.Render(lipgloss.NewStyle().Width(11).Render(label)) + " " + value
}

func formatDashboardFocus(result app.DashboardResult) []string {
	lines := make([]string, 0, len(result.Assignments)+len(result.Lectures))
	for _, row := range result.Assignments {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			warnBadgeStyle.Render("과제"),
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			row.CourseName+" · "+row.Assignment.Title,
		))
	}
	for _, row := range result.Lectures {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			badgeStyle.Render("강의"),
			mutedStyle.Render(formatTime(row.Lecture.EndAt)),
			row.CourseName+" · "+row.Lecture.Title,
		))
	}
	if len(lines) == 0 {
		return []string{emptyStyle.Render("처리할 항목이 없습니다")}
	}
	return lines
}

func formatDashboardCourseFocus(course app.DashboardCourse) []string {
	lines := make([]string, 0, len(course.Assignments)+len(course.Lectures))
	for _, row := range course.Assignments {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			warnBadgeStyle.Render("과제"),
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			row.Assignment.Title,
		))
	}
	for _, row := range course.Lectures {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			badgeStyle.Render("강의"),
			mutedStyle.Render(formatTime(row.Lecture.EndAt)),
			row.Lecture.Title,
		))
	}
	if len(lines) == 0 {
		return []string{emptyStyle.Render("처리할 항목이 없습니다")}
	}
	return lines
}

func formatDashboardCourseStatus(course app.DashboardCourse) []string {
	lines := make([]string, 0, 3)
	if course.Attendance != nil {
		if course.Attendance.Err != nil {
			lines = append(lines, warnTextStyle.Render("출석 상세를 불러오지 못했습니다"))
		} else {
			lines = append(lines, fmt.Sprintf("출석  %s · %s · %s",
				successTextStyle.Render(fmt.Sprintf("O %d", course.Attendance.Completed)),
				warnTextStyle.Render(fmt.Sprintf("X %d", course.Attendance.Absent)),
				warnTextStyle.Render(fmt.Sprintf("L %d", course.Attendance.Late)),
			))
		}
	}
	if course.Evaluation != nil {
		lines = append(lines, warnTextStyle.Render("수업평가 미완료"))
	}
	if len(lines) == 0 {
		return nil
	}
	return lines
}

func formatAssignmentSummary(rows []app.AssignmentRow) []string {
	if len(rows) == 0 {
		return []string{emptyStyle.Render("예정된 과제가 없습니다")}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", mutedStyle.Render(formatTime(row.Assignment.DueAt)), row.CourseName, row.Assignment.Title))
	}
	return lines
}

func formatLectureSummary(rows []app.LectureRow) []string {
	if len(rows) == 0 {
		return []string{emptyStyle.Render("수강할 온라인 강의가 없습니다")}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", mutedStyle.Render(formatTime(row.Lecture.EndAt)), row.CourseName, row.Lecture.Title))
	}
	return lines
}

func formatNoticeSummary(rows []app.NoticeRow) []string {
	if len(rows) == 0 {
		return []string{emptyStyle.Render("공지 없음")}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			mutedStyle.Render(formatTime(row.Notice.Registered)),
			truncateText(row.CourseName, 22),
			truncateText(row.Notice.Title, 30),
		))
	}
	return lines
}

func formatDue(result app.DueResult) string {
	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%s ~ %s", result.From.Format("2006-01-02"), result.Until.Format("2006-01-02"))))
	b.WriteString("\n\n")
	if len(result.Items) == 0 {
		b.WriteString(emptyStyle.Render("예정된 데드라인이 없습니다"))
		b.WriteString("\n")
	}
	for _, item := range result.Items {
		course := item.CourseName
		if course == "" {
			course = "-"
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			mutedStyle.Render(item.DueAt.Format("2006-01-02 15:04")),
			badgeStyle.Render(item.Kind),
			course,
			item.Title,
		))
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
		return emptyStyle.Render("과제가 없습니다") + "\n"
	}
	var b strings.Builder
	for _, row := range rows {
		status := warnBadgeStyle.Render("미제출")
		if row.Assignment.Submitted {
			status = successBadgeStyle.Render("제출")
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			mutedStyle.Render(formatTime(row.Assignment.DueAt)),
			status,
			row.CourseName,
			row.Assignment.Title,
		))
	}
	return b.String()
}

func formatNotices(rows []app.NoticeRow) string {
	if len(rows) == 0 {
		return emptyStyle.Render("강의 공지가 없습니다") + "\n"
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("%s  %s  %s\n",
			mutedStyle.Render(formatTime(row.Notice.Registered)),
			row.CourseName,
			row.Notice.Title,
		))
	}
	return b.String()
}

func formatLectures(rows []app.LectureRow) string {
	if len(rows) == 0 {
		return emptyStyle.Render("온라인 강의가 없습니다") + "\n"
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			mutedStyle.Render(lectureProgress(row.Lecture)),
			row.CourseName,
			emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요"),
			row.Lecture.Title,
		))
	}
	return b.String()
}

func configRows(settings app.ConfigSettings, options app.CategoryOptions) []configRow {
	return []configRow{
		{
			key:      "reminder.name",
			section:  "Reminder",
			label:    "미리알림 목록",
			value:    emptyFallback(settings.Reminder.ListName, settingspkg.DefaultReminderListName),
			hint:     categoryHint(options.Reminders, settings.Reminder.UseExistingList),
			editable: true,
			cycle:    true,
		},
		{
			key:      "reminder.alarm-before-min",
			section:  "Reminder",
			label:    "알림 시간",
			value:    fmt.Sprintf("마감 %s 전", formatMinutes(settings.Reminder.AlarmBeforeMin)),
			hint:     "←→ 60분 단위  enter 직접 입력",
			editable: true,
		},
		{
			key:      "calendar.name",
			section:  "Calendar",
			label:    "캘린더",
			value:    emptyFallback(settings.Calendar.Name, settingspkg.DefaultReminderListName),
			hint:     categoryHint(options.Calendars, settings.Calendar.UseExistingList),
			editable: true,
			cycle:    true,
		},
		{
			key:      "download.dir",
			section:  "Download",
			label:    "저장 폴더",
			value:    settings.Download.Dir,
			hint:     "enter 직접 입력",
			editable: true,
		},
		{
			key:     "download.concurrency",
			section: "Download",
			label:   "동시 다운로드",
			value:   fmt.Sprintf("%d workers", settings.Download.Concurrency),
			hint:    "←→ 변경",
		},
		{
			key:     "download.caffeinate",
			section: "Download",
			label:   "절전 방지",
			value:   enabledLabel(settings.Download.Caffeinate),
			hint:    "←→ 전환",
		},
		{
			key:     "download.keep-partial",
			section: "Download",
			label:   "부분 파일 보존",
			value:   enabledLabel(settings.Download.KeepPartial),
			hint:    "←→ 전환",
		},
		{
			key:     "transcript.concurrency",
			section: "Transcript",
			label:   "전사 worker",
			value:   fmt.Sprintf("%d workers", settings.Transcript.Concurrency),
			hint:    fmt.Sprintf("←→ 1-%d", settingspkg.MaxTranscriptConcurrency),
		},
		{
			key:     "reset",
			section: "General",
			label:   "설정 초기화",
			value:   "기본값으로 복원",
			hint:    "enter 실행",
			reset:   true,
		},
	}
}

func categoryHint(options []string, useExisting bool) string {
	if len(options) == 0 {
		return "enter 직접 입력"
	}
	if useExisting {
		return "←→ 기존 목록  enter 직접 입력"
	}
	return "←→ 기존 목록  enter 새 카테고리"
}

func enabledLabel(value bool) string {
	if value {
		return "켜짐"
	}
	return "꺼짐"
}

func formatMinutes(minutes int) string {
	if minutes%1440 == 0 {
		days := minutes / 1440
		if days == 1 {
			return "1일"
		}
		return fmt.Sprintf("%d일", days)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%d시간", minutes/60)
	}
	return fmt.Sprintf("%d분", minutes)
}

func cycleStringOption(options []string, current string, delta int) (string, bool) {
	values := uniqueStrings(append([]string{current}, options...))
	if len(values) < 2 {
		return "", false
	}
	index := 0
	for i, value := range values {
		if value == current {
			index = i
			break
		}
	}
	next := (index + delta + len(values)) % len(values)
	return values[next], true
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func formatConfig(settings app.ConfigSettings) string {
	term := strings.TrimSpace(settings.Term.Value)
	if term == "" {
		term = "자동"
	}
	return renderSection("General", []string{
		"term  " + term,
	}) + "\n" + renderSection("Reminder", []string{
		"name  " + settings.Reminder.ListName,
		fmt.Sprintf("use-existing-list  %t", settings.Reminder.UseExistingList),
		fmt.Sprintf("alarm-before-min  %d", settings.Reminder.AlarmBeforeMin),
	}) + "\n" + renderSection("Calendar", []string{
		"name  " + settings.Calendar.Name,
		fmt.Sprintf("use-existing-list  %t", settings.Calendar.UseExistingList),
	}) + "\n" + renderSection("Download", []string{
		"dir  " + settings.Download.Dir,
		fmt.Sprintf("concurrency  %d", settings.Download.Concurrency),
		fmt.Sprintf("caffeinate  %t", settings.Download.Caffeinate),
		fmt.Sprintf("keep-partial  %t", settings.Download.KeepPartial),
	}) + "\n" + renderSection("Transcript", []string{
		fmt.Sprintf("concurrency  %d", settings.Transcript.Concurrency),
	})
}

func formatRoomAvailableResults(results []app.RoomAvailableResult) string {
	if len(results) == 0 {
		return emptyStyle.Render("조건에 맞는 빈 강의실이 없습니다")
	}
	var b strings.Builder
	warnings := make(map[string]struct{})
	for resultIndex, result := range results {
		if resultIndex > 0 {
			b.WriteString("\n")
		}
		status := roomAvailableStatus(result.Weekday, result.Periods)
		b.WriteString(sectionStyle.Render(status))
		if result.Cached {
			b.WriteString(" ")
			b.WriteString(mutedStyle.Render("cache"))
		}
		b.WriteString("\n")
		if len(result.Rooms) == 0 {
			b.WriteString(emptyStyle.Render("조건에 맞는 빈 강의실이 없습니다"))
			b.WriteString("\n")
		} else {
			for index, room := range result.Rooms {
				b.WriteString(fmt.Sprintf("%d. %s  %s\n", index+1, room.Room, mutedStyle.Render(status)))
			}
		}
		for _, warning := range result.Warnings {
			warnings[warning] = struct{}{}
		}
	}
	if len(warnings) > 0 {
		keys := make([]string, 0, len(warnings))
		for warning := range warnings {
			keys = append(keys, warning)
		}
		sort.Strings(keys)
		b.WriteString("\n")
		b.WriteString(warnBadgeStyle.Render("WARN"))
		b.WriteString(fmt.Sprintf(" %d개 과목의 강의시간 조회 실패\n", len(keys)))
		limit := len(keys)
		if limit > 10 {
			limit = 10
		}
		for _, warning := range keys[:limit] {
			b.WriteString("- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
		if len(keys) > limit {
			b.WriteString(fmt.Sprintf("- ... %d개 생략\n", len(keys)-limit))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func roomAvailableStatus(weekday int, periods []int) string {
	return app.RoomWeekdayLabel(weekday) + " " + roomPeriodsLabel(periods) + " 비어있음"
}

func roomPeriodsLabel(periods []int) string {
	normalized := normalizeRoomPeriodsForView(periods)
	if len(normalized) == 0 {
		return "교시 미지정"
	}
	contiguous := true
	for index := 1; index < len(normalized); index++ {
		if normalized[index] != normalized[index-1]+1 {
			contiguous = false
			break
		}
	}
	if contiguous {
		if normalized[0] == normalized[len(normalized)-1] {
			return fmt.Sprintf("%d교시", normalized[0])
		}
		return fmt.Sprintf("%d-%d교시", normalized[0], normalized[len(normalized)-1])
	}
	labels := make([]string, 0, len(normalized))
	for _, period := range normalized {
		labels = append(labels, strconv.Itoa(period))
	}
	return strings.Join(labels, ", ") + "교시"
}

func normalizeRoomPeriodsForView(periods []int) []int {
	normalized := make([]int, 0, len(periods))
	seen := make(map[int]struct{}, len(periods))
	for _, period := range periods {
		if period <= 0 {
			continue
		}
		if _, ok := seen[period]; ok {
			continue
		}
		seen[period] = struct{}{}
		normalized = append(normalized, period)
	}
	sort.Ints(normalized)
	return normalized
}

func roomSelectedDaysLabel(days []int) string {
	if len(days) == 0 {
		return "선택 없음"
	}
	labels := make([]string, 0, len(days))
	for _, weekday := range days {
		labels = append(labels, app.RoomWeekdayLabel(weekday))
	}
	return strings.Join(labels, ", ")
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

func renderSection(title string, lines []string) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(title))
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func formatSectionErrors(errors []app.DashboardSectionError) string {
	lines := make([]string, 0, len(errors))
	for _, sectionError := range errors {
		lines = append(lines, errorStyle.Render(sectionError.Section)+" "+sectionError.Err.Error())
	}
	return renderSection("확인 실패", lines)
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= limit {
		return value
	}
	if limit <= 1 {
		return "…"
	}
	var b strings.Builder
	for _, r := range value {
		next := b.String() + string(r)
		if lipgloss.Width(next+"…") > limit {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "…"
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
