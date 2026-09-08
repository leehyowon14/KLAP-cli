package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerAttendanceOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printAttendanceList(app.AttendanceListResult{})
	r.printCdpAttendance(app.CdpAttendanceResult{})
	want := " ()\n출석 현황이 없습니다\nCDP 출석내역\n총 출석: 0회\nCDP 출석내역이 없습니다\n* 출석내역은 출석 후 약 일주일 후에 반영됩니다.\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
