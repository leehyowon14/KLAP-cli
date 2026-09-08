package klas

import (
	"context"
	"testing"
)

func TestNoticeSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").Notices(context.Background(), "2026,1", Course{}); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	for _, body := range []string{"{", "{}", `{"board":{}}`} {
		if _, err := schemaResponseClient(body).NoticeDetail(context.Background(), "2026,1", Course{}, "1", "1"); !IsErrorKind(err, ErrorSchema) {
			t.Fatalf("body=%s err=%v", body, err)
		}
	}
}
