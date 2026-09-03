package tui

import (
	"strings"

	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func lectureDownloadable(row app.LectureRow) bool {
	return strings.TrimSpace(row.Lecture.ContentID) != ""
}
