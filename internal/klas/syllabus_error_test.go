package klas

import (
	"context"
	"testing"
)

func TestSyllabusSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").SyllabusList(context.Background(), "2026,1", "", ""); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").SyllabusBySubjectID(context.Background(), "subject"); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
