package calendar

import "time"

type Event struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	StartAt         time.Time  `json:"startAt"`
	EndAt           time.Time  `json:"endAt"`
	AllDay          bool       `json:"allDay"`
	Notes           string     `json:"notes"`
	URL             string     `json:"url"`
	Recurrence      string     `json:"recurrence,omitempty"`
	RecurrenceEnd   *time.Time `json:"recurrenceEnd,omitempty"`
	KnownSourceHash string     `json:"knownSourceHash,omitempty"`
	ForceUpdate     bool       `json:"forceUpdate,omitempty"`
}

type SyncRequest struct {
	CalendarName    string  `json:"calendarName"`
	UseExistingList bool    `json:"useExistingList"`
	Events          []Event `json:"events"`
}

type SyncResult struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Skipped   int      `json:"skipped"`
	SyncedIDs []string `json:"syncedIds"`
}
