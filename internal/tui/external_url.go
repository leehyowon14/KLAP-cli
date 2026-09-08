package tui

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

func (m model) canOpenKlasURL() bool {
	switch m.active {
	case screenAssignments, screenNotices, screenLectures, screenAssignmentDetail, screenNoticeDetail:
		return true
	default:
		return false
	}
}

func (m model) openCurrentKlasURL() tea.Cmd {
	url := ""
	switch m.active {
	case screenAssignments:
		if row, ok := m.selectedAssignmentRow(); ok {
			url = row.DetailURL
		}
	case screenNotices:
		if row, ok := m.selectedNoticeRow(); ok {
			url = row.DetailURL
		}
	case screenLectures:
		if row, ok := m.selectedLectureRow(); ok {
			if strings.TrimSpace(row.Lecture.PlayURL) == "" {
				id := row.ID
				return func() tea.Msg {
					result, err := m.service.LectureOpenURL(m.ctx, id, app.UserOption{})
					if err != nil {
						return statusMsg{status: "KLAS 원문 열기 실패", err: err}
					}
					if err := m.openExternalURL(result.URL); err != nil {
						return statusMsg{status: "KLAS 원문 열기 실패", err: err}
					}
					return statusMsg{}
				}
			}
			url = row.Lecture.PlayURL
		}
	case screenAssignmentDetail:
		url = m.assignments.assignmentDetail.DetailURL
	case screenNoticeDetail:
		url = m.notices.noticeDetail.DetailURL
	}
	if strings.TrimSpace(url) == "" {
		return nil
	}
	return func() tea.Msg {
		if err := m.openExternalURL(url); err != nil {
			return statusMsg{status: "KLAS 원문 열기 실패", err: err}
		}
		return statusMsg{}
	}
}

func (m model) openExternalURL(target string) error {
	if m.opener == nil {
		return errors.New("TUI opener가 없습니다")
	}
	return m.opener(m.ctx, target, "KLAS URL")
}
