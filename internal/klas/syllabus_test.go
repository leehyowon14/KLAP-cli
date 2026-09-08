package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSyllabusJSONOmitsRawContactPayload(t *testing.T) {
	syllabus := Syllabus{
		SubjectID:  "subject-1",
		KoreanName: "정규화 과목",
		Professor:  "정규화 교수",
		Purpose:    "정규화 목표",
		Times:      []SyllabusTime{{Weekday: "월", Periods: []int{1, 2}, Room: "R101"}},
		Raw: syllabusDataItem{
			Email:   "raw-secret@example.com",
			PhoneNo: "raw-secret-phone",
			TelNo:   "raw-secret-tel",
		},
		RawTimeResponse: []syllabusTimeItem{{Room: "raw-secret-room"}},
	}
	payload, err := json.Marshal(syllabus)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) || bytes.Contains(payload, []byte(`"RawTimeResponse"`)) {
		t.Fatalf("Syllabus JSON contains raw contact payload: %s", payload)
	}
	var roundTrip Syllabus
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.SubjectID != syllabus.SubjectID || roundTrip.KoreanName != syllabus.KoreanName || roundTrip.Professor != syllabus.Professor || roundTrip.Purpose != syllabus.Purpose || len(roundTrip.Times) != 1 || roundTrip.Times[0].Room != "R101" {
		t.Fatalf("Syllabus round trip = %+v", roundTrip)
	}
}

func TestSyllabusSubjectIDFromCourseCode(t *testing.T) {
	got, err := SyllabusSubjectIDFromCourseCode("2026,1", "I040-3-3951-01")
	if err != nil {
		t.Fatalf("SyllabusSubjectIDFromCourseCode() error = %v", err)
	}
	if got != "U202613951I040013" {
		t.Fatalf("SyllabusSubjectIDFromCourseCode() = %q", got)
	}
}

func TestSyllabusListItemSubjectID(t *testing.T) {
	item := SyllabusListItem{
		ThisYear:      "2026",
		Hakgi:         "1",
		OpenMajorCode: "I040",
		OpenGrade:     "3",
		OpenGwamokNo:  "3951",
		BunbanNo:      "01",
	}
	subjectID, err := item.SubjectID()
	if err != nil {
		t.Fatalf("SubjectID() error = %v", err)
	}
	if item.CourseCode() != "I040-3-3951-01" || subjectID != "U202613951I040013" {
		t.Fatalf("item ids = %q, %q", item.CourseCode(), subjectID)
	}
}

func TestBuildSyllabus(t *testing.T) {
	syllabus := buildSyllabus("U202613951I040013", syllabusDataItem{
		OpenMajorCode:  "I040",
		OpenGrade:      "3",
		OpenGwamokNo:   "3951",
		BunbanNo:       "01",
		KoreanName:     "컴퓨터그래픽스",
		EnglishName:    "Computer Graphics",
		Professor:      "김동준",
		CourseType:     "전선",
		Credits:        flexibleString("3"),
		CurrentNum:     flexibleString("42"),
		Face100Opt:     "Y",
		AttendanceRate: 10,
		MidtermRate:    25,
		Week1Lecture:   "오리엔테이션",
		Week10Lecture:  "Global Illumination",
		Week10Subs:     "동영상 보강",
	}, []syllabusTimeItem{
		{Weekday: "화", Time1: flexibleString("3"), Room: "새빛103"},
	})

	if syllabus.CourseCode != "I040-3-3951-01" || syllabus.Operation != "100%대면강의" {
		t.Fatalf("buildSyllabus() = %+v", syllabus)
	}
	if syllabus.CurrentNum != "42" {
		t.Fatalf("buildSyllabus() current num = %q", syllabus.CurrentNum)
	}
	if len(syllabus.Schedule) != 2 || syllabus.Schedule[1].Week != 10 || syllabus.Schedule[1].SubNote != "동영상 보강" {
		t.Fatalf("buildSyllabus() schedule = %+v", syllabus.Schedule)
	}
	if len(syllabus.Times) != 1 || syllabus.Times[0].Periods[0] != 3 || syllabus.Times[0].Room != "새빛103" {
		t.Fatalf("buildSyllabus() times = %+v", syllabus.Times)
	}
	if syllabus.Evaluation.Attendance != 10 || syllabus.Evaluation.Midterm != 25 {
		t.Fatalf("buildSyllabus() evaluation = %+v", syllabus.Evaluation)
	}
}
