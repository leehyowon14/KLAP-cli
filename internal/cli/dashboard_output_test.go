package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerDashboardOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printDashboard(app.DashboardResult{})
	want := "KLAP Dashboard |  ()\n\n과제\n  예정된 미제출 과제가 없습니다\n\n온라인 강의\n  수강할 온라인 강의/학습활동이 없습니다\n\n공지\n  최근 강의 공지가 없습니다\n\n출석\n  출석 현황이 없습니다\n\n수업평가\n  수업평가 기간이 아닙니다. 중간/기말 차수는 평가 기간에만 표시됩니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
