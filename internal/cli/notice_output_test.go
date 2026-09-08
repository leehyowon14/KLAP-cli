package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"testing"
)

func TestRunnerNoticeOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printNoticeRows(nil)
	r.printNoticeRows([]app.NoticeRow{{ID: "id", CourseName: "course", Notice: klas.Notice{Title: "title"}}})
	want := "강의 공지가 없습니다\n  id | 작성일 확인 필요 | course |  | title\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
