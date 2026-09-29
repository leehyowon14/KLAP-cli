package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestNoticeModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{}`
		switch {
		case strings.HasSuffix(req.URL.Path, "/BoardStdList.do"):
			body = `{"list":[{"boardNo":42,"masterNo":7,"title":" title ","userNm":" author ","registDt":"2026-09-01 09:00:00","topAt":"Y","readCnt":3,"fileCnt":2}],"page":{"totalPages":1}}`
		case strings.HasSuffix(req.URL.Path, "/BoardStdView.do"):
			body = `{"board":{"boardNo":42,"masterNo":7,"title":" title ","content":"<p>body</p>","userNm":" author ","registDt":"2026-09-01 09:00:00","topAt":"Y","readCnt":3,"atchFileId":99}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := client.Notices(context.Background(), "2026,1", Course{Value: "subject"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v error=%v", rows, err)
	}
	row := noticeModel(rows[0])
	if row.BoardNo != "42" || row.MasterNo != "7" || row.Title != "title" || row.Author != "author" || row.Registered == nil || !row.Top || row.ReadCount != "3" || row.FileCount != "2" {
		t.Fatalf("row=%+v", row)
	}
	assertModelJSONParity(t, rows[0], row)
	detail, err := client.NoticeDetail(context.Background(), "2026,1", Course{Value: "subject"}, "42", "7")
	if err != nil {
		t.Fatal(err)
	}
	model := noticeDetailModel(detail)
	if model.ContentHTML != "<p>body</p>" || model.ContentText != "body" || model.Attachment != "99" {
		t.Fatalf("detail=%+v", model)
	}
	assertModelJSONParity(t, detail, model)
	assertModelJSONParity(t, klas.Notice{}, noticeModel(klas.Notice{}))
	assertModelJSONParity(t, klas.NoticeDetail{}, noticeDetailModel(klas.NoticeDetail{}))
}
