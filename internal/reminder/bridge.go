package reminder

import "time"

type Assignment struct {
	ID              string     `json:"id"`
	LegacyIDs       []string   `json:"legacyIds,omitempty"`
	TermValue       string     `json:"termValue,omitempty"`
	Title           string     `json:"title"`
	Course          string     `json:"course"`
	DueAt           *time.Time `json:"dueAt"`
	Submitted       bool       `json:"submitted"`
	DetailURL       string     `json:"detailUrl"`
	Notes           string     `json:"notes"`
	KnownSourceHash string     `json:"knownSourceHash,omitempty"`
	ForceUpdate     bool       `json:"forceUpdate,omitempty"`
}

type SyncRequest struct {
	ListName        string       `json:"listName"`
	UseExistingList bool         `json:"useExistingList"`
	AlarmBeforeMin  int          `json:"alarmBeforeMin"`
	Assignments     []Assignment `json:"assignments"`
}

type SyncResult struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Completed int      `json:"completed"`
	Skipped   int      `json:"skipped"`
	SyncedIDs []string `json:"syncedIds"`
}
