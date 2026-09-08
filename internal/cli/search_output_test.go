package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerSearchOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printSearch(app.SearchResult{Query: "absent"})
	want := "검색: absent\n검색 결과가 없습니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
