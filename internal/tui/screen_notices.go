package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

type noticeScreenModel struct {
	noticeRows   []app.NoticeRow
	noticeDetail app.NoticeDetailResult
	pager        coursePager
	detailCursor int
}

type noticeScreenService interface {
	NoticeList(context.Context, app.NoticeListOptions) ([]app.NoticeRow, error)
	NoticeDetail(context.Context, string, app.UserOption) (app.NoticeDetailResult, error)
}

func (m *noticeScreenModel) Loaded(rows []app.NoticeRow) { m.noticeRows = rows }
func (m *noticeScreenModel) DetailLoaded(result app.NoticeDetailResult) {
	m.noticeDetail = result
	m.detailCursor = 0
}
func (m *noticeScreenModel) resetDetailCursor() { m.detailCursor = 0 }

func (m noticeScreenModel) groups(width int) []contentCourseGroup {
	return noticeContentGroups(m.noticeRows, maxInt(24, minInt(92, width-12)))
}
func (m noticeScreenModel) selectedRow(width int) (app.NoticeRow, bool) {
	group := m.pager.currentGroup(m.groups(width))
	return selectedRowByCourse(m.noticeRows, group.name, m.pager.contentCursor, func(row app.NoticeRow) string { return row.CourseName })
}
func (m model) selectedNoticeRow() (app.NoticeRow, bool) {
	return m.notices.selectedRow(m.width)
}

func (m *noticeScreenModel) Update(msg tea.Msg, width, height int, detail bool, ctx context.Context, service noticeScreenService) (childAction, bool) {
	if !detail && m.pager.Update(msg, m.groups(width)) {
		return childAction{}, true
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	if detail {
		switch {
		case key.String() == "up":
			m.moveDetailCursor(-1, width, height)
		case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
			m.moveDetailCursor(1, width, height)
		default:
			return childAction{}, false
		}
		return childAction{}, true
	}
	if key.String() != "enter" {
		return childAction{}, false
	}
	row, ok := m.selectedRow(width)
	if !ok {
		return childAction{setStatus: true, status: "선택된 공지가 없습니다"}, true
	}
	return childAction{navigate: true, target: screenNoticeDetail, cmd: loadNoticeDetail(ctx, service, row.ID)}, true
}

func loadNoticeDetail(ctx context.Context, service noticeScreenService, id string) tea.Cmd {
	return func() tea.Msg {
		result, err := service.NoticeDetail(ctx, id, app.UserOption{})
		return detailMsg{screen: screenNoticeDetail, notice: result, err: err}
	}
}
func (m *noticeScreenModel) moveDetailCursor(delta, width, height int) {
	lines := noticeDetailLines(m.noticeDetail, width)
	maxCursor := maxInt(0, len(lines)-visibleBodyRows(height, 0))
	m.detailCursor = clampInt(m.detailCursor+delta, 0, maxCursor)
}
func (m noticeScreenModel) View(width, height int, detail bool, status string) string {
	if !detail {
		return m.pager.View(m.groups(width), height, screenNotices)
	}
	lines := noticeDetailLines(m.noticeDetail, width)
	if len(lines) == 0 {
		return emptyStyle.Render("상세 정보가 없습니다") + "\n"
	}
	rendered := renderWindowedLines(lines, m.detailCursor, visibleBodyRows(height, 0))
	if status != "" {
		rendered += "\n" + footerStyle.Render(status) + "\n"
	}
	return rendered
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

func (m *noticeScreenModel) Reset() { *m = noticeScreenModel{} }

func loadNotices(ctx context.Context, service noticeScreenService, refresh, prefetch bool) tea.Cmd {
	return func() tea.Msg {
		rows, err := service.NoticeList(ctx, app.NoticeListOptions{Refresh: refresh})
		return loadMsg{screen: screenNotices, prefetch: prefetch, notices: rows, err: err}
	}
}
