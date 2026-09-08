package tui

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
	"time"
)

func TestNoticeContentGroupsMarksPinnedNotices(t *testing.T) {
	registered := time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)
	groups := noticeContentGroups([]app.NoticeRow{{
		CourseName: "컴퓨터그래픽스",
		Notice: app.Notice{
			Title:      "중요 공지",
			Top:        true,
			Registered: &registered,
		},
	}}, 96)

	if len(groups) != 1 || len(groups[0].lines) != 1 {
		t.Fatalf("noticeContentGroups() = %+v", groups)
	}
	if !strings.Contains(groups[0].lines[0], "Pinned") || !strings.Contains(groups[0].lines[0], "중요 공지") {
		t.Fatalf("pinned notice line = %q", groups[0].lines[0])
	}
	if strings.Index(groups[0].lines[0], "Pinned") < strings.Index(groups[0].lines[0], "중요 공지") {
		t.Fatalf("pinned badge should be rendered after title: %q", groups[0].lines[0])
	}
}

func TestDetailScrollClampsAtEdges(t *testing.T) {
	content := strings.Repeat("본문 줄\n", 40)
	m := model{
		active:       screenNoticeDetail,
		width:        80,
		height:       14,
		noticeDetail: app.NoticeDetailResult{ID: "1", CourseName: "강의", DetailURL: "https://klas.kw.ac.kr", Detail: app.NoticeDetail{Title: "공지", ContentText: content}},
	}
	m.moveDetailCursor(1)
	if m.detailCursor != 1 {
		t.Fatalf("detailCursor after first down = %d", m.detailCursor)
	}
	for i := 0; i < 100; i++ {
		m.moveDetailCursor(1)
	}
	bottom := m.detailCursor
	if bottom <= 1 {
		t.Fatalf("detailCursor bottom = %d", bottom)
	}
	for i := 0; i < 100; i++ {
		m.moveDetailCursor(-1)
	}
	if m.detailCursor != 0 {
		t.Fatalf("detailCursor after up clamp = %d", m.detailCursor)
	}
}
