package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerAssignmentOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printAssignmentRows(nil)
	r.printAssignmentRows([]app.AssignmentRow{{ID: "id", CourseName: "course", Assignment: app.Assignment{Title: "title"}}})
	want := "과제가 없습니다\nid | 마감 확인 필요 | 미제출 | course | title\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
