package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"reflect"
	"strings"
	"testing"
)

func TestFormatConfigShowsDownloadConcurrency(t *testing.T) {
	view := formatConfig(app.ConfigSettings{
		Reminder: app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
		Calendar: app.CalendarSettings{
			Name:                     "학사일정",
			UseExistingList:          true,
			TimetableName:            "시간표",
			TimetableUseExistingList: true,
		},
		Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 7, Caffeinate: true, KeepPartial: false},
		Transcript: app.TranscriptSettings{Concurrency: 2},
	})
	if !strings.Contains(view, "Calendar") || !strings.Contains(view, "academic-name  학사일정") || !strings.Contains(view, "timetable-name  시간표") {
		t.Fatalf("formatConfig() missing calendar: %q", view)
	}
	if !strings.Contains(view, "concurrency  7") {
		t.Fatalf("formatConfig() missing concurrency: %q", view)
	}
	if !strings.Contains(view, "caffeinate  true") {
		t.Fatalf("formatConfig() missing caffeinate: %q", view)
	}
	if !strings.Contains(view, "keep-partial  false") {
		t.Fatalf("formatConfig() missing keep-partial: %q", view)
	}
	if !strings.Contains(view, "Transcript") || !strings.Contains(view, "concurrency  2") {
		t.Fatalf("formatConfig() missing transcript concurrency: %q", view)
	}
}

func TestConfigRowsExposeCategorySelection(t *testing.T) {
	settings := app.ConfigSettings{
		Reminder: app.ReminderSettings{ListName: "To-do", UseExistingList: true, AlarmBeforeMin: 60},
		Calendar: app.CalendarSettings{
			Name:                     "학사일정",
			UseExistingList:          true,
			TimetableName:            "시간표",
			TimetableUseExistingList: true,
		},
		Download: app.DownloadSettings{
			Dir:         "downloads",
			Concurrency: 4,
			Caffeinate:  true,
			KeepPartial: false,
		},
		Transcript: app.TranscriptSettings{Concurrency: 1},
	}
	options := app.CategoryOptions{
		Reminders: []string{"개인", "To-do"},
		Calendars: []string{"개인", "학사일정", "시간표"},
	}
	rows := configRows(settings, options)

	if len(rows) == 0 {
		t.Fatal("configRows() returned no rows")
	}
	var reminderRow configRow
	for _, row := range rows {
		if row.key == "reminder.name" {
			reminderRow = row
			break
		}
	}
	if reminderRow.key != "reminder.name" || !reminderRow.cycle || !reminderRow.editable {
		t.Fatalf("reminder row = %+v", reminderRow)
	}
	scheduleRows := configRowsForPage(settings, options, configPageSchedule)
	foundAcademicCalendar := false
	foundTimetableCalendar := false
	for _, row := range scheduleRows {
		if row.key == "calendar.name" && row.value == "학사일정" && row.page == configPageSchedule {
			foundAcademicCalendar = true
		}
		if row.key == "timetable-calendar.name" && row.value == "시간표" && row.page == configPageSchedule {
			foundTimetableCalendar = true
		}
	}
	if !foundAcademicCalendar || !foundTimetableCalendar {
		t.Fatalf("configRows() missing calendar category row: %+v", rows)
	}
}

func TestConfigRowsShowCurrentUserAndTerm(t *testing.T) {
	m := model{
		configSettings: app.ConfigSettings{
			Term: app.TermSettings{Value: "2026-1", Label: "2026년도 1학기"},
		},
		configUsers: []app.UserRow{
			{User: app.User{StudentID: "20250001"}},
			{User: app.User{StudentID: "20250002"}, Current: true},
		},
		configTerms: []app.TermRow{
			{Term: app.Term{Value: "2025-2", Label: "2025년도 2학기"}},
			{Term: app.Term{Value: "2026-1", Label: "2026년도 1학기"}, Current: true},
		},
	}
	rows := m.currentConfigRows()
	if len(rows) < 2 || rows[0].key != "user.current" || rows[0].value != "20250002" {
		t.Fatalf("user config row = %+v", rows)
	}
	if rows[1].key != "term.current" || rows[1].value != "2026-1  2026년도 1학기" {
		t.Fatalf("term config row = %+v", rows[1])
	}
	m.configSettings.Term = app.TermSettings{Value: "2024-1", Label: "2024년도 1학기"}
	m.configTerms[1].Current = false
	if got := currentTermLabel(m.configTerms, m.configSettings.Term); got != "선택 필요" {
		t.Fatalf("currentTermLabel(unmatched) = %q", got)
	}
}

