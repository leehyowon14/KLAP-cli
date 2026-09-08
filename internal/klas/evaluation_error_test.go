package klas

import (
	"context"
	"testing"
)

func TestEvaluationSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").EvaluationTerm(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").EvaluationStudent(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").EvaluationCourses(context.Background(), EvaluationTerm{}); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").EvaluationForm(context.Background(), EvaluationTerm{}, EvaluationCourse{}); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
