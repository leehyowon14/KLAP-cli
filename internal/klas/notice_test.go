package klas

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestNoticeJSONOmitsRawPayload(t *testing.T) {
	registered := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	notice := Notice{
		BoardNo:    "board",
		MasterNo:   "master",
		Title:      "정규화 제목",
		Registered: &registered,
		Raw: noticeItem{
			Content:  "raw-secret-content",
			UserName: "raw-secret-author",
		},
	}
	payload, err := json.Marshal(notice)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("Notice JSON contains Raw payload: %s", payload)
	}
	var roundTrip Notice
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.BoardNo != notice.BoardNo || roundTrip.MasterNo != notice.MasterNo || roundTrip.Title != notice.Title || roundTrip.Registered == nil || !roundTrip.Registered.Equal(registered) {
		t.Fatalf("Notice round trip = %+v", roundTrip)
	}
}

func TestNoticeItemAcceptsNumericIDs(t *testing.T) {
	var item noticeItem
	err := json.Unmarshal([]byte(`{
		"boardNo": 1161280,
		"masterNo": 1000000,
		"title": "공지",
		"readCnt": 12,
		"fileCnt": 1
	}`), &item)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.BoardNo.String() != "1161280" || item.MasterNo.String() != "1000000" {
		t.Fatalf("numeric notice ids were not preserved: %+v", item)
	}
	if item.ReadCount.String() != "12" || item.FileCount.String() != "1" {
		t.Fatalf("numeric notice counts were not preserved: %+v", item)
	}
}

func TestParseNoticeListResponseReadsPaginationAndPinned(t *testing.T) {
	notices, totalPages, err := parseNoticeListResponse([]byte(`{
		"list": [
			{
				"boardNo": 1161280,
				"masterNo": 1000000,
				"title": "고정 공지",
				"topAt": "Y",
				"readCnt": 12,
				"fileCnt": 1,
				"userNm": "작성자",
				"registDt": "2026-05-29T02:20:00.000+09:00"
			}
		],
		"page": {
			"currentPage": 0,
			"pageSize": 10,
			"totalElements": 12,
			"totalPages": 2
		}
	}`))
	if err != nil {
		t.Fatalf("parseNoticeListResponse() error = %v", err)
	}
	if totalPages != 2 {
		t.Fatalf("totalPages = %d, want 2", totalPages)
	}
	if len(notices) != 1 || !notices[0].Top || notices[0].Title != "고정 공지" {
		t.Fatalf("notices = %+v", notices)
	}
}

func TestParseNoticeListResponseDefaultsMissingPage(t *testing.T) {
	_, totalPages, err := parseNoticeListResponse([]byte(`{"list":[]}`))
	if err != nil {
		t.Fatalf("parseNoticeListResponse() error = %v", err)
	}
	if totalPages != 1 {
		t.Fatalf("totalPages = %d, want 1", totalPages)
	}
}
