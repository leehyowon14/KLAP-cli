package app

import (
	"reflect"
	"testing"
)

func TestAcademicSyncOptionsKeepsDecisionsOutsideQuery(t *testing.T) {
	queryType := reflect.TypeFor[AcademicListOptions]()
	if _, exists := queryType.FieldByName("SyncDecisions"); exists {
		t.Fatal("query accepts sync decisions")
	}
	command := AcademicSyncOptions{Decisions: map[string]SyncDecision{"key": SyncDecisionKeep}}
	if command.Decisions["key"] != SyncDecisionKeep {
		t.Fatal("decision lost")
	}
}
