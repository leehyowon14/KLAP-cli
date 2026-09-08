package tui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type fakeConfigScreenService struct {
	configScreenService
	settings app.ConfigSettings
	writes   int
	reads    int
	update   app.ConfigUpdate
	err      error
	readErr  error
	selected string
}

func (s *fakeConfigScreenService) ConfigSettings() (app.ConfigSettings, error) {
	s.reads++
	return s.settings, s.readErr
}
func (s *fakeConfigScreenService) CategoryOptionsContext(context.Context) (app.CategoryOptions, error) {
	return app.CategoryOptions{}, nil
}
func (s *fakeConfigScreenService) Users(context.Context) ([]app.UserRow, error) { return nil, nil }
func (s *fakeConfigScreenService) UpdateConfig(update app.ConfigUpdate) (app.ConfigSettings, error) {
	s.writes++
	s.update = update
	if s.err == nil && update.ReminderAlarmBeforeMin != nil {
		s.settings.Reminder.AlarmBeforeMin = *update.ReminderAlarmBeforeMin
	}
	return s.settings, s.err
}
func (s *fakeConfigScreenService) SelectUser(_ context.Context, value string) error {
	s.writes++
	s.selected = value
	return s.err
}

func TestConfigChildSaveDefersIOAndRejectsDuplicateAndStaleResults(t *testing.T) {
	service := &fakeConfigScreenService{}
	m := configScreenModel{}
	m.startConfigEdit(configRow{key: "reminder.alarm-before-min"})
	m.configInput.SetValue("60")
	action, handled := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenConfigInput, false, context.Background(), service)
	if !handled || action.cmd == nil || !m.pending || service.writes != 0 || service.reads != 0 {
		t.Fatal("save not deferred")
	}
	duplicate, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenConfigInput, false, context.Background(), service)
	if duplicate.cmd != nil {
		t.Fatal("duplicate save")
	}
	msg := action.cmd().(configSavedMsg)
	if !m.Accepts(msg) || m.Accepts(configSavedMsg{generation: msg.generation + 1}) || service.writes != 1 {
		t.Fatal("result identity")
	}
	action = m.Saved(msg)
	if m.pending || m.Accepts(msg) || !action.navigate || action.target != screenConfig || m.configSettings.Reminder.AlarmBeforeMin != 60 || m.configEditing != "" {
		t.Fatalf("state=%+v action=%+v", m, action)
	}
}
func TestConfigChildFailurePreservesInputAndAllowsRetry(t *testing.T) {
	want := errors.New("save failed")
	service := &fakeConfigScreenService{err: want}
	m := configScreenModel{}
	m.startConfigEdit(configRow{key: "reminder.alarm-before-min"})
	m.configInput.SetValue("60")
	action, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenConfigInput, false, context.Background(), service)
	action = m.Saved(action.cmd().(configSavedMsg))
	if !errors.Is(action.err, want) || action.navigate || m.pending || m.configInput.Value() != "60" || m.configEditing == "" || service.reads != 0 {
		t.Fatalf("action=%+v state=%+v", action, m)
	}
	service.err = nil
	action, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenConfigInput, false, context.Background(), service)
	if action.cmd == nil {
		t.Fatal("retry absent")
	}
}
func TestConfigChildCancellationDoesNotWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &fakeConfigScreenService{}
	m := configScreenModel{}
	action := m.save(ctx, service, func() error { service.writes++; return nil }, false)
	msg := action.cmd().(configSavedMsg)
	if !errors.Is(msg.err, context.Canceled) || service.writes != 0 || service.reads != 0 {
		t.Fatalf("message=%+v", msg)
	}
}

