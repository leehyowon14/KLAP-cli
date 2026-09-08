package app

import "context"

// Downloader owns HTTP transfer, partial-file resume and final file promotion.
// Calls are blocking; progress is synchronous and observes completed writes.
type Downloader interface {
	Status(string) (DownloadStatusResult, error)
	VideoPath(string, string, LectureFileName, string, int) string
	DownloadFile(context.Context, string, string, bool, func(int64, int64)) (int64, error)
}

// LectureFileName is the normalized metadata used by the filename adapter.
// Week selection remains a lecture-domain rule, independent of filesystem names.
type LectureFileName struct {
	ModuleTitle, Title, ContentID, LearningSeq string
	Week                                       int
}

func lectureFileName(lecture Lecture) LectureFileName {
	return LectureFileName{ModuleTitle: lecture.ModuleTitle, Title: lecture.Title, ContentID: lecture.ContentID, LearningSeq: lecture.LearningSeq, Week: lectureWeekNumber(lecture)}
}
