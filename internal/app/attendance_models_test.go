package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestAttendanceModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/std/ads/admst/KwAttendStdGwakmokList.do":
			body = `[{"thisYear":"2026","hakgi":"1","openMajorCode":"0000","openGrade":"3","openGwamokNo":"1234","bunbanNo":"01","gwamokKname":" course ","memberName":" professor ","codeName1":"type","hakjumNum":3,"sisuNum":3,"currentNum":20,"yoil":"월"}]`
		case "/std/ads/admst/KwAttendStdAttendList.do":
			body = `[{"weeklyseq":2,"attendancediv1":"AT","attendancedate1":"2026-09-01","attendancediv2":"AB","attendancedate2":"2026-09-02"}]`
		case "/std/cps/atnlc/CdpAtendInfo.do":
			body = `[{"cdpDate":"2026-09-01","cdpSeq":7,"title":" seminar ","memberName":" speaker ","cnt":1}]`
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	courses, err := client.AttendanceCourses(context.Background(), "2026,1")
	if err != nil || len(courses) != 1 {
		t.Fatalf("courses=%+v error=%v", courses, err)
	}
	course := attendanceCourseModel(courses[0])
	if course.Name != "course" || course.Professor != "professor" || course.Credits != "3" || course.CurrentNum != "20" || course.SubjectID == "" {
		t.Fatalf("course=%+v", course)
	}
	assertModelJSONParity(t, courses[0], course)
	sessions, err := client.AttendanceSessions(context.Background(), "2026,1", courses[0])
	if err != nil {
		t.Fatal(err)
	}
	models := attendanceSessionModels(sessions)
	if len(models) != 1 || models[0].Week != "2" || len(models[0].Slots) != 2 || models[0].Slots[0].Mark != "O" || models[0].Slots[1].Mark != "X" {
		t.Fatalf("sessions=%+v", models)
	}
	assertModelJSONParity(t, sessions[0], models[0])
	for index, slot := range sessions[0].Slots {
		assertModelJSONParity(t, slot, models[0].Slots[index])
	}
	report, err := client.CdpAttendance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	model := cdpAttendanceReportModel(report)
	if model.TotalCount != "1" || len(model.Rows) != 1 || model.Rows[0].Title != "seminar" || model.Rows[0].Speaker != "speaker" {
		t.Fatalf("report=%+v", model)
	}
	assertModelJSONParity(t, report, model)
	assertModelJSONParity(t, report.Rows[0], model.Rows[0])
	assertModelJSONParity(t, klas.AttendanceSession{}, attendanceSessionModel(klas.AttendanceSession{}))
	assertModelJSONParity(t, klas.CdpAttendanceReport{}, cdpAttendanceReportModel(klas.CdpAttendanceReport{}))
	if attendanceSessionModels(nil) != nil || attendanceSlotModels(nil) != nil || cdpAttendanceModels(nil) != nil {
		t.Fatal("nil slice changed")
	}
	if attendanceSessionModels([]klas.AttendanceSession{}) == nil || attendanceSlotModels([]klas.AttendanceSlot{}) == nil || cdpAttendanceModels([]klas.CdpAttendance{}) == nil {
		t.Fatal("empty slice changed")
	}
}
