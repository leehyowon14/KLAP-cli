package app

type TimetableSyncOptions struct {
	Query     TimetableOptions
	Decisions map[string]SyncDecision
}
