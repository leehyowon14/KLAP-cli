package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

func (m model) openNoticeDetail() (tea.Model, tea.Cmd) {
	row, ok := m.selectedNoticeRow()
	if !ok {
		m.syncStatus = "선택된 공지가 없습니다"
		return m, nil
	}
	m.active = screenNoticeDetail
	m.detailBack = screenNotices
	m.loading = true
	m.err = nil
	m.syncStatus = ""
	return m, m.loadNoticeDetail(row.ID)
}

func (m model) loadNoticeDetail(id string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.service.NoticeDetail(m.ctx, id, app.UserOption{})
		return detailMsg{screen: screenNoticeDetail, notice: result, err: err}
	}
}

func (m model) selectedNoticeRow() (app.NoticeRow, bool) {
	group := m.currentContentGroup(m.width)
	return selectedRowByCourse(m.noticeRows, group.name, m.activePager().contentCursor, func(row app.NoticeRow) string {
		return row.CourseName
	})
}

func noticeContentGroups(rows []app.NoticeRow, width int) []contentCourseGroup {
	groups := make([]contentCourseGroup, 0)
	indexByName := make(map[string]int)
	for _, row := range rows {
		groupIndex := contentGroupIndex(&groups, indexByName, row.CourseName)
		date := mutedStyle.Render(formatTime(row.Notice.Registered))
		badge := ""
		titleWidth := maxInt(12, width-19)
		if row.Notice.Top {
			date = warnTextStyle.Render(formatTime(row.Notice.Registered))
			badge = " " + warnBadgeStyle.Render("Pinned")
			titleWidth = maxInt(12, width-28)
		}
		title := truncateText(row.Notice.Title, titleWidth)
		groups[groupIndex].lines = append(groups[groupIndex].lines, fmt.Sprintf("%s  %s%s",
			date,
			title,
			badge,
		))
	}
	return groups
}

func noticeDetailLines(result app.NoticeDetailResult, width int) []string {
	if strings.TrimSpace(result.ID) == "" {
		return nil
	}
	detail := result.Detail
	lines := []string{
		sectionStyle.Render(detail.Title),
		mutedStyle.Render(result.CourseName),
		"",
		"작성일  " + formatTime(detail.Registered),
	}
	if detail.Author != "" {
		lines = append(lines, "작성자  "+detail.Author)
	}
	if detail.Top {
		lines = append(lines, "중요  예")
	}
	if detail.ReadCount != "" {
		lines = append(lines, "조회수  "+detail.ReadCount)
	}
	if detail.Attachment != "" {
		lines = append(lines, "첨부 묶음  "+detail.Attachment)
	}
	lines = append(lines, "", mutedStyle.Render("KLAS  "+result.DetailURL))
	if strings.TrimSpace(detail.ContentText) != "" {
		lines = append(lines, "", sectionStyle.Render("본문"))
		lines = appendWrappedLines(lines, detail.ContentText, width)
	}
	return lines
}

func formatNotices(rows []app.NoticeRow) string {
	if len(rows) == 0 {
		return emptyStyle.Render("강의 공지가 없습니다") + "\n"
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("%s  %s  %s\n",
			mutedStyle.Render(formatTime(row.Notice.Registered)),
			row.CourseName,
			row.Notice.Title,
		))
	}
	return b.String()
}
