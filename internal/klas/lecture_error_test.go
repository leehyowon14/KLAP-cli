package klas

import (
	"context"
	"testing"
)

func TestLectureSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").Lectures(context.Background(), "2026,1", Course{}); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}

func TestLectureProgressSchemaErrors(t *testing.T) {
	for _, body := range []string{"{", "{}", `{"prog":"invalid"}`} {
		if _, err := parseLectureProgress([]byte(body)); !IsErrorKind(err, ErrorSchema) {
			t.Fatalf("body=%s err=%v", body, err)
		}
	}
}
