package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerRoomOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printRoomFree(app.RoomQueryResult{})
	r.printRoomBusy(app.RoomQueryResult{})
	r.printRoomAvailable(app.RoomAvailableResult{})
	r.printRoomIndex(app.RoomIndexResult{})
	want := "매칭되는 강의실이 없습니다\n매칭되는 강의실이 없습니다\n빈 강의실 | 학기 미지정 | 0 교시 미지정 비어있음\n조건에 맞는 빈 강의실이 없습니다\n강의실 인덱스 | 학기 미지정 | 0개\n조회된 강의실이 없습니다\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
