package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerSubjectOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printSubjectSearch(app.SubjectSearchResult{})
	want := " ()\n검색된 과목이 없습니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
