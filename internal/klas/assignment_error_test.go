package klas

import (
	"context"
	"testing"
)

func TestAssignmentSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").Assignments(context.Background(), "2026,1", Course{}); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	for _, body := range []string{"{", "{}"} {
		if _, err := schemaResponseClient(body).AssignmentDetail(context.Background(), "2026,1", Course{}, "1"); !IsErrorKind(err, ErrorSchema) {
			t.Fatalf("body=%s err=%v", body, err)
		}
	}
}
