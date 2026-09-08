package klas

import (
	"context"
	"testing"
)

func TestAttendanceSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").AttendanceCourses(context.Background(), "2026,1"); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").AttendanceSessions(context.Background(), "2026,1", AttendanceCourse{}); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").CdpAttendance(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
