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
