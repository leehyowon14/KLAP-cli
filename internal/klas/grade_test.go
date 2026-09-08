package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestGradeReportJSONOmitsRawPayload(t *testing.T) {
	report := GradeReport{
		Summary: GradeSummary{
			EarnedCredits: 18,
			GPA:           "4.0",
			Raw:           gradeSummaryItem{GPA: 1.23},
		},
		Terms: []GradeTerm{{
			Year:  "2026",
			Hakgi: "1",
			Label: "2026-1",
			Raw:   gradeTermItem{HakgiOrder: "raw-secret-term"},
			Courses: []GradeCourse{{
				Name:       "정규화 과목",
				CourseCode: "COURSE-1",
				Grade:      "A0",
				Raw:        gradeCourseItem{Name: "raw-secret-course"},
			}},
		}},
	}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("GradeReport JSON contains Raw payload: %s", payload)
	}
	var roundTrip GradeReport
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.Summary.EarnedCredits != report.Summary.EarnedCredits || roundTrip.Summary.GPA != report.Summary.GPA || len(roundTrip.Terms) != 1 || roundTrip.Terms[0].Label != report.Terms[0].Label || len(roundTrip.Terms[0].Courses) != 1 || roundTrip.Terms[0].Courses[0].Grade != "A0" {
		t.Fatalf("GradeReport round trip = %+v", roundTrip)
	}
}

func TestRankJSONOmitsRawPayload(t *testing.T) {
	rank := Rank{
		Year:      "2026",
		Hakgi:     "1",
		TermValue: "2026,1",
		GPA:       "4.0",
		ClassRank: "3",
		ClassSize: "50",
		Raw:       rankItem{Warning: "raw-secret-rank"},
	}
	payload, err := json.Marshal(rank)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("Rank JSON contains Raw payload: %s", payload)
	}
	var roundTrip Rank
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.TermValue != rank.TermValue || roundTrip.GPA != rank.GPA || roundTrip.ClassRank != rank.ClassRank || roundTrip.ClassSize != rank.ClassSize {
		t.Fatalf("Rank round trip = %+v", roundTrip)
	}
}
