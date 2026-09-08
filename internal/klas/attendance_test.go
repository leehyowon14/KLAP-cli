package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestAttendanceJSONOmitsRawPayload(t *testing.T) {
	report := struct {
		Course   AttendanceCourse
		Sessions []AttendanceSession
		CDP      CdpAttendance
	}{
		Course: AttendanceCourse{
			CourseCode: "COURSE-1",
			Name:       "정규화 과목",
			Professor:  "정규화 교수",
			Raw:        attendanceCourseItem{KoreanName: "raw-secret-course"},
		},
		Sessions: []AttendanceSession{{
			Week:  "1",
			Slots: []AttendanceSlot{{Index: 1, Status: "출석"}},
			Raw:   attendanceSessionItem{AttendanceDiv1: "raw-secret-session"},
		}},
		CDP: CdpAttendance{
			Date: "2026-06-01",
			Seq:  "1",
			Raw:  cdpAttendanceItem{Title: "raw-secret-cdp"},
		},
	}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("attendance JSON contains Raw payload: %s", payload)
	}
	var roundTrip struct {
		Course   AttendanceCourse
		Sessions []AttendanceSession
		CDP      CdpAttendance
	}
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.Course.CourseCode != report.Course.CourseCode || roundTrip.Course.Name != report.Course.Name || len(roundTrip.Sessions) != 1 || roundTrip.Sessions[0].Week != "1" || len(roundTrip.Sessions[0].Slots) != 1 || roundTrip.CDP.Date != report.CDP.Date {
		t.Fatalf("attendance round trip = %+v", roundTrip)
	}
}

func TestBuildAttendanceCourse(t *testing.T) {
	course := buildAttendanceCourse(attendanceCourseItem{
		ThisYear:      "2026",
		Hakgi:         "1",
		OpenMajorCode: "I040",
		OpenGrade:     "3",
		OpenGwamokNo:  "3951",
		BunbanNo:      "01",
		KoreanName:    " 컴퓨터그래픽스 ",
		Professor:     " 김동준 ",
		CourseType:    "전선",
		Credits:       flexibleString("3"),
		CreditHours:   flexibleString("3"),
		CurrentNum:    flexibleString("40"),
		Weekday:       "화3,목4",
	})
	if course.CourseCode != "I040-3-3951-01" || course.Name != "컴퓨터그래픽스" || course.Professor != "김동준" {
		t.Fatalf("buildAttendanceCourse() = %+v", course)
	}
	if course.SubjectID != "U202613951I040013" {
		t.Fatalf("SubjectID = %q", course.SubjectID)
	}
	if course.Credits != "3" || course.CreditHours != "3" || course.CurrentNum != "40" || course.Weekday != "화3,목4" {
		t.Fatalf("buildAttendanceCourse() details = %+v", course)
	}
}

func TestBuildAttendanceSession(t *testing.T) {
	session := buildAttendanceSession(attendanceSessionItem{
		WeeklySeq:       flexibleString("1"),
		AttendanceDiv1:  "AB",
		AttendanceDiv2:  "AT",
		AttendanceDate1: "20260306",
		AttendanceDate2: "20260306",
	})
	if session.Week != "1" || len(session.Slots) != 2 {
		t.Fatalf("buildAttendanceSession() = %+v", session)
	}
	if session.Slots[0].Mark != "X" || session.Slots[1].Mark != "O" {
		t.Fatalf("attendance marks = %+v", session.Slots)
	}
	if AttendanceStatusMark("LT") != "L" || AttendanceStatusMark("LE") != "R" || AttendanceStatusMark("OA") != "A" {
		t.Fatal("AttendanceStatusMark() unexpected mapping")
	}
}
