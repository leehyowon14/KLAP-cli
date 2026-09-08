package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerEvaluationOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printEvaluationList(app.EvaluationListResult{})
	r.printEvaluationSubmitResult(app.EvaluationSubmitResult{})
	want := "수업평가\n수업평가 기간이 아닙니다\n수업평가 기간이 아닙니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
