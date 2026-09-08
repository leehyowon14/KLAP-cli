package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTimetableEntryJSONOmitsRawPayload(t *testing.T) {
	entry := TimetableEntry{
		SubjectID:   "subject-1",
		SubjectName: "정규화 과목",
		Weekday:     2,
		Period:      3,
		Room:        "R101",
		Raw:         map[string]any{"memberName": "raw-secret-professor"},
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("TimetableEntry JSON contains Raw payload: %s", payload)
	}
	var roundTrip TimetableEntry
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.SubjectID != entry.SubjectID || roundTrip.SubjectName != entry.SubjectName || roundTrip.Weekday != entry.Weekday || roundTrip.Period != entry.Period || roundTrip.Room != entry.Room {
		t.Fatalf("TimetableEntry round trip = %+v", roundTrip)
	}
}

func TestParseTimetableEntries(t *testing.T) {
	entries := parseTimetableEntries([]timetableRow{
		{
			"wtTime":        "1",
			"wtHasSchedule": "Y",
			"wtSpan_2":      "2",
			"wtSubj_2":      "SUBJ001",
			"wtSubjNm_2":    "자료구조",
			"wtLocHname_2":  "새빛관 101",
			"wtProfNm_2":    "홍길동",
		},
		{
			"wtTime":        float64(9),
			"wtHasSchedule": "Y",
			"wtSubj_3":      "SUBJ002",
			"wtSubjNm_3":    "온라인강의",
			"wtSpan_3":      "bad",
		},
		{
			"wtTime":        "2",
			"wtHasSchedule": "N",
			"wtSubjNm_1":    "빈 교시",
		},
	})

	if len(entries) != 2 {
		t.Fatalf("parseTimetableEntries() len = %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].Weekday != 2 || entries[0].Period != 1 || entries[0].Span != 2 {
		t.Fatalf("regular entry was not parsed correctly: %+v", entries[0])
	}
	if entries[0].SubjectName != "자료구조" || entries[0].Room != "새빛관 101" || entries[0].Online {
		t.Fatalf("regular entry fields were not parsed correctly: %+v", entries[0])
	}
	if entries[1].Weekday != 3 || entries[1].Period != 9 || entries[1].Span != 1 || !entries[1].Online {
		t.Fatalf("online entry was not parsed correctly: %+v", entries[1])
	}
}
