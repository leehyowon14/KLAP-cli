package reminder

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAssignmentJSONIncludesLegacyIDsForBridgeMigration(t *testing.T) {
	payload, err := json.Marshal(Assignment{
		ID:        "assignment:v1:stable",
		LegacyIDs: []string{"2:7"},
		TermValue: "2026,1",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(payload), `"legacyIds":["2:7"]`) {
		t.Fatalf("Assignment JSON = %s", payload)
	}
	if !strings.Contains(string(payload), `"termValue":"2026,1"`) {
		t.Fatalf("Assignment JSON = %s", payload)
	}
}

func TestAssignmentJSONOmitsEmptyLegacyIDs(t *testing.T) {
	payload, err := json.Marshal(Assignment{ID: "assignment:v1:stable"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "legacyIds") {
		t.Fatalf("Assignment JSON = %s", payload)
	}
}
