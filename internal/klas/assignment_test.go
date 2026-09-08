package klas

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestAssignmentJSONOmitsRawPayload(t *testing.T) {
	dueAt := time.Date(2026, 6, 1, 23, 59, 0, 0, time.UTC)
	assignment := Assignment{
		OrdSeq: "7",
		Title:  "정규화 제목",
		DueAt:  &dueAt,
		Raw: assignmentListItem{
			Title: "raw-secret-title",
		},
	}
	payload, err := json.Marshal(assignment)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret-title")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("Assignment JSON contains Raw payload: %s", payload)
	}
	var roundTrip Assignment
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.OrdSeq != assignment.OrdSeq || roundTrip.Title != assignment.Title || roundTrip.DueAt == nil || !roundTrip.DueAt.Equal(dueAt) {
		t.Fatalf("Assignment round trip = %+v", roundTrip)
	}
}

func TestAssignmentListItemAcceptsNumericIDs(t *testing.T) {
	var item assignmentListItem
	err := json.Unmarshal([]byte(`{
		"ordseq": 7,
		"weeklyseq": 15,
		"weeklysubseq": 1,
		"title": "과제명",
		"expiredate": "2026-06-17 23:59:59",
		"submityn": "N"
	}`), &item)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.OrdSeq.String() != "7" || item.WeeklySeq.String() != "15" || item.WeeklySubSeq.String() != "1" {
		t.Fatalf("numeric ids were not preserved: %+v", item)
	}
}
