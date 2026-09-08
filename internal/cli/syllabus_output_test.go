package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerSyllabusOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printSyllabus(app.SyllabusResult{})
	want := " ()\n과목: \n학정번호: 확인 필요\n과목ID: \n\n평가: 출석 0 / 학습 0 / 중간 0 / 기말 0 / 과제 0 / 퀴즈 0 / 기타 0\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
