package download

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"path/filepath"
	"strings"
	"testing"
)

func TestLectureFilenameSanitizesPathComponents(t *testing.T) {
	got := NewClient(nil).VideoPath("root", "오픈소스/실습", app.LectureFileName{
		ModuleTitle: "1주차: 소개",
		Week:        1,
		Title:       "Git? GitHub* 시작",
	}, "https://media.example.com/video.mp4?token=1", 1)
	want := filepath.Join("root", "오픈소스_실습", "video", "1-1. Git_ GitHub_ 시작.mp4")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureFilenameUsesKlasWeekSequence(t *testing.T) {
	got := NewClient(nil).VideoPath("root", "컴퓨터그래픽스", app.LectureFileName{
		Week:        14,
		ModuleTitle: "보강",
		Title:       "기말/정리",
	}, "https://media.example.com/final.mov", 2)
	want := filepath.Join("root", "컴퓨터그래픽스", "video", "14-2. 기말_정리.mov")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestFilenameFallbackAndSanitization(t *testing.T) {
	got := NewClient(nil).VideoPath("root", "../", app.LectureFileName{}, "invalid", 0)
	if got != filepath.Join("root", "_", "video", "lecture.mp4") {
		t.Fatalf("fallback=%q", got)
	}
	if got := sanitizePathComponent(" . "); got != "" {
		t.Fatalf("dots=%q", got)
	}
	if got := sanitizePathComponent(strings.Repeat("가", 121)); len([]rune(got)) != 120 {
		t.Fatalf("length=%d", len([]rune(got)))
	}
}
