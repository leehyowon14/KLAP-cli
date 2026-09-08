package app

type AssignmentSyncOptions struct {
	Query     AssignmentListOptions
	Decisions map[string]SyncDecision
}