func TestHomeToConfigStillLoadsUncachedSettings(t *testing.T) {
	m := model{active: screenHome}
	next, cmd := m.applyChildAction(childAction{navigate: true, target: screenConfig})
	got := next.(model)
	if got.active != screenConfig || !got.loading || cmd == nil {
		t.Fatal("Home entry bypassed Config loader")
	}
	m = model{active: screenConfigInput}
	next, cmd = m.applyChildAction(configRoute(screenConfig))
	if next.(model).loading || cmd != nil {
		t.Fatal("internal return should retain loaded settings")
	}
}
func TestConfigChoiceInvalidatesRootOnlyAfterSuccessfulSelection(t *testing.T) {
	for _, failed := range []bool{false, true} {
		service := &fakeConfigScreenService{readErr: errors.New("refresh failed")}
		if failed {
			service.err = errors.New("select failed")
		}
		child := configScreenModel{configChoiceKey: "user.current", configUsers: []app.UserRow{{User: app.User{StudentID: "user"}}}}
		action, _ := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, screenConfigChoice, false, context.Background(), service)
		if service.writes != 0 {
			t.Fatal("eager select")
		}
		msg := action.cmd().(configSavedMsg)
		root := model{active: screenConfigChoice, config: child, loadedScreens: map[screen]bool{screenDashboard: true}, dashboard: dashboardScreenModel{dashboardPage: 2}}
		next, _ := root.Update(msg)
		got := next.(model)
		if failed {
			if !got.loadedScreens[screenDashboard] || got.active != screenConfigChoice {
				t.Fatal("failed selection invalidated state")
			}
		} else if got.loadedScreens[screenDashboard] || got.dashboard.dashboardPage != 0 || got.active != screenConfig || got.err == nil {
			t.Fatal("successful selection must invalidate even if refresh fails")
		}
	}
}

func TestConfigValueChangesPreserveDomainSettings(t *testing.T) {
	tests := []struct {
		name   string
		change func(configScreenService) error
		check  func(app.ConfigSettings) bool
	}{
		{"download-dir", func(s configScreenService) error { return saveConfigInput(s, "download.dir", "custom") }, func(s app.ConfigSettings) bool { return s.Download.Dir == "custom" }},
		{"reminder-name", func(s configScreenService) error {
			return saveConfigChoice(context.Background(), s, "reminder.name", "personal", true)
		}, func(s app.ConfigSettings) bool {
			return s.Reminder.ListName == "personal" && s.Reminder.UseExistingList
		}},
		{"academic-name", func(s configScreenService) error { return saveConfigInput(s, "calendar.name", "academic") }, func(s app.ConfigSettings) bool { return s.Calendar.Name == "academic" && !s.Calendar.UseExistingList }},
		{"timetable-name", func(s configScreenService) error {
			return saveConfigChoice(context.Background(), s, "timetable-calendar.name", "timetable", true)
		}, func(s app.ConfigSettings) bool {
			return s.Calendar.TimetableName == "timetable" && s.Calendar.TimetableUseExistingList
		}},
		{"download-lower-bound", func(s configScreenService) error { return adjustConfigValue(s, "download.concurrency", -100) }, func(s app.ConfigSettings) bool { return s.Download.Concurrency == 1 }},
		{"transcript-lower-bound", func(s configScreenService) error { return adjustConfigValue(s, "transcript.concurrency", -100) }, func(s app.ConfigSettings) bool { return s.Transcript.Concurrency == 1 }},
		{"transcript-upper-bound", func(s configScreenService) error { return adjustConfigValue(s, "transcript.concurrency", 100) }, func(s app.ConfigSettings) bool { return s.Transcript.Concurrency == app.MaxTranscriptConcurrency }},
		{"caffeinate", func(s configScreenService) error { return adjustConfigValue(s, "download.caffeinate", 1) }, func(s app.ConfigSettings) bool { return !s.Download.Caffeinate }},
		{"keep-partial", func(s configScreenService) error { return adjustConfigValue(s, "download.keep-partial", 1) }, func(s app.ConfigSettings) bool { return s.Download.KeepPartial }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTUITestService(t)
			if err := tt.change(service); err != nil {
				t.Fatal(err)
			}
			settings, err := service.ConfigSettings()
			if err != nil || !tt.check(settings) {
				t.Fatalf("settings=%+v err=%v", settings, err)
			}
		})
	}
}
