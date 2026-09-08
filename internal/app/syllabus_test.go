package app

import (
	"testing"
)

func TestLooksLikeSyllabusCourseCode(t *testing.T) {
	if !looksLikeSyllabusCourseCode("I040-3-3951-01") {
		t.Fatal("looksLikeSyllabusCourseCode() expected true")
	}
	if looksLikeSyllabusCourseCode("컴퓨터그래픽스") {
		t.Fatal("looksLikeSyllabusCourseCode() expected false")
	}
}
