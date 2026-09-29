package klas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestBoardAttachmentDownload(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		content, typ string
		size         int64
		fail         bool
	}{
		{"ok", 200, "%PDF-test", "application/pdf", 9, false},
		{"empty", 200, "", "application/octet-stream", 0, false},
		{"truncated", 200, "abc", "application/pdf", 20, true},
		{"login", 401, "", "text/html", 0, true},
		{"error-page", 200, "<html>error</html>", "text/html", 0, true},
		{"missing", 404, "", "text/plain", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, e := r.Cookie("SESSION"); e != nil {
					t.Error("missing session")
				}
				w.Header().Set("Content-Type", tc.typ)
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.content)
			}))
			defer srv.Close()
			c, _ := NewClient()
			c.baseURL, _ = url.Parse(srv.URL)
			c.SetSession(Session{Cookies: map[string]string{"SESSION": "test"}})
			dir := t.TempDir()
			p, e := c.DownloadBoardAttachment(context.Background(), BoardAttachment{Name: "../../資料.pdf", Size: tc.size, download: "/common/file/DownloadFile/test/1"}, dir)
			if (e != nil) != tc.fail {
				t.Fatalf("err=%v", e)
			}
			if tc.fail {
				items, _ := os.ReadDir(dir)
				if len(items) != 0 {
					t.Fatal("partial file remains")
				}
			} else {
				b, e := os.ReadFile(p)
				if e != nil || string(b) != tc.content || filepath.Base(p) != "資料.pdf" {
					t.Fatal(p, e)
				}
			}
		})
	}
}
func TestAttachmentRejectsForeignURLAndCancellation(t *testing.T) {
	c, _ := NewClient()
	if _, e := c.DownloadBoardAttachment(context.Background(), BoardAttachment{download: "https://example.com/common/file/DownloadFile/x/1"}, t.TempDir()); e == nil {
		t.Fatal("foreign URL accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := c.DownloadBoardAttachment(ctx, BoardAttachment{download: "/common/file/DownloadFile/x/1"}, t.TempDir()); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
func TestAttachmentList(t *testing.T) {
	for _, body := range []string{`[]`, `[{"fileSn":1,"fileName":"자료.pdf","fileSize":9,"download":"/common/file/DownloadFile/x/1"}]`, `[{"fileSn":1}]`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c, _ := NewClient()
		c.baseURL, _ = url.Parse(srv.URL)
		rows, e := c.BoardAttachments(context.Background(), "id")
		if body == `[{"fileSn":1}]` {
			if e == nil {
				t.Fatal("incomplete accepted")
			}
		} else if e != nil {
			t.Fatal(e)
		} else if len(rows) > 0 && rows[0].FileSN != "1" {
			t.Fatal(rows)
		}
		srv.Close()
	}
}

func TestReadBoardAttachmentWithoutFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		fmt.Fprint(w, "%PDF-test")
	}))
	defer srv.Close()
	c, _ := NewClient()
	c.baseURL, _ = url.Parse(srv.URL)
	data, err := c.ReadBoardAttachment(context.Background(), BoardAttachment{Name: "test.pdf", Size: 9, download: "/common/file/DownloadFile/x/1"})
	if err != nil || string(data) != "%PDF-test" {
		t.Fatal(err, string(data))
	}
	if _, err = c.ReadBoardAttachment(context.Background(), BoardAttachment{Size: MaxAttachmentMemoryBytes + 1}); err == nil {
		t.Fatal("oversized preview accepted")
	}
	if _, err = c.ReadBoardAttachment(context.Background(), BoardAttachment{Size: 10, download: "/common/file/DownloadFile/x/1"}); err == nil {
		t.Fatal("truncated memory response accepted")
	}
}
