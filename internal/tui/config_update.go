package tui

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type configScreenModel struct {
	configEditing      string
	configInput        textinput.Model
	configSettings     app.ConfigSettings
	configOptions      app.CategoryOptions
	configUsers        []app.UserRow
	configTerms        []app.TermRow
	configCursor       int
	configPage         int
	configChoiceKey    string
	configChoiceCursor int
	pending            bool
	generation         uint64
}

type configScreenService interface {
	ConfigSettings() (app.ConfigSettings, error)
	CategoryOptionsContext(context.Context) (app.CategoryOptions, error)
	Users(context.Context) ([]app.UserRow, error)
	TermList(context.Context, app.TermListOptions) ([]app.TermRow, error)
	SelectUser(context.Context, string) error
	SelectTerm(context.Context, string, app.UserOption) (app.TermSettings, error)
	SetDownloadConfig(string, int, *bool, *bool) (app.DownloadSettings, error)
	SetReminderConfig(string, bool) (app.ReminderSettings, error)
	SetAcademicCalendarConfig(string, bool) (app.CalendarSettings, error)
	SetTimetableCalendarConfig(string, bool) (app.CalendarSettings, error)
	UpdateConfig(app.ConfigUpdate) (app.ConfigSettings, error)
	DownloadSettings() (app.DownloadSettings, error)
	TranscriptSettings() (app.TranscriptSettings, error)
	SetTranscriptConfig(int) (app.TranscriptSettings, error)
	ResetConfigSettings() (app.ConfigSettings, error)
}

type configSavedMsg struct {
	generation uint64
	err        error
	loaded     *loadMsg
	invalidate bool
}

func configRoute(target screen) childAction {
	return childAction{navigate: true, target: target, setError: true}
}

func (m *configScreenModel) Loaded(msg loadMsg) {
	m.configSettings, m.configOptions, m.configUsers, m.configTerms = msg.config, msg.categories, msg.users, msg.terms
	m.clampConfigCursor()
}
func (m *configScreenModel) UsersLoaded(users []app.UserRow) { m.configUsers = users }
func (m configScreenModel) Accepts(msg configSavedMsg) bool {
	return m.pending && m.generation == msg.generation
}
func (m *configScreenModel) Saved(msg configSavedMsg) childAction {
	m.pending = false
	if msg.err != nil {
		return childAction{setError: true, err: msg.err}
	}
	m.configEditing, m.configChoiceKey, m.configChoiceCursor = "", "", 0
	action := configRoute(screenConfig)
	if msg.loaded != nil {
		if msg.loaded.err != nil {
			action.err = msg.loaded.err
		} else {
			m.Loaded(*msg.loaded)
		}
	}
	return action
}

func (m *configScreenModel) save(ctx context.Context, service configScreenService, operation func() error, invalidate bool) childAction {
	m.pending = true
	m.generation++
	generation := m.generation
	return childAction{setError: true, cmd: func() tea.Msg {
		if err := ctx.Err(); err != nil {
			return configSavedMsg{generation: generation, err: err}
		}
		if err := operation(); err != nil {
			return configSavedMsg{generation: generation, err: err}
		}
		loaded := loadConfigMsg(ctx, service)
		return configSavedMsg{generation: generation, loaded: &loaded, invalidate: invalidate}
	}}
}

func (m *configScreenModel) Update(msg tea.Msg, route screen, loading bool, ctx context.Context, service configScreenService) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	k := key.String()
	if k == "ctrl+c" || (route != screenConfigInput && keyMatches(k, "q", "ㅂ")) {
		return childAction{cmd: tea.Quit}, true
	}
	if m.pending {
		return childAction{}, true
	}
	if route == screenConfigInput {
		switch k {
		case "esc":
			m.configEditing = ""
			return configRoute(screenConfig), true
		case "enter":
			field, value := m.configEditing, strings.TrimSpace(m.configInput.Value())
			return m.save(ctx, service, func() error { return saveConfigInput(service, field, value) }, false), true
		}
		var cmd tea.Cmd
		m.configInput, cmd = m.configInput.Update(msg)
		return childAction{cmd: cmd}, true
	}
	if route == screenConfigChoice {
		choices := m.currentConfigChoices()
		switch {
		case k == "esc" || keyMatches(k, "b", "ㅠ"):
			m.configChoiceKey, m.configChoiceCursor = "", 0
			return configRoute(screenConfig), true
		case k == "up" || keyMatches(k, "k", "ㅏ"):
			if len(choices) == 0 {
				m.configChoiceCursor = 0
			} else {
				m.configChoiceCursor = (m.configChoiceCursor - 1 + len(choices)) % len(choices)
			}
		case k == "down" || keyMatches(k, "j", "ㅓ"):
			if len(choices) == 0 {
				m.configChoiceCursor = 0
			} else {
				m.configChoiceCursor = (m.configChoiceCursor + 1) % len(choices)
			}
		case k == "enter":
			if len(choices) == 0 {
				return childAction{}, true
			}
			m.configChoiceCursor = clampInt(m.configChoiceCursor, 0, len(choices)-1)
			field, value := m.configChoiceKey, choices[m.configChoiceCursor]
			if value == directInputChoice {
				return m.startConfigEdit(configRow{key: field, label: configInputLabel(field), editable: true}), true
			}
			existing := containsString(uniqueStrings(m.currentConfigOptionsForKey()), value)
			invalidate := field == "user.current" || field == "term.current"
			return m.save(ctx, service, func() error { return saveConfigChoice(ctx, service, field, value, existing) }, invalidate), true
		}
		return childAction{}, true
	}
	if loading {
		return childAction{}, false
	}
	switch {
	case k == "up" || keyMatches(k, "k", "ㅏ"):
		m.moveConfigCursor(-1)
	case k == "down" || keyMatches(k, "j", "ㅓ"):
		m.moveConfigCursor(1)
	case k == "left":
		m.moveConfigPage(-1)
	case k == "right":
		m.moveConfigPage(1)
	case k == "enter":
		row, ok := m.currentConfigRow()
		if !ok {
			return childAction{}, true
		}
		if row.reset {
			return m.save(ctx, service, func() error { _, err := service.ResetConfigSettings(); return err }, false), true
		}
		if row.cycle {
			return m.startConfigChoice(row), true
		}
		if row.editable {
			return m.startConfigEdit(row), true
		}
		return m.adjust(ctx, service, 1), true
	case k == "tab" || k == "]":
		return m.adjust(ctx, service, 1), true
	case k == "shift+tab" || k == "backtab" || k == "[":
		return m.adjust(ctx, service, -1), true
	default:
		return childAction{}, false
	}
	return childAction{}, true
}

