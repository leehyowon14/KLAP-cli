package klas

import (
	"testing"
)

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
