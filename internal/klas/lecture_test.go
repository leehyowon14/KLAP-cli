package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLectureJSONOmitsRawPayload(t *testing.T) {
	lecture := Lecture{
		ContentID:    "content-1",
		LearningSeq:  "42",
		Title:        "정규화 강의",
		Progress:     "30",
		RequiredTime: "60",
		Raw: lectureListItem{
			GroupCode: "raw-secret-group",
			SubjectID: "raw-secret-subject",
		},
	}
	payload, err := json.Marshal(lecture)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("Lecture JSON contains Raw payload: %s", payload)
	}
	var roundTrip Lecture
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.ContentID != lecture.ContentID || roundTrip.LearningSeq != lecture.LearningSeq || roundTrip.Title != lecture.Title || roundTrip.Progress != lecture.Progress || roundTrip.RequiredTime != lecture.RequiredTime {
		t.Fatalf("Lecture round trip = %+v", roundTrip)
	}
	if _, err := lectureLearningStatusPayload(roundTrip, "Y"); err == nil {
		t.Fatal("cached Lecture unexpectedly retained action payload")
	}
}

func TestLectureKeyPatternSupportsQuoteVariants(t *testing.T) {
	cases := []string{
		`"lecKey": 'abc-123'`,
		`'lecKey': "abc-123"`,
	}

	for _, body := range cases {
		match := lectureKeyPattern.FindStringSubmatch(body)
		if len(match) < 2 || match[1] != "abc-123" {
			t.Fatalf("lectureKeyPattern did not parse %q: %+v", body, match)
		}
	}
}

func TestParseLectureProgress(t *testing.T) {
	progress, err := parseLectureProgress([]byte(`{"data":{"totalTime":"10","ptime":"50","prog":20}}`))
	if err != nil {
		t.Fatalf("parseLectureProgress() error = %v", err)
	}
	if progress.TotalTime != "10" || progress.PTime != "50" || progress.Progress != 20 || progress.Completed {
		t.Fatalf("parseLectureProgress() = %+v", progress)
	}

	progress, err = parseLectureProgress([]byte(`{"totalTime":"50","ptime":"50","prog":100}`))
	if err != nil {
		t.Fatalf("parseLectureProgress() root error = %v", err)
	}
	if !progress.Completed {
		t.Fatalf("parseLectureProgress() completed = %+v", progress)
	}
}

func TestLectureViewerForm(t *testing.T) {
	form, err := lectureViewerForm(Lecture{Raw: lectureListItem{
		GroupCode: "G",
		SubjectID: "S",
		Year:      "2026",
		Hakgi:     "1",
		Bunban:    "01",
		Module:    flexibleString("M"),
		OID:       "OID",
		PTime:     flexibleString("50"),
		WeekNo:    flexibleString("3"),
		WeeklySeq: flexibleString("2"),
		TotalTime: flexibleString("0"),
		Progress:  flexibleString("0"),
		Lesson:    flexibleString("001"),
		IsPreview: "N",
	}})
	if err != nil {
		t.Fatalf("lectureViewerForm() error = %v", err)
	}
	if form.Get("weeklyseq") != "3" || form.Get("weeklysubseq") != "2" || form.Get("profYN") != "Y" {
		t.Fatalf("lectureViewerForm() = %v", form)
	}
}

func TestLectureLearningStatusPayload(t *testing.T) {
	payload, err := lectureLearningStatusPayload(Lecture{Raw: lectureListItem{
		GroupCode:   "G",
		SubjectID:   "S",
		Year:        "2026",
		Hakgi:       "1",
		Bunban:      "01",
		LearningSeq: flexibleString("15"),
	}}, "Y")
	if err != nil {
		t.Fatalf("lectureLearningStatusPayload() error = %v", err)
	}
	if payload["lrnSn"] != "15" || payload["lrnStatus"] != "Y" {
		t.Fatalf("lectureLearningStatusPayload() = %v", payload)
	}
}

func TestLectureListItemLearningTimeFields(t *testing.T) {
	var item lectureListItem
	if err := json.Unmarshal([]byte(`{"rcognTime":10,"achivTime":0,"learnTime":"0","totRcognTime":"60","totAchivTime":"50"}`), &item); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.RcognTime.String() != "10" || item.AchivTime.String() != "0" || item.TotRcognTime.String() != "60" {
		t.Fatalf("learning time fields were not preserved: %+v", item)
	}
}

func TestLectureListTimesDoNotTreatDurationAsAttendance(t *testing.T) {
	cases := []struct{ raw, required, achieved string }{
		{`{"ptime":"50","rcognTime":"60","totalTime":"60","achivTime":"0","prog":0}`, "60", "0"},
		{`{"ptime":"41","rcognTime":"41","totalTime":"41","achivTime":"41","prog":100}`, "41", "41"},
		{`{"ptime":"50","totalTime":"60"}`, "50", ""},
		{`{"rcognTime":60,"achivTime":0,"learnTime":30}`, "60", "0"},
		{`{"rcognTime":null,"totRcognTime":70,"achivTime":null,"totAchivTime":12}`, "70", "12"},
		{`{}`, "", ""},
	}
	for _, tc := range cases {
		var item lectureListItem
		if err := json.Unmarshal([]byte(tc.raw), &item); err != nil {
			t.Fatal(err)
		}
		required, achieved := lectureListTimes(item)
		if required != tc.required || achieved != tc.achieved {
			t.Fatalf("got %q/%q for %s", achieved, required, tc.raw)
		}
	}
}
func TestLectureFirstAttendanceDates(t *testing.T) {
	for _, raw := range []string{`{}`, `{"firstEdu":null,"firstEnd":null}`, `{"firstEdu":"","firstEnd":"invalid"}`} {
		var item lectureListItem
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			t.Fatal(err)
		}
		if parseKlasDateTime(item.FirstEdu) != nil || parseKlasDateTime(item.FirstEnd) != nil {
			t.Fatal("unknown dates must stay nil")
		}
	}
	date := parseKlasDateTime("2026-09-01 15:01")
	if date == nil || date.Format("2006-01-02T15:04-07:00") != "2026-09-01T15:01+09:00" {
		t.Fatal("KST timestamp")
	}
}
