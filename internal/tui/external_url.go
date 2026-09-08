package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"os/exec"
	"runtime"
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
					if err := openExternalURL(result.URL); err != nil {
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
		if err := openExternalURL(url); err != nil {
			return statusMsg{status: "KLAS 원문 열기 실패", err: err}
		}
		return statusMsg{}
	}
}

func openExternalURL(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("열 URL이 없습니다")
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("KLAS URL 열기 실패: %w", err)
	}
	return nil
}
