package klas

import (
	"context"
	"testing"
)

func TestTimetableSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").Timetable(context.Background(), "2026,1"); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
