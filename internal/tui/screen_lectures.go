package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

func (m model) selectedLectureRow() (app.LectureRow, bool) {
	group := m.currentContentGroup(m.width)
	return selectedRowByCourse(m.lectureRows, group.name, m.activePager().contentCursor, func(row app.LectureRow) string {
		return row.CourseName
	})
}

func lectureContentGroups(rows []app.LectureRow, width int) []contentCourseGroup {
	groups := make([]contentCourseGroup, 0)
	indexByName := make(map[string]int)
	progressWidth := 9
	moduleWidth := maxInt(18, minInt(34, width/3))
	for _, row := range rows {
		groupIndex := contentGroupIndex(&groups, indexByName, row.CourseName)
		progressText := lectureProgress(row.Lecture)
		progress := lectureProgressStyle(row.Lecture).Render(fixedColumn(progressText, progressWidth))
		module := emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요")
		module = fixedColumn(module, moduleWidth)
		titleWidth := maxInt(12, width-progressWidth-moduleWidth-6)
		title := truncateText(row.Lecture.Title, titleWidth)
		groups[groupIndex].lines = append(groups[groupIndex].lines, fmt.Sprintf("%s  %s  %s",
			progress,
			module,
			title,
		))
	}
	return groups
}

func lectureProgressStyle(lecture app.Lecture) lipgloss.Style {
	if lectureCompleted(lecture) {
		return successTextStyle
	}
	return warnTextStyle
}

func lectureCompleted(lecture app.Lecture) bool {
	if strings.TrimSpace(lecture.ContentID) != "" {
		progress, err := strconv.ParseFloat(strings.TrimSpace(lecture.Progress), 64)
		return err == nil && progress >= 100
	}
	achieved, achievedErr := strconv.ParseFloat(strings.TrimSpace(lecture.AchievedTime), 64)
	required, requiredErr := strconv.ParseFloat(strings.TrimSpace(lecture.RequiredTime), 64)
	return achievedErr == nil && requiredErr == nil && required > 0 && achieved >= required
}

func formatLectures(rows []app.LectureRow) string {
	if len(rows) == 0 {
		return emptyStyle.Render("온라인 강의가 없습니다") + "\n"
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			mutedStyle.Render(lectureProgress(row.Lecture)),
			row.CourseName,
			emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요"),
			row.Lecture.Title,
		))
	}
	return b.String()
}

func lectureProgress(lecture app.Lecture) string {
	if strings.TrimSpace(lecture.ContentID) != "" {
		progress := strings.TrimSpace(lecture.Progress)
		if progress == "" {
			return "진도 확인 필요"
		}
		return progress + "%"
	}
	achieved := strings.TrimSpace(lecture.AchievedTime)
	if achieved == "" {
		achieved = "0"
	}
	required := strings.TrimSpace(lecture.RequiredTime)
	if required == "" {
		required = "?"
	}
	return achieved + "/" + required + "분"
}
