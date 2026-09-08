package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestConfigAlarmInputUsesTypedUpdate(t *testing.T) {
	for _, value := range []string{"60", "0", "invalid"} {
		t.Run(value, func(t *testing.T) {
			service := newTUITestService(t)
			before, err := service.ConfigSettings()
			if err != nil {
				t.Fatal(err)
			}
			input := textinput.New()
			input.SetValue(value)
			m := model{service: service, active: screenConfigInput, configEditing: "reminder.alarm-before-min", configInput: input}
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := next.(model)
			after, err := service.ConfigSettings()
			if err != nil {
				t.Fatal(err)
			}
			if value == "60" {
				if got.err != nil || after.Reminder.AlarmBeforeMin != 60 || got.active != screenConfig {
					t.Fatalf("state=%v settings=%#v error=%v", got.active, after, got.err)
				}
			} else {
				if got.err == nil || after.Reminder.AlarmBeforeMin != before.Reminder.AlarmBeforeMin || got.active != screenConfigInput {
					t.Fatalf("invalid state=%v settings=%#v error=%v", got.active, after, got.err)
				}
			}
		})
	}
}
