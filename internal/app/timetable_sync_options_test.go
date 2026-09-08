package app

import (
	"reflect"
	"testing"
)

func TestTimetableSyncOptionsKeepsDecisionsOutsideQuery(t *testing.T) {
	queryType := reflect.TypeFor[TimetableOptions]()
	if _, exists := queryType.FieldByName("SyncDecisions"); exists {
		t.Fatal("query accepts sync decisions")
	}
	command := TimetableSyncOptions{Decisions: map[string]SyncDecision{"key": SyncDecisionKeep}}
	if command.Decisions["key"] != SyncDecisionKeep {
		t.Fatal("decision lost")
	}
}
