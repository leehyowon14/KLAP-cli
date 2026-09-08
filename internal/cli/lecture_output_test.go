package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerLectureOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printLectureRows(nil)
	r.printLectureStatusRows(nil)
	r.printLectureDownloadAllResult(app.LectureDownloadAllResult{})
	r.printDownloadStatus(app.DownloadStatusResult{})
	r.printLectureAttendAllResult(app.LectureAttendAllResult{})
	want := "온라인 강의가 없습니다\n온라인 강의가 없습니다\n다운로드할 온라인 강의가 없습니다\n다운로드 폴더: \n파일 수: 0\n크기: 0 B\n다운로드된 파일이 없습니다\n수강할 온라인 강의가 없습니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
