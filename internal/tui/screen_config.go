package tui

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
	"time"
)

type configRow struct {
	key      string
	page     int
	section  string
	label    string
	value    string
	hint     string
	editable bool
	cycle    bool
	reset    bool
}

const (
	configPageGeneral = iota
	configPageSchedule
	configPageDownload
)

var configPageLabels = []string{"일반", "일정", "다운로드"}

const directInputChoice = "[직접 입력]"

func (m model) updateConfigChoice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenConfig
		m.configChoiceKey = ""
		m.configChoiceCursor = 0
		m.err = nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		choices := m.currentConfigChoices()
		if len(choices) == 0 {
			m.configChoiceCursor = 0
		} else if m.configChoiceCursor <= 0 {
			m.configChoiceCursor = len(choices) - 1
		} else {
			m.configChoiceCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		choices := m.currentConfigChoices()
		if len(choices) == 0 || m.configChoiceCursor >= len(choices)-1 {
			m.configChoiceCursor = 0
		} else {
			m.configChoiceCursor++
		}
	case key == "enter":
		choices := m.currentConfigChoices()
		if len(choices) == 0 {
			return m, nil
		}
		if m.configChoiceCursor < 0 {
			m.configChoiceCursor = 0
		}
		if m.configChoiceCursor >= len(choices) {
			m.configChoiceCursor = len(choices) - 1
		}
		choice := choices[m.configChoiceCursor]
		row := configRow{key: m.configChoiceKey, label: configInputLabel(m.configChoiceKey), editable: true}
		if choice == directInputChoice {
			return m.startConfigEdit(row)
		}
		if err := m.applyConfigCategoryChoice(m.configChoiceKey, choice); err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.active = screenConfig
		m.configChoiceKey = ""
		m.configChoiceCursor = 0
		m.refreshConfigContent()
	}
	return m, nil
}

func (m model) updateConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c":
		return m, tea.Quit
	case key == "esc":
		m.active = screenConfig
		m.configEditing = ""
		m.err = nil
		return m, nil
	case key == "enter":
		value := strings.TrimSpace(m.configInput.Value())
		var err error
		switch m.configEditing {
		case "download.dir":
			_, err = m.service.SetDownloadConfig(value, 0, nil, nil)
		case "reminder.name":
			_, err = m.service.SetReminderConfig(value, false)
		case "calendar.name":
			_, err = m.service.SetAcademicCalendarConfig(value, false)
		case "timetable-calendar.name":
			_, err = m.service.SetTimetableCalendarConfig(value, false)
		case "reminder.alarm-before-min":
			var minutes int
			minutes, err = strconv.Atoi(value)
			if err != nil {
				err = errors.New("알림 시간에는 1 이상의 정수가 필요합니다")
			} else {
				_, err = m.service.UpdateConfig(app.ConfigUpdate{ReminderAlarmBeforeMin: &minutes})
			}
		}
		if err != nil {
			m.err = err
		} else {
			m.err = nil
			m.active = screenConfig
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
	rows := m.currentConfigRows()
	if len(rows) == 0 {
		m.configCursor = 0
		return
	}
	m.configCursor = (m.configCursor + delta + len(rows)) % len(rows)
}

func (m *model) moveConfigPage(delta int) {
	if len(configPageLabels) == 0 {
		m.configPage = 0
		m.configCursor = 0
		return
	}
	m.configPage = (m.configPage + delta + len(configPageLabels)) % len(configPageLabels)
	m.configCursor = 0
	m.clampConfigCursor()
}

