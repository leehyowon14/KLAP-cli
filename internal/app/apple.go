package app

// WithAppleAdapters returns an independent service facade. KLAS requests,
// conflict policy and synchronization state still belong to the same core.
// Callers can replace only the final platform I/O, for example with a paired Mac.
func (s *Service) WithAppleAdapters(reminders ReminderSyncer, calendar CalendarSyncer, categories CategoryLister) *Service {
	copy := *s
	copy.requestSession = nil
	copy.reminderSyncer, copy.calendarSyncer, copy.categoryLister = reminders, calendar, categories
	return &copy
}
