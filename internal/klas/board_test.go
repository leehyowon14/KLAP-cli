package klas

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBoardMaterialContextAndPagination(t *testing.T) {
	calls := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "LctrumHomeStdInfo.do"):
			fmt.Fprint(w, `{}`)
		case strings.HasSuffix(r.URL.Path, "BoardStdList.do"):
			fmt.Fprint(w, `{"list":[{"boardNo":1,"masterNo":2,"title":"자료"}],"page":{"totalPages":2}}`)
		default:
			fmt.Fprint(w, `{"board":{"boardNo":1,"masterNo":2,"title":"자료","content":"<p>본문</p>"}}`)
		}
	}))
	defer srv.Close()
	c, _ := NewClient()
	c.baseURL, _ = url.Parse(srv.URL)
	rows, e := c.BoardPosts(context.Background(), "material", "2026,2", Course{Value: "course"})
	if e != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, e)
	}
	if len(calls) != 3 || !strings.HasSuffix(calls[0], "LctrumHomeStdInfo.do") || !strings.Contains(calls[1], "6972896bfe72408eb72926780e85d041") {
		t.Fatal(calls)
	}
	d, e := c.BoardPost(context.Background(), "material", "2026,2", Course{Value: "course"}, "1", "2")
	if e != nil || d.ContentText != "본문" {
		t.Fatalf("detail=%v err=%v", d, e)
	}
	n := len(calls)
	if _, e = c.BoardPosts(context.Background(), "invalid", "", Course{}); e == nil || len(calls) != n {
		t.Fatal("invalid board made request")
	}
}
