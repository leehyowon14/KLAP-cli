package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kw-klap/klap-cli/internal/app"
)

type lectureSelectionModel struct {
	rows     []app.LectureRow
	selected map[string]bool
	cursor   int
	width    int
	height   int
	done     bool
	canceled bool
}

func RunLectureSelection(ctx context.Context, rows []app.LectureRow) ([]string, error) {
	_ = ctx
	if len(rows) == 0 {
		return nil, nil
	}
	model := lectureSelectionModel{
		rows:     rows,
		selected: make(map[string]bool, len(rows)),
	}
	for _, row := range rows {
		if lectureDownloadable(row) {
			model.selected[row.ID] = true
		}
	}
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}
	result, ok := finalModel.(lectureSelectionModel)
	if !ok || result.canceled {
		return nil, context.Canceled
	}
	ids := make([]string, 0)
	for _, row := range result.rows {
		if result.selected[row.ID] {
			ids = append(ids, row.ID)
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("선택된 강의가 없습니다")
	}
	return ids, nil
}

func (m lectureSelectionModel) Init() tea.Cmd {
	return nil
}

func (m lectureSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		switch {
		case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
			m.canceled = true
			return m, tea.Quit
		case key == "up" || keyMatches(key, "k", "ㅏ"):
			if m.cursor > 0 {
				m.cursor--
			}
		case key == "down" || keyMatches(key, "j", "ㅓ"):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case key == " ":
			m.toggleCurrent()
		case keyMatches(key, "a", "ㅁ"):
			m.toggleAll()
		case key == "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m lectureSelectionModel) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("KLAP"))
	b.WriteString(mutedStyle.Render(" select"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 96)))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("LECTURES"))
	b.WriteString("\n")
	visibleRows := maxInt(5, m.height-8)
	if m.height <= 0 {
		visibleRows = 18
	}
	if visibleRows > len(m.rows) {
		visibleRows = len(m.rows)
	}
	start := m.cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(m.rows) {
		start = maxInt(0, len(m.rows)-visibleRows)
	}
	end := start + visibleRows
	for index := start; index < end; index++ {
		row := m.rows[index]
		selected := m.selected[row.ID]
		marker := "  "
		if index == m.cursor {
			marker = "› "
		}
		check := "[ ]"
		if selected {
			check = "[x]"
		}
		if !lectureDownloadable(row) {
			check = "[-]"
		}
		label := truncateText(lectureDownloadLabel(row), maxInt(24, minInt(62, width-16)))
		line := fmt.Sprintf("%s%s  %s", marker, check, label)
		if index == m.cursor {
			line = menuSelectedStyle.Render(line)
		} else if !lectureDownloadable(row) {
			line = mutedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < len(m.rows) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(m.rows))))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("space 선택  |  a 전체  |  enter 다운로드  |  q 취소"))
	return appStyle.Render(b.String())
}

func (m *lectureSelectionModel) toggleCurrent() {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	row := m.rows[m.cursor]
	if !lectureDownloadable(row) {
		return
	}
	m.selected[row.ID] = !m.selected[row.ID]
}

func (m *lectureSelectionModel) toggleAll() {
	allSelected := true
	for _, row := range m.rows {
		if lectureDownloadable(row) && !m.selected[row.ID] {
			allSelected = false
			break
		}
	}
	for _, row := range m.rows {
		if lectureDownloadable(row) {
			m.selected[row.ID] = !allSelected
		}
	}
}

func lectureDownloadable(row app.LectureRow) bool {
	return strings.TrimSpace(row.Lecture.ContentID) != ""
}
