package app

import (
	"reflect"
	"testing"
)

func TestAssignmentSyncOptionsKeepsDecisionsOutsideQuery(t *testing.T) {
	queryType := reflect.TypeFor[AssignmentListOptions]()
	if _, exists := queryType.FieldByName("SyncDecisions"); exists {
		t.Fatal("query accepts sync decisions")
	}
	command := AssignmentSyncOptions{Decisions: map[string]SyncDecision{"key": SyncDecisionKeep}}
	if command.Decisions["key"] != SyncDecisionKeep {
		t.Fatal("decision lost")
	}
}