func TestConfigChoicesUseRegisteredUsersAndTerms(t *testing.T) {
	m := model{
		configUsers: []app.UserRow{
			{User: app.User{StudentID: "20250001"}},
			{User: app.User{StudentID: "20250002"}, Current: true},
		},
		configTerms: []app.TermRow{
			{Term: app.Term{Value: "2025-2", Label: "2025년도 2학기"}},
			{Term: app.Term{Value: "2026-1", Label: "2026년도 1학기"}, Current: true},
		},
	}
	m.configChoiceKey = "user.current"
	if got := m.currentConfigChoices(); !reflect.DeepEqual(got, []string{"20250001", "20250002"}) {
		t.Fatalf("user choices = %#v", got)
	}
	if got := m.currentConfigChoiceIndex(); got != 1 {
		t.Fatalf("user choice index = %d", got)
	}
	m.configChoiceKey = "term.current"
	if got := m.currentConfigChoices(); !reflect.DeepEqual(got, []string{"2025-2  2025년도 2학기", "2026-1  2026년도 1학기"}) {
		t.Fatalf("term choices = %#v", got)
	}
	if got := m.currentConfigChoiceIndex(); got != 1 {
		t.Fatalf("term choice index = %d", got)
	}
	if got := termSelectorFromChoice("2026-1  2026년도 1학기"); got != "2026-1" {
		t.Fatalf("termSelectorFromChoice() = %q", got)
	}
}

func TestConfigCursorWraps(t *testing.T) {
	m := model{
		configSettings: app.ConfigSettings{
			Reminder:   app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Calendar:   app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 3, Caffeinate: true},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"To-do"}, Calendars: []string{"시간표"}},
		configPage:    configPageDownload,
	}
	m.moveConfigCursor(-1)
	if m.configCursor != len(configRowsForPage(m.configSettings, m.configOptions, configPageDownload))-1 {
		t.Fatalf("configCursor after up wrap = %d", m.configCursor)
	}
	m.moveConfigCursor(1)
	if m.configCursor != 0 {
		t.Fatalf("configCursor after down wrap = %d", m.configCursor)
	}
}

func TestConfigPageNavigationResetsCursor(t *testing.T) {
	m := model{
		configSettings: app.ConfigSettings{
			Reminder:   app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Calendar:   app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 3, Caffeinate: true},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"To-do"}, Calendars: []string{"시간표"}},
		configCursor:  2,
	}
	m.moveConfigPage(1)
	if m.configPage != configPageSchedule || m.configCursor != 0 {
		t.Fatalf("config page/cursor = %d/%d", m.configPage, m.configCursor)
	}
}

func TestConfigArrowKeysNavigatePages(t *testing.T) {
	m := model{
		active: screenConfig,
		configSettings: app.ConfigSettings{
			Reminder:   app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Calendar:   app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 3, Caffeinate: true},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"To-do"}, Calendars: []string{"시간표"}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := updated.(model)
	if got.configPage != configPageSchedule {
		t.Fatalf("configPage after right = %d", got.configPage)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = updated.(model)
	if got.configPage != configPageGeneral {
		t.Fatalf("configPage after left = %d", got.configPage)
	}
}

func TestCategoryChoicesAppendDirectInput(t *testing.T) {
	got := categoryChoices([]string{"개인", "회사"}, "시간표")
	want := []string{"개인", "회사", "시간표", directInputChoice}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categoryChoices() = %#v, want %#v", got, want)
	}
}

func TestConfigEnterOpensChoiceViewForCategories(t *testing.T) {
	m := model{
		active: screenConfig,
		configSettings: app.ConfigSettings{
			Reminder: app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"개인", "To-do"}},
		configCursor:  2,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if got.active != screenConfigChoice || got.configChoiceKey != "reminder.name" {
		t.Fatalf("config choice state = %v/%q", got.active, got.configChoiceKey)
	}
}

func TestConfigEnterOpensInputViewForTextRows(t *testing.T) {
	m := model{
		active: screenConfig,
		configSettings: app.ConfigSettings{
			Reminder: app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Download: app.DownloadSettings{Dir: "downloads", Concurrency: 3},
		},
		configPage: configPageDownload,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if got.active != screenConfigInput || got.configEditing != "download.dir" {
		t.Fatalf("config input state = %v/%q", got.active, got.configEditing)
	}
}
