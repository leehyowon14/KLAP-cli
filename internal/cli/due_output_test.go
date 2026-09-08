package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerDueOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printDue(app.DueResult{})
	want := "데드라인: 0001-01-01 ~ 0001-01-01\n예정된 데드라인이 없습니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
