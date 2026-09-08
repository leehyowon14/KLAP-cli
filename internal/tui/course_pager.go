package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func selectedRowByCourse[T any](rows []T, courseName string, cursor int, course func(T) string) (T, bool) {
	var zero T
	if strings.TrimSpace(courseName) == "" || cursor < 0 {
		return zero, false
	}
	seen := 0
	for _, row := range rows {
		name := strings.TrimSpace(course(row))
		if name == "" {
			name = "과목 확인 필요"
		}
		if name != courseName {
			continue
		}
		if seen == cursor {
			return row, true
		}
		seen++
	}
	return zero, false
}

type contentCourseGroup struct {
	name  string
	lines []string
}

func (m model) isCoursePagedScreen() bool {
	switch m.active {
	case screenAssignments, screenNotices, screenLectures:
		return true
	default:
		return false
	}
}

func (m model) contentGroups(width int) []contentCourseGroup {
	lineWidth := maxInt(24, minInt(92, width-12))
	switch m.active {
	case screenAssignments:
		return assignmentContentGroups(m.assignmentRows, lineWidth)
	case screenNotices:
		return noticeContentGroups(m.noticeRows, lineWidth)
	case screenLectures:
		return lectureContentGroups(m.lectureRows, lineWidth)
	default:
		return nil
	}
}

func (m model) currentContentGroup(width int) contentCourseGroup {
	return m.pager.currentGroup(m.contentGroups(width))
}

func (m coursePager) currentGroup(groups []contentCourseGroup) contentCourseGroup {
	if len(groups) == 0 {
		return contentCourseGroup{}
	}
	index := m.contentCourse
	if index < 0 {
		index = 0
	}
	if index >= len(groups) {
		index = len(groups) - 1
	}
	return groups[index]
}

func (m *model) moveContentCourse(delta int) { m.pager.moveCourse(m.contentGroups(m.width), delta) }

func (m *coursePager) moveCourse(groups []contentCourseGroup, delta int) {
	if len(groups) == 0 {
		m.contentCourse = 0
		m.contentCursor = 0
		return
	}
	m.contentCourse += delta
	if m.contentCourse < 0 {
		m.contentCourse = len(groups) - 1
	}
	if m.contentCourse >= len(groups) {
		m.contentCourse = 0
	}
	m.contentCursor = 0
}

func (m *model) moveContentCursor(delta int) { m.pager.moveCursor(m.contentGroups(m.width), delta) }

func (m *coursePager) moveCursor(groups []contentCourseGroup, delta int) {
	group := m.currentGroup(groups)
	if len(group.lines) == 0 {
		m.contentCursor = 0
		return
	}
	m.contentCursor += delta
	if m.contentCursor < 0 {
		m.contentCursor = len(group.lines) - 1
	}
	if m.contentCursor >= len(group.lines) {
		m.contentCursor = 0
	}
}

func contentGroupIndex(groups *[]contentCourseGroup, indexByName map[string]int, courseName string) int {
	name := strings.TrimSpace(courseName)
	if name == "" {
		name = "과목 확인 필요"
	}
	index, ok := indexByName[name]
	if !ok {
		index = len(*groups)
		indexByName[name] = index
		*groups = append(*groups, contentCourseGroup{name: name})
	}
	return index
}

func (m model) renderCoursePagedPanel(width int) string {
	return m.pager.View(m.contentGroups(width), m.height, m.active)
}

func (m coursePager) View(groups []contentCourseGroup, height int, active screen) string {
	if len(groups) == 0 {
		switch active {
		case screenAssignments:
			return emptyStyle.Render("과제가 없습니다") + "\n"
		case screenNotices:
			return emptyStyle.Render("강의 공지가 없습니다") + "\n"
		case screenLectures:
			return emptyStyle.Render("온라인 강의가 없습니다") + "\n"
		default:
			return emptyStyle.Render("표시할 내용이 없습니다") + "\n"
		}
	}

	group := m.currentGroup(groups)
	page := m.contentCourse + 1
	if page < 1 {
		page = 1
	}
	if page > len(groups) {
		page = len(groups)
	}

	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%d/%d  %s", page, len(groups), group.name)))
	b.WriteString("\n\n")

	visibleRows := maxInt(5, height-10)
	if height <= 0 {
		visibleRows = 16
	}
	if visibleRows > len(group.lines) {
		visibleRows = len(group.lines)
	}

	cursor := m.contentCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(group.lines) {
		cursor = len(group.lines) - 1
	}

	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(group.lines) {
		start = maxInt(0, len(group.lines)-visibleRows)
	}
	end := start + visibleRows

	for index := start; index < end; index++ {
		marker := "  "
		if index == cursor {
			marker = "› "
		}
		line := marker + group.lines[index]
		if index == cursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < len(group.lines) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(group.lines))))
		b.WriteString("\n")
	}
	return b.String()
}

type coursePager struct {
	contentCourse int
	contentCursor int
}

func (m *coursePager) Update(msg tea.Msg, groups []contentCourseGroup) bool {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	switch {
	case key.String() == "up":
		m.moveCursor(groups, -1)
	case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
		m.moveCursor(groups, 1)
	case key.String() == "left":
		m.moveCourse(groups, -1)
	case key.String() == "right":
		m.moveCourse(groups, 1)
	default:
		return false
	}
	return true
}
