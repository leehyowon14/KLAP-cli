package app

type LectureSyncOptions struct {
	Query     LectureListOptions
	Decisions map[string]SyncDecision
}
