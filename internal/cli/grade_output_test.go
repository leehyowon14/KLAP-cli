package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerGradeOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printGrade(app.GradeResult{})
	want := "성적\n신청 학점: 전체 0 / 전공 0 / 교양 0 / 기타 0\n취득 학점: 전체 0 / 전공 0 / 교양 0 / 기타 0\n평점: 학적부 기준 - / 성적증명서 기준 -\n\n성적 내역이 없습니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
