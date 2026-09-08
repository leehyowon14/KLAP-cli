package klas

import (
	"context"
	"testing"
)

func TestCourseSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").Courses(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
