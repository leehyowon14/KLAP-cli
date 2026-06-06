package settings

import "testing"

func TestDefaultReminderSettings(t *testing.T) {
	got := Default()
	if got.Reminder.ListName != "Kwangwoon Univ." {
		t.Fatalf("ListName = %q", got.Reminder.ListName)
	}
	if got.Reminder.UseExistingList {
		t.Fatal("UseExistingList should default to false")
	}
	if got.Reminder.AlarmBeforeMin != 1440 {
		t.Fatalf("AlarmBeforeMin = %d", got.Reminder.AlarmBeforeMin)
	}
}

func TestNormalizeReminderSettings(t *testing.T) {
	settings := Settings{}
	settings.Normalize()
	if settings.Reminder.ListName != "Kwangwoon Univ." {
		t.Fatalf("ListName = %q", settings.Reminder.ListName)
	}
	if settings.Reminder.AlarmBeforeMin != 1440 {
		t.Fatalf("AlarmBeforeMin = %d", settings.Reminder.AlarmBeforeMin)
	}
}
