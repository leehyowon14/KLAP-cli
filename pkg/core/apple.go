package core

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/calendar"
	"github.com/leehyowon14/KLAP-cli/internal/category"
	"github.com/leehyowon14/KLAP-cli/internal/platform/macos"
	"github.com/leehyowon14/KLAP-cli/internal/reminder"
)

type (
	ReminderSyncer  = app.ReminderSyncer
	CalendarSyncer  = app.CalendarSyncer
	CategoryLister  = app.CategoryLister
	ReminderRequest = reminder.SyncRequest
	ReminderResult  = reminder.SyncResult
	CalendarRequest = calendar.SyncRequest
	CalendarResult  = calendar.SyncResult
	AppleCategories = category.Options
)

// NativeAppleAdapters use the existing signed Swift helpers. The caller supplies
// trusted installation paths; remote requests never choose an executable.
func NativeAppleAdapters(reminderPath, calendarPath, categoryPath string) (ReminderSyncer, CalendarSyncer, CategoryLister) {
	return macos.NewReminderBridge(reminderPath), macos.NewCalendarBridge(calendarPath), macos.NewCategoryBridge(categoryPath)
}
