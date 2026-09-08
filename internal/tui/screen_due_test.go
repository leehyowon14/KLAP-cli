package tui

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
	"time"
)

func TestDuePageLinesSplitByKind(t *testing.T) {
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	result := app.DueResult{Items: []app.DueItem{
		{Kind: "과제", CourseName: "컴퓨터그래픽스", Title: "과제1", DueAt: dueAt},
		{Kind: "온라인 강의", CourseName: "오픈소스", Title: "HuggingFace", DueAt: dueAt},
		{Kind: "학사일정", Title: "종강", DueAt: dueAt},
	}}

	summary := strings.Join(duePageLines(result, 0, 96), "\n")
	if !strings.Contains(summary, "OVERVIEW") || !strings.Contains(summary, "FOCUS") || !strings.Contains(summary, "3 items") || !strings.Contains(summary, "HuggingFace") {
		t.Fatalf("summary lines = %q", summary)
	}
	assignments := strings.Join(duePageLines(result, 1, 96), "\n")
	if !strings.Contains(assignments, "과제1") || strings.Contains(assignments, "HuggingFace") {
		t.Fatalf("assignment due lines = %q", assignments)
	}
	academic := strings.Join(duePageLines(result, 3, 96), "\n")
	if !strings.Contains(academic, "종강") || strings.Contains(academic, "과제1") {
		t.Fatalf("academic due lines = %q", academic)
	}
}
