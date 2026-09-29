package app

import (
	"context"
	"testing"
)

func TestBoardRejectsIncompleteContextBeforeSession(t *testing.T) {
	s := &Service{}
	for _, o := range []BoardOptions{{Kind: "other", TermValue: "2026,2", SubjectID: "id"}, {Kind: "notice", SubjectID: "id"}, {Kind: "material", TermValue: "2026,2"}} {
		if _, e := s.BoardList(context.Background(), o); e == nil {
			t.Fatal("list accepted invalid context")
		}
		if _, e := s.BoardDetail(context.Background(), o); e == nil {
			t.Fatal("detail accepted invalid context")
		}
		if _, e := s.BoardDownload(context.Background(), o, "1", t.TempDir()); e == nil {
			t.Fatal("download accepted invalid context")
		}
	}
}
