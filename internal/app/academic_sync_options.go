package app

type AcademicSyncOptions struct {
	Query     AcademicListOptions
	Decisions map[string]SyncDecision
}