func (m *configScreenModel) startConfigChoice(row configRow) childAction {
	m.configChoiceKey = row.key
	m.configChoiceCursor = m.currentConfigChoiceIndex()
	return configRoute(screenConfigChoice)
}
func (m *configScreenModel) adjust(ctx context.Context, service configScreenService, delta int) childAction {
	row, ok := m.currentConfigRow()
	if !ok {
		return childAction{}
	}
	switch row.key {
	case "reminder.name", "calendar.name", "timetable-calendar.name":
		return m.startConfigChoice(row)
	case "reminder.alarm-before-min":
		next := maxInt(1, m.configSettings.Reminder.AlarmBeforeMin+delta*60)
		return m.save(ctx, service, func() error {
			_, err := service.UpdateConfig(app.ConfigUpdate{ReminderAlarmBeforeMin: &next})
			return err
		}, false)
	case "download.concurrency", "download.caffeinate", "download.keep-partial", "transcript.concurrency":
		return m.save(ctx, service, func() error { return adjustConfigValue(service, row.key, delta) }, false)
	default:
		return childAction{}
	}
}

func saveConfigInput(service configScreenService, key, value string) error {
	switch key {
	case "download.dir":
		_, err := service.SetDownloadConfig(value, 0, nil, nil)
		return err
	case "reminder.name":
		_, err := service.SetReminderConfig(value, false)
		return err
	case "calendar.name":
		_, err := service.SetAcademicCalendarConfig(value, false)
		return err
	case "timetable-calendar.name":
		_, err := service.SetTimetableCalendarConfig(value, false)
		return err
	case "reminder.alarm-before-min":
		minutes, err := strconv.Atoi(value)
		if err != nil {
			return errors.New("알림 시간에는 1 이상의 정수가 필요합니다")
		}
		_, err = service.UpdateConfig(app.ConfigUpdate{ReminderAlarmBeforeMin: &minutes})
		return err
	default:
		return nil
	}
}
func saveConfigChoice(ctx context.Context, service configScreenService, key, value string, existing bool) error {
	switch key {
	case "user.current":
		return service.SelectUser(ctx, strings.TrimSpace(value))
	case "term.current":
		_, err := service.SelectTerm(ctx, termSelectorFromChoice(value), app.UserOption{})
		return err
	case "reminder.name":
		_, err := service.SetReminderConfig(value, existing)
		return err
	case "calendar.name":
		_, err := service.SetAcademicCalendarConfig(value, existing)
		return err
	case "timetable-calendar.name":
		_, err := service.SetTimetableCalendarConfig(value, existing)
		return err
	default:
		return nil
	}
}
func adjustConfigValue(service configScreenService, key string, delta int) error {
	if key == "transcript.concurrency" {
		settings, err := service.TranscriptSettings()
		if err != nil {
			return err
		}
		_, err = service.SetTranscriptConfig(clampInt(settings.Concurrency+delta, 1, app.MaxTranscriptConcurrency))
		return err
	}
	settings, err := service.DownloadSettings()
	if err != nil {
		return err
	}
	switch key {
	case "download.concurrency":
		_, err = service.SetDownloadConfig("", maxInt(1, settings.Concurrency+delta), nil, nil)
	case "download.caffeinate":
		next := !settings.Caffeinate
		_, err = service.SetDownloadConfig("", 0, &next, nil)
	case "download.keep-partial":
		next := !settings.KeepPartial
		_, err = service.SetDownloadConfig("", 0, nil, &next)
	}
	return err
}

func (m configScreenModel) View(width int, route screen, err error) string {
	switch route {
	case screenConfigChoice:
		return m.renderConfigChoiceView(width, err)
	case screenConfigInput:
		return m.renderConfigInputView(width, err)
	default:
		return m.renderConfigPanel(width)
	}
}
