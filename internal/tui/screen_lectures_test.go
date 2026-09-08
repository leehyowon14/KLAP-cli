package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

func TestLectureContentGroupsAlignsTitleColumn(t *testing.T) {
	groups := lectureContentGroups([]app.LectureRow{
		{
			CourseName: "Gen-AI",
			Lecture: app.Lecture{
				Progress:    "100",
				ContentID:   "a",
				ModuleTitle: "Basics of Python I",
				Title:       "Python Basics I",
			},
		},
		{
			CourseName: "Gen-AI",
			Lecture: app.Lecture{
				Progress:    "100",
				ContentID:   "b",
				ModuleTitle: "Basics of Python II / Goal of Data Science and Data Storytelling",
				Title:       "Python_Basics_II",
			},
		},
	}, 80)

	if len(groups) != 1 || len(groups[0].lines) != 2 {
		t.Fatalf("lectureContentGroups() = %+v", groups)
	}
	firstTitleAt := strings.Index(groups[0].lines[0], "Python Basics I")
	secondTitleAt := strings.Index(groups[0].lines[1], "Python_Basics_II")
	if firstTitleAt < 0 || secondTitleAt < 0 {
		t.Fatalf("title not found: %q / %q", groups[0].lines[0], groups[0].lines[1])
	}
	firstTitleWidth := lipgloss.Width(groups[0].lines[0][:firstTitleAt])
	secondTitleWidth := lipgloss.Width(groups[0].lines[1][:secondTitleAt])
	if firstTitleWidth != secondTitleWidth {
		t.Fatalf("title columns not aligned: %q / %q", groups[0].lines[0], groups[0].lines[1])
	}
	if strings.Contains(groups[0].lines[1], "\n") {
		t.Fatalf("lecture line contains newline: %q", groups[0].lines[1])
	}
}

func TestLectureCompletedDetectsPercentAndMinuteProgress(t *testing.T) {
	if !lectureCompleted(app.Lecture{ContentID: "content", Progress: "100"}) {
		t.Fatal("content lecture with 100% progress should be completed")
	}
	if lectureCompleted(app.Lecture{ContentID: "content", Progress: "99"}) {
		t.Fatal("content lecture below 100% should not be completed")
	}
	if !lectureCompleted(app.Lecture{AchievedTime: "10", RequiredTime: "10"}) {
		t.Fatal("minute based lecture with achieved >= required should be completed")
	}
	if lectureCompleted(app.Lecture{AchievedTime: "9", RequiredTime: "10"}) {
		t.Fatal("minute based lecture below required time should not be completed")
	}
}
