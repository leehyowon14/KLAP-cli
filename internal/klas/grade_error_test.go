package klas

import (
	"context"
	"testing"
)

func TestGradeSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").Grades(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").Ranks(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