func (m model) activateConfigCurrent() (tea.Model, tea.Cmd) {
	row, ok := m.currentConfigRow()
	if !ok {
		return m, nil
	}
	if row.reset {
		return m.resetConfigSettings()
	}
	if row.cycle {
		return m.startConfigChoice(row)
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
		return m.startConfigChoice(row)
	case "calendar.name":
		return m.startConfigChoice(row)
	case "timetable-calendar.name":
		return m.startConfigChoice(row)
	case "reminder.alarm-before-min":
		next := m.configSettings.Reminder.AlarmBeforeMin + delta*60
		if next < 1 {
			next = 1
		}
		if _, err := m.service.UpdateConfig(app.ConfigUpdate{ReminderAlarmBeforeMin: &next}); err != nil {
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

func (m model) startConfigChoice(row configRow) (tea.Model, tea.Cmd) {
	m.active = screenConfigChoice
	m.configChoiceKey = row.key
	m.configChoiceCursor = m.currentConfigChoiceIndex()
	m.err = nil
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
		input.Placeholder = "학사일정 캘린더"
	case "timetable-calendar.name":
		input.SetValue(m.configSettings.Calendar.TimetableName)
		input.Placeholder = "시간표 캘린더"
	case "reminder.alarm-before-min":
		input.SetValue(strconv.Itoa(m.configSettings.Reminder.AlarmBeforeMin))
		input.Placeholder = "분 단위 알림 시간"
	default:
		input.SetValue(row.value)
		input.Placeholder = row.label
	}
	input.Prompt = row.key + " "
	input.Focus()
	m.active = screenConfigInput
	m.configEditing = row.key
	m.configChoiceKey = ""
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
	if next > app.MaxTranscriptConcurrency {
		next = app.MaxTranscriptConcurrency
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
	msg := m.loadConfigMsg()
	if msg.err != nil {
		m.err = msg.err
		return
	}
	m.configSettings = msg.config
	m.configOptions = msg.categories
	m.configUsers = msg.users
	m.configTerms = msg.terms
	m.clampConfigCursor()
	m.content = msg.content
	m.loadedAt = time.Now()
}

func (m model) currentConfigRows() []configRow {
	rows := configRowsForPage(m.configSettings, m.configOptions, m.configPage)
	for index := range rows {
		switch rows[index].key {
		case "user.current":
			rows[index].value = currentUserLabel(m.configUsers)
		case "term.current":
			rows[index].value = currentTermLabel(m.configTerms, m.configSettings.Term)
		}
	}
	return rows
}

func (m model) currentConfigChoices() []string {
	switch m.configChoiceKey {
	case "user.current":
		return userChoices(m.configUsers)
	case "term.current":
		return termChoices(m.configTerms)
	case "reminder.name":
		return categoryChoices(m.configOptions.Reminders, m.configSettings.Reminder.ListName)
	case "calendar.name":
		return categoryChoices(m.configOptions.Calendars, m.configSettings.Calendar.Name)
	case "timetable-calendar.name":
		return categoryChoices(m.configOptions.Calendars, m.configSettings.Calendar.TimetableName)
	default:
		return nil
	}
}

func (m model) currentConfigChoiceIndex() int {
	current := ""
	switch m.configChoiceKey {
	case "user.current":
		choices := userChoices(m.configUsers)
		current = currentUserLabel(m.configUsers)
		for index, choice := range choices {
			if choice == current {
				return index
			}
		}
		return 0
	case "term.current":
		choices := termChoices(m.configTerms)
		current = currentTermChoice(m.configTerms, m.configSettings.Term)
		for index, choice := range choices {
			if choice == current {
				return index
			}
		}
		return 0
	case "reminder.name":
		current = m.configSettings.Reminder.ListName
	case "calendar.name":
		current = m.configSettings.Calendar.Name
	case "timetable-calendar.name":
		current = m.configSettings.Calendar.TimetableName
	default:
		return 0
	}
	index, _ := categoryChoiceIndex(m.currentConfigOptionsForKey(), current)
	return index
}

func (m model) currentConfigOptionsForKey() []string {
	switch m.configChoiceKey {
	case "user.current":
		return userChoices(m.configUsers)
	case "term.current":
		return termChoices(m.configTerms)
	case "reminder.name":
		return m.configOptions.Reminders
	case "calendar.name", "timetable-calendar.name":
		return m.configOptions.Calendars
	default:
		return nil
	}
}

func (m model) applyConfigCategoryChoice(key string, value string) error {
	useExisting := containsString(uniqueStrings(m.currentConfigOptionsForKey()), value)
	switch key {
	case "user.current":
		if err := m.service.SelectUser(m.ctx, strings.TrimSpace(value)); err != nil {
			return err
		}
		m.resetLoadedMainScreens()
		return nil
	case "term.current":
		selector := termSelectorFromChoice(value)
		if _, err := m.service.SelectTerm(m.ctx, selector, app.UserOption{}); err != nil {
			return err
		}
		m.resetLoadedMainScreens()
		return nil
	case "reminder.name":
		_, err := m.service.SetReminderConfig(value, useExisting)
		return err
	case "calendar.name":
		_, err := m.service.SetAcademicCalendarConfig(value, useExisting)
		return err
	case "timetable-calendar.name":
		_, err := m.service.SetTimetableCalendarConfig(value, useExisting)
		return err
	default:
		return nil
	}
}

func (m *model) clampConfigCursor() {
	rows := m.currentConfigRows()
	if len(rows) == 0 {
		m.configCursor = 0
		return
	}
	if m.configCursor < 0 {
		m.configCursor = 0
		return
	}
	if m.configCursor >= len(rows) {
		m.configCursor = len(rows) - 1
	}
}

func (m model) currentConfigRow() (configRow, bool) {
	rows := m.currentConfigRows()
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

func (m model) renderConfigChoiceView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render(configInputLabel(m.configChoiceKey)))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("하나만 선택할 수 있습니다."))
	b.WriteString("\n\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	choices := m.currentConfigChoices()
	if len(choices) == 0 {
		b.WriteString(emptyStyle.Render("선택할 항목이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	for index, choice := range choices {
		marker := "  "
		if index == m.configChoiceCursor {
			marker = "› "
		}
		check := "( )"
		if index == m.currentConfigChoiceIndex() && choice != directInputChoice {
			check = "(*)"
		}
		line := fmt.Sprintf("%s%s  %s", marker, check, choice)
		if index == m.configChoiceCursor {
			line = menuSelectedStyle.Render(line)
		} else if choice == directInputChoice {
			line = mutedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderHelpText("↑↓ 선택  |  enter 적용  |  b 뒤로  |  q 종료", width))
	return b.String()
}

func (m model) renderConfigInputView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render(configInputLabel(m.configEditing)))
	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	b.WriteString(m.configInput.View())
	b.WriteString("\n\n")
	b.WriteString(renderHelpText("enter 저장  |  esc 뒤로  |  q 종료", width))
	return b.String()
}

func (m model) renderConfigPanel(width int) string {
	rows := m.currentConfigRows()
	if len(rows) == 0 {
		return emptyStyle.Render("설정 항목이 없습니다") + "\n"
	}
	valueWidth := maxInt(16, minInt(42, width-44))
	var b strings.Builder
	b.WriteString(m.renderConfigPageTabs())
	b.WriteString("\n\n")
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
		hint := m.renderConfigHint(row)
		line := fmt.Sprintf("%s%s  %s  %s", marker, label, valueText, hint)
		if index == m.configCursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) renderConfigPageTabs() string {
	parts := make([]string, 0, len(configPageLabels))
	for index, label := range configPageLabels {
		if index == m.configPage {
			parts = append(parts, menuSelectedStyle.Render(label))
			continue
		}
		parts = append(parts, mutedStyle.Render(label))
	}
	return strings.Join(parts, mutedStyle.Render(" / "))
}

func (m model) renderConfigHint(row configRow) string {
	switch row.key {
	case "user.current":
		return mutedStyle.Render("enter 선택")
	case "term.current":
		return mutedStyle.Render("enter 선택")
	case "reminder.name":
		return mutedStyle.Render("enter 선택")
	case "calendar.name":
		return mutedStyle.Render("enter 선택")
	case "timetable-calendar.name":
		return mutedStyle.Render("enter 선택")
	default:
		return mutedStyle.Render(row.hint)
	}
}

func (m model) loadConfigMsg() loadMsg {
	settings, err := m.service.ConfigSettings()
	if err != nil {
		return loadMsg{err: err}
	}
	categories, err := m.service.CategoryOptions()
	if err != nil {
		return loadMsg{err: err}
	}
	users, err := m.service.Users(m.ctx)
	if err != nil {
		return loadMsg{err: err}
	}
	terms := []app.TermRow{}
	if len(users) > 0 {
		if rows, termErr := m.service.TermList(m.ctx, app.TermListOptions{}); termErr == nil {
			terms = rows
		}
	}
	return loadMsg{
		content:    formatConfig(settings),
		config:     settings,
		categories: categories,
		users:      users,
		terms:      terms,
	}
}

func configRows(settings app.ConfigSettings, options app.CategoryOptions) []configRow {
	_ = options
	return []configRow{
		{
			key:     "user.current",
			page:    configPageGeneral,
			section: "General",
			label:   "현재 계정",
			value:   "등록 계정 중 선택",
			hint:    "enter 선택",
			cycle:   true,
		},
		{
			key:     "term.current",
			page:    configPageGeneral,
			section: "General",
			label:   "현재 학기",
			value:   emptyFallback(settings.Term.Label, emptyFallback(settings.Term.Value, "자동")),
			hint:    "enter 선택",
			cycle:   true,
		},
		{
			key:      "reminder.name",
			page:     configPageGeneral,
			section:  "Reminder",
			label:    "미리알림 목록",
			value:    emptyFallback(settings.Reminder.ListName, app.DefaultReminderListName),
			editable: true,
			cycle:    true,
		},
		{
			key:      "reminder.alarm-before-min",
			page:     configPageGeneral,
			section:  "Reminder",
			label:    "알림 시간",
			value:    fmt.Sprintf("마감 %s 전", formatMinutes(settings.Reminder.AlarmBeforeMin)),
			hint:     "[] 60분 단위  enter 입력",
			editable: true,
		},
		{
			key:      "calendar.name",
			page:     configPageSchedule,
			section:  "Calendar",
			label:    "학사일정 캘린더",
			value:    emptyFallback(settings.Calendar.Name, app.DefaultAcademicCalendarName),
			editable: true,
			cycle:    true,
		},
		{
			key:      "timetable-calendar.name",
			page:     configPageSchedule,
			section:  "Calendar",
			label:    "시간표 캘린더",
			value:    emptyFallback(settings.Calendar.TimetableName, app.DefaultTimetableCalendarName),
			editable: true,
			cycle:    true,
		},
		{
			key:      "download.dir",
			page:     configPageDownload,
			section:  "Download",
			label:    "저장 폴더",
			value:    settings.Download.Dir,
			hint:     "enter 입력",
			editable: true,
		},
		{
			key:     "download.concurrency",
			page:    configPageDownload,
			section: "Download",
			label:   "동시 다운로드",
			value:   fmt.Sprintf("%d workers", settings.Download.Concurrency),
			hint:    "[] 변경",
		},
		{
			key:     "download.caffeinate",
			page:    configPageDownload,
			section: "Download",
			label:   "절전 방지",
			value:   enabledLabel(settings.Download.Caffeinate),
			hint:    "[] 전환",
		},
		{
			key:     "download.keep-partial",
			page:    configPageDownload,
			section: "Download",
			label:   "부분 파일 보존",
			value:   enabledLabel(settings.Download.KeepPartial),
			hint:    "[] 전환",
		},
		{
			key:     "transcript.concurrency",
			page:    configPageDownload,
			section: "Transcript",
			label:   "전사 worker",
			value:   fmt.Sprintf("%d workers", settings.Transcript.Concurrency),
			hint:    fmt.Sprintf("[] 1-%d", app.MaxTranscriptConcurrency),
		},
		{
			key:     "reset",
			page:    configPageGeneral,
			section: "General",
			label:   "설정 초기화",
			value:   "기본값으로 복원",
			hint:    "enter 실행",
			reset:   true,
		},
	}
}

func configRowsForPage(settings app.ConfigSettings, options app.CategoryOptions, page int) []configRow {
	rows := configRows(settings, options)
	filtered := make([]configRow, 0, len(rows))
	for _, row := range rows {
		if row.page == page {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func categoryChoices(options []string, current string) []string {
	values := uniqueStrings(options)
	filtered := values[:0]
	for _, value := range values {
		if value != directInputChoice {
			filtered = append(filtered, value)
		}
	}
	values = filtered
	current = strings.TrimSpace(current)
	if current != "" && !containsString(values, current) {
		values = append(values, current)
	}
	values = append(values, directInputChoice)
	return values
}

func categoryChoiceIndex(options []string, current string) (int, []string) {
	choices := categoryChoices(options, current)
	current = strings.TrimSpace(current)
	for index, value := range choices {
		if value == current {
			return index, choices
		}
	}
	return 0, choices
}

func userChoices(users []app.UserRow) []string {
	choices := make([]string, 0, len(users))
	for _, user := range users {
		studentID := strings.TrimSpace(user.User.StudentID)
		if studentID != "" {
			choices = append(choices, studentID)
		}
	}
	return choices
}

func currentUserLabel(users []app.UserRow) string {
	for _, user := range users {
		if user.Current && strings.TrimSpace(user.User.StudentID) != "" {
			return user.User.StudentID
		}
	}
	if len(users) == 1 {
		return users[0].User.StudentID
	}
	return "선택 필요"
}

func termChoices(terms []app.TermRow) []string {
	choices := make([]string, 0, len(terms))
	for _, row := range terms {
		value := strings.TrimSpace(row.Term.Value)
		if value == "" {
			continue
		}
		label := strings.TrimSpace(row.Term.Label)
		if label == "" {
			choices = append(choices, value)
			continue
		}
		choices = append(choices, value+"  "+label)
	}
	return choices
}

func currentTermChoice(terms []app.TermRow, settings app.TermSettings) string {
	currentValue := strings.TrimSpace(settings.Value)
	for _, row := range terms {
		if row.Current || (currentValue != "" && row.Term.Value == currentValue) {
			value := strings.TrimSpace(row.Term.Value)
			label := strings.TrimSpace(row.Term.Label)
			if label == "" {
				return value
			}
			return value + "  " + label
		}
	}
	return ""
}

func currentTermLabel(terms []app.TermRow, settings app.TermSettings) string {
	choice := currentTermChoice(terms, settings)
	if choice != "" {
		return choice
	}
	if len(terms) > 0 {
		return "선택 필요"
	}
	return emptyFallback(settings.Label, emptyFallback(settings.Value, "자동"))
}

func termSelectorFromChoice(choice string) string {
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return ""
	}
	return strings.Fields(choice)[0]
}

func configInputLabel(key string) string {
	switch key {
	case "user.current":
		return "현재 계정"
	case "term.current":
		return "현재 학기"
	case "download.dir":
		return "저장 폴더"
	case "reminder.name":
		return "미리알림 목록"
	case "calendar.name":
		return "학사일정 캘린더"
	case "timetable-calendar.name":
		return "시간표 캘린더"
	case "reminder.alarm-before-min":
		return "알림 시간"
	default:
		return "설정"
	}
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

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
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
		"academic-name  " + settings.Calendar.Name,
		fmt.Sprintf("academic-use-existing-list  %t", settings.Calendar.UseExistingList),
		"timetable-name  " + settings.Calendar.TimetableName,
		fmt.Sprintf("timetable-use-existing-list  %t", settings.Calendar.TimetableUseExistingList),
	}) + "\n" + renderSection("Download", []string{
		"dir  " + settings.Download.Dir,
		fmt.Sprintf("concurrency  %d", settings.Download.Concurrency),
		fmt.Sprintf("caffeinate  %t", settings.Download.Caffeinate),
		fmt.Sprintf("keep-partial  %t", settings.Download.KeepPartial),
	}) + "\n" + renderSection("Transcript", []string{
		fmt.Sprintf("concurrency  %d", settings.Transcript.Concurrency),
	})
}
