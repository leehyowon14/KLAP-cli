package klas

import (
	"encoding/json"
	"testing"
)

func TestLooksLikeLoginHTML(t *testing.T) {
	cases := []struct {
		name string
		body []byte
		want bool
	}{
		{name: "html", body: []byte("<html><body>login</body></html>"), want: true},
		{name: "login form path", body: []byte("<script>location='/usr/cmn/login/LoginForm.do'</script>"), want: true},
		{name: "json", body: []byte(`{"loginRequired":true}`), want: false},
		{name: "empty", body: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeLoginHTML(tc.body); got != tc.want {
				t.Fatalf("looksLikeLoginHTML() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFirstFieldError(t *testing.T) {
	got := firstFieldError([]fieldError{{Message: ""}, {Message: "개인번호 또는 비밀번호가 일치하지 않습니다."}}, "fallback")
	if got != "개인번호 또는 비밀번호가 일치하지 않습니다." {
		t.Fatalf("firstFieldError() = %q", got)
	}

	got = firstFieldError(nil, "fallback")
	if got != "fallback" {
		t.Fatalf("firstFieldError() fallback = %q", got)
	}
}

func TestAssignmentListItemAcceptsNumericIDs(t *testing.T) {
	var item assignmentListItem
	err := json.Unmarshal([]byte(`{
		"ordseq": 7,
		"weeklyseq": 15,
		"weeklysubseq": 1,
		"title": "과제명",
		"expiredate": "2026-06-17 23:59:59",
		"submityn": "N"
	}`), &item)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.OrdSeq.String() != "7" || item.WeeklySeq.String() != "15" || item.WeeklySubSeq.String() != "1" {
		t.Fatalf("numeric ids were not preserved: %+v", item)
	}
}

func TestHTMLToTextKeepsBlockBreaks(t *testing.T) {
	input := `<p>과제 설명</p><p><a href="https://example.com">https://example.com</a></p><p>제출 내용</p><ol><li>GitHub repository 주소</li><li>youtube 링크</li></ol>`

	got := htmlToText(input)
	want := "과제 설명\n\nhttps://example.com\n\n제출 내용\n\n- GitHub repository 주소\n- youtube 링크"
	if got != want {
		t.Fatalf("htmlToText() = %q, want %q", got, want)
	}
}

func TestHTMLToTextKeepsNumberedListCompact(t *testing.T) {
	input := `<p>제출 내용</p><p>1. GitHub repository 주소</p><p>2. youtube 링크</p>`

	got := htmlToText(input)
	want := "제출 내용\n\n1. GitHub repository 주소\n2. youtube 링크"
	if got != want {
		t.Fatalf("htmlToText() = %q, want %q", got, want)
	}
}
