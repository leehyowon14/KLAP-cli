package settings

func (s *Settings) Normalize() {
	if s.Reminder.ListName == "" {
		s.Reminder.ListName = DefaultReminderListName
	}
	if s.Reminder.AlarmBeforeMin <= 0 {
		s.Reminder.AlarmBeforeMin = 24 * 60
	}
	s.Calendar.Normalize()
	s.Download.Normalize()
	s.Transcript.Normalize()
}

func (c *Calendar) Normalize() {
	if c.AcademicName == "" {
		if c.Name != "" {
			c.AcademicName = c.Name
			c.AcademicUseExistingList = c.UseExistingList
		} else {
			c.AcademicName = DefaultAcademicCalendarName
		}
	}
	if c.TimetableName == "" {
		c.TimetableName = DefaultTimetableCalendarName
	}
	c.Name = c.AcademicName
	c.UseExistingList = c.AcademicUseExistingList
}

func DownloadCaffeinateEnabled(download Download) bool {
	download.Normalize()
	return download.Caffeinate != nil && *download.Caffeinate
}

func (d *Download) Normalize() {
	if d.Dir == "" || d.Dir == "downloads" {
		d.Dir = DefaultDownloadDir()
	}
	if d.Concurrency <= 0 {
		d.Concurrency = DefaultDownloadConcurrency
	}
	if d.Caffeinate == nil {
		d.Caffeinate = boolPtr(true)
	}
}

func (t *Transcript) Normalize() {
	if t.Concurrency <= 0 {
		t.Concurrency = DefaultTranscriptConcurrency
	}
	if t.Concurrency > MaxTranscriptConcurrency {
		t.Concurrency = MaxTranscriptConcurrency
	}
}
