package app

type LectureTransferStage string

const (
	LectureStageResolve         LectureTransferStage = "resolve"
	LectureStageDownload        LectureTransferStage = "download"
	LectureStageDone            LectureTransferStage = "done"
	LectureStageSkip            LectureTransferStage = "skip"
	LectureStageError           LectureTransferStage = "error"
	LectureStageTranscribe      LectureTransferStage = "transcribe"
	LectureStageTranscribed     LectureTransferStage = "transcribed"
	LectureStageTranscriptError LectureTransferStage = "transcript-error"
)
