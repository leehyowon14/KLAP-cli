package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kw-klap/klap-cli/internal/app"
	"github.com/kw-klap/klap-cli/internal/klas"
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
	screenConfig
	screenDownloadSelect
	screenDownloadConfirm
	screenDownloadProgress
)

type menuItem struct {
	title  string
	help   string
	screen screen
}

type model struct {
	ctx                context.Context
	service            *app.Service
	menu               []menuItem
	cursor             int
	active             screen
	loading            bool
	err                error
	content            string
	width              int
	height             int
	loadedAt           time.Time
	configEditing      string
	configInput        textinput.Model
	downloadRows       []app.LectureRow
	downloadSelected   map[string]bool
	downloadCourse     int
	downloadCursor     int
	downloadTranscribe bool
	downloadProgress   *lectureDownloadModel
}

type loadMsg struct {
	screen  screen
	content string
	err     error
}

type downloadRowsMsg struct {
	rows []app.LectureRow
	err  error
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
			if m.active != screenHome {
				m.active = screenHome
				m.err = nil
				m.content = ""
				m.loading = false
			}
		case key == "up" || keyMatches(key, "k", "ㅏ"):
			if m.active == screenHome && m.cursor > 0 {
				m.cursor--
			}
		case key == "down" || keyMatches(key, "j", "ㅓ"):
			if m.active == screenHome && m.cursor < len(m.menu)-1 {
				m.cursor++
			}
		case key == "enter":
			if m.active == screenHome && len(m.menu) > 0 {
				target := m.menu[m.cursor].screen
				m.active = target
				m.loading = true
				m.err = nil
				m.content = ""
				return m, m.load(target, false)
			}
		case keyMatches(key, "r", "ㄱ"):
			if m.active != screenHome {
				m.loading = true
				m.err = nil
				m.content = ""
				return m, m.load(m.active, true)
			}
		case m.active == screenConfig && keyMatches(key, "d", "ㅇ"):
			return m.startDownloadDirEdit()
		case m.active == screenConfig && (key == "+" || key == "="):
			return m.adjustDownloadConcurrency(1)
		case m.active == screenConfig && key == "-":
			return m.adjustDownloadConcurrency(-1)
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

func (m model) loadDownloadRows() tea.Cmd {
	return func() tea.Msg {
		rows, err := m.service.LectureList(m.ctx, app.LectureListOptions{Refresh: true})
		return downloadRowsMsg{rows: rows, err: err}
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
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if m.downloadCursor > 0 {
			m.downloadCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if m.downloadCursor < len(m.currentDownloadRows()) {
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
		return m.startDownloadProgress()
	case keyMatches(key, "n", "ㅜ"):
		m.downloadTranscribe = false
		return m.startDownloadProgress()
	case key == "left" || key == "right" || key == "tab":
		m.downloadTranscribe = !m.downloadTranscribe
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
	selectedRows := m.selectedDownloadRows()
	runCtx, cancel := context.WithCancel(m.ctx)
	progress := lectureDownloadModel{
		ctx:     runCtx,
		cancel:  cancel,
		service: m.service,
		request: LectureDownloadRequest{
			All:         true,
			Rows:        selectedRows,
			LectureIDs:  m.selectedDownloadIDs(),
			Concurrency: settings.Concurrency,
			Transcribe:  m.downloadTranscribe,
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
		if m.downloadProgress.done && (key.String() == "esc" || keyMatches(key.String(), "b", "ㅠ")) {
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

type downloadCourseGroup struct {
	name string
	rows []app.LectureRow
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

func (m model) updateConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.configEditing = ""
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.configInput.Value())
		if m.configEditing == "download.dir" {
			if _, err := m.service.SetDownloadConfig(value, 0); err != nil {
				m.err = err
			} else {
				m.err = nil
				m.configEditing = ""
				m.refreshConfigContent()
			}
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.configInput, cmd = m.configInput.Update(msg)
	return m, cmd
}

func (m model) startDownloadDirEdit() (tea.Model, tea.Cmd) {
	settings, err := m.service.DownloadSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	input := textinput.New()
	input.SetValue(settings.Dir)
	input.Placeholder = "다운로드 폴더"
	input.Prompt = "download.dir "
	input.Focus()
	m.configEditing = "download.dir"
	m.configInput = input
	return m, nil
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
	if _, err := m.service.SetDownloadConfig("", next); err != nil {
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
	m.content = formatConfig(settings)
	m.loadedAt = time.Now()
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
	case screenDownloadProgress:
		if m.downloadProgress == nil {
			return appStyle.Render(errorStyle.Render("다운로드 상태가 없습니다"))
		}
		return m.downloadProgress.View()
	}
	if m.active == screenHome {
		return appStyle.Render(m.renderHomeView(width))
	}

	header := m.renderHeader(width)
	rule := mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 120))))
	panel := panelStyle.Width(maxInt(48, width-4)).Render(m.renderPanel())
	footer := footerStyle.Render(m.footerHelp())
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, rule, "", panel, "", footer))
}

func (m model) footerHelp() string {
	if m.active == screenLectures {
		return "d 다운로드  b/esc 뒤로  r 새로고침  q 종료"
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

func (m model) renderPanel() string {
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
	if strings.TrimSpace(m.content) == "" {
		b.WriteString(emptyStyle.Render("표시할 내용이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(m.content)
	if !strings.HasSuffix(m.content, "\n") {
		b.WriteString("\n")
	}
	if m.active == screenConfig {
		b.WriteString("\n")
		if m.configEditing != "" {
			b.WriteString(m.configInput.View())
			b.WriteString("\n")
			b.WriteString(footerStyle.Render("enter 저장  esc 취소"))
		} else {
			b.WriteString(footerStyle.Render("d download.dir 편집  +/- 동시 다운로드"))
		}
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
	case screenDownloadSelect:
		return "Download"
	case screenDownloadConfirm:
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
	case screenConfig:
		return "현재 유저 설정"
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
	}) + "\n" + renderSection("Download", []string{
		"dir  " + settings.Download.Dir,
		fmt.Sprintf("concurrency  %d", settings.Download.Concurrency),
	})
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
