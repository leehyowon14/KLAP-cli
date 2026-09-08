package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

func (m lectureScreenModel) selectedRow(width int) (app.LectureRow, bool) {
	group := m.pager.currentGroup(m.groups(width))
	return selectedRowByCourse(m.lectureRows, group.name, m.pager.contentCursor, func(row app.LectureRow) string {
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

type lectureScreenModel struct {
	lectureRows []app.LectureRow
	pager       coursePager
}

func (m *lectureScreenModel) Loaded(rows []app.LectureRow) { m.lectureRows = rows }
func (m *lectureScreenModel) Reset()                       { *m = lectureScreenModel{} }
func (m lectureScreenModel) groups(width int) []contentCourseGroup {
	return lectureContentGroups(m.lectureRows, maxInt(24, minInt(92, width-12)))
}
func (m lectureScreenModel) View(width, height int) string {
	return m.pager.View(m.groups(width), height, screenLectures)
}
func (m *lectureScreenModel) Update(msg tea.Msg, width int, loading bool) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	if keyMatches(key.String(), "d", "ㅇ") {
		return childAction{navigate: true, target: screenDownloadSelect}, true
	}
	if loading {
		return childAction{}, false
	}
	if m.pager.Update(msg, m.groups(width)) {
		return childAction{}, true
	}
	if keyMatches(key.String(), "a", "ㅁ") {
		return childAction{navigate: true, target: screenAttendConfirm}, true
	}
	if keyMatches(key.String(), "m", "ㅡ") {
		return childAction{navigate: true, target: screenAttendSelect}, true
	}
	return childAction{}, false
}

type lectureScreenService interface {
	LectureList(context.Context, app.LectureListOptions) ([]app.LectureRow, error)
}

func loadLectures(ctx context.Context, service lectureScreenService, refresh, prefetch bool) tea.Cmd {
	return func() tea.Msg {
		rows, err := service.LectureList(ctx, app.LectureListOptions{Refresh: refresh})
		return loadMsg{screen: screenLectures, prefetch: prefetch, lectures: rows, err: err}
	}
}
func (m model) selectedLectureRow() (app.LectureRow, bool) { return m.lectures.selectedRow(m.width) }

func (m *lectureScreenModel) ResetListPosition() { m.pager = coursePager{} }
