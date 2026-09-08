package tui

import (
	"context"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"time"
)

type transcriptLanguage struct {
	label  string
	locale string
}

var transcriptLanguages = []transcriptLanguage{
	{label: "한국어", locale: "ko-KR"},
	{label: "English", locale: "en-US"},
	{label: "日本語", locale: "ja-JP"},
	{label: "中文", locale: "zh-CN"},
	{label: "Deutsch", locale: "de-DE"},
	{label: "Français", locale: "fr-FR"},
	{label: "Español", locale: "es-ES"},
}

type downloadRowsMsg struct {
	rows []app.LectureRow
	err  error
}

func (m model) loadDownloadRows() tea.Cmd {
	return func() tea.Msg {
		rows, err := m.service.LectureList(m.ctx, app.LectureListOptions{Refresh: true})
		return downloadRowsMsg{rows: rows, err: err}
	}
}

func (m model) updateDownloadSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenLectures
		m.loading = true
		m.err = nil
		m.markScreenLoading(screenLectures)
		return m, m.load(screenLectures, false)
	case m.loading:
		return m, nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		maxCursor := len(m.currentDownloadRows())
		if maxCursor <= 0 {
			m.downloadCursor = 0
		} else if m.downloadCursor <= 0 {
			m.downloadCursor = maxCursor
		} else {
			m.downloadCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		maxCursor := len(m.currentDownloadRows())
		if maxCursor <= 0 || m.downloadCursor >= maxCursor {
			m.downloadCursor = 0
		} else {
			m.downloadCursor++
		}
	case key == "left":
		if m.downloadCourse > 0 {
			m.downloadCourse--
			m.downloadCursor = 0
		}
	case key == "right":
		if m.downloadCourse < len(m.downloadGroups())-1 {
			m.downloadCourse++
			m.downloadCursor = 0
		}
	case key == " ":
		m.toggleDownloadCurrent()
	case keyMatches(key, "a", "ㅁ"):
		m.toggleDownloadAll()
	case key == "enter":
		if len(m.selectedDownloadIDs()) == 0 {
			m.err = errors.New("선택된 강의가 없습니다")
			return m, nil
		}
		m.err = nil
		m.active = screenDownloadConfirm
		m.downloadTranscribe = false
	}
	return m, nil
}

func (m model) updateDownloadConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenDownloadSelect
	case keyMatches(key, "y", "ㅛ"):
		m.downloadTranscribe = true
		m.active = screenDownloadLanguage
		m.downloadLanguage = 0
	case keyMatches(key, "n", "ㅜ"):
		m.downloadTranscribe = false
		return m.startDownloadProgress()
	case key == "left" || key == "right" || key == "tab":
		m.downloadTranscribe = !m.downloadTranscribe
	case key == "enter":
		if m.downloadTranscribe {
			m.active = screenDownloadLanguage
			m.downloadLanguage = 0
			return m, nil
		}
		return m.startDownloadProgress()
	}
	return m, nil
}

func (m model) updateDownloadLanguage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenDownloadConfirm
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if len(transcriptLanguages) == 0 {
			m.downloadLanguage = 0
		} else if m.downloadLanguage <= 0 {
			m.downloadLanguage = len(transcriptLanguages) - 1
		} else {
			m.downloadLanguage--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if len(transcriptLanguages) == 0 || m.downloadLanguage >= len(transcriptLanguages)-1 {
			m.downloadLanguage = 0
		} else {
			m.downloadLanguage++
		}
	case key == "enter":
		return m.startDownloadProgress()
	}
	return m, nil
}

func (m model) startDownloadProgress() (tea.Model, tea.Cmd) {
	settings, err := m.service.DownloadSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	transcriptSettings, err := m.service.TranscriptSettings()
	if err != nil {
		m.err = err
		return m, nil
	}
	selectedRows := m.selectedDownloadRows()
	runCtx, cancel := context.WithCancel(m.ctx)
	progress := lectureDownloadModel{
		ctx:    runCtx,
		cancel: cancel,
		request: LectureDownloadRequest{
			LectureIDs:            m.selectedDownloadIDs(),
			Concurrency:           settings.Concurrency,
			Transcribe:            m.downloadTranscribe,
			TranscriptLocale:      m.selectedTranscriptLocale(),
			TranscriptConcurrency: transcriptSettings.Concurrency,
		},
		updates:   make(chan tea.Msg, 64),
		items:     initialDownloadStatusLines(selectedRows),
		startedAt: time.Now(),
	}
	progress.pipeline = m.service.NewLectureDownloadRun(runCtx, progress.pipelineOptions())
	m.active = screenDownloadProgress
	m.err = nil
	m.downloadProgress = &progress
	return m, progress.Init()
}

func (m model) updateDownloadProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.downloadProgress == nil {
		m.active = screenLectures
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.downloadProgress.canceling {
			return m, nil
		}
		if key.String() == "ctrl+c" || keyMatches(key.String(), "q", "ㅂ") {
			return m, m.downloadProgress.cancelAndWait(true)
		}
		if keyMatches(key.String(), "h", "ㅗ") {
			return m, m.downloadProgress.cancelAndWait(false)
		}
		if m.downloadProgress.done && (key.String() == "esc" || keyMatches(key.String(), "b", "ㅠ")) {
			m.active = screenLectures
			m.loading = true
			m.downloadProgress = nil
			return m, m.load(screenLectures, true)
		}
		if key.String() == "esc" && !m.downloadProgress.done {
			return m, m.downloadProgress.cancelAndCleanup()
		}
	}
	updated, cmd := m.downloadProgress.Update(msg)
	progress, ok := updated.(lectureDownloadModel)
	if !ok {
		return m, cmd
	}
	m.downloadProgress = &progress
	return m, cmd
}

func (m *model) toggleDownloadCurrent() {
	rows := m.currentDownloadRows()
	if m.downloadCursor == 0 {
		m.toggleDownloadCourse()
		return
	}
	rowIndex := m.downloadCursor - 1
	if rowIndex < 0 || rowIndex >= len(rows) {
		return
	}
	row := rows[rowIndex]
	if !lectureDownloadable(row) {
		return
	}
	m.downloadSelected[row.ID] = !m.downloadSelected[row.ID]
}

func (m *model) toggleDownloadCourse() {
	rows := m.currentDownloadRows()
	allSelected := true
	for _, row := range rows {
		if lectureDownloadable(row) && !m.downloadSelected[row.ID] {
			allSelected = false
			break
		}
	}
	for _, row := range rows {
		if lectureDownloadable(row) {
			m.downloadSelected[row.ID] = !allSelected
		}
	}
}

func (m *model) toggleDownloadAll() {
	allSelected := true
	hasDownloadable := false
	for _, row := range m.downloadRows {
		if !lectureDownloadable(row) {
			continue
		}
		hasDownloadable = true
		if !m.downloadSelected[row.ID] {
			allSelected = false
			break
		}
	}
	if !hasDownloadable {
		return
	}
	for _, row := range m.downloadRows {
		if lectureDownloadable(row) {
			m.downloadSelected[row.ID] = !allSelected
		}
	}
}

func (m model) selectedDownloadIDs() []string {
	ids := make([]string, 0)
	for _, row := range m.downloadRows {
		if m.downloadSelected[row.ID] {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

func (m model) selectedDownloadRows() []app.LectureRow {
	rows := make([]app.LectureRow, 0)
	for _, row := range m.downloadRows {
		if m.downloadSelected[row.ID] {
			rows = append(rows, row)
		}
	}
	return rows
}

func (m model) selectedTranscriptLocale() string {
	if m.downloadLanguage < 0 || m.downloadLanguage >= len(transcriptLanguages) {
		return transcriptLanguages[0].locale
	}
	return transcriptLanguages[m.downloadLanguage].locale
}

type downloadCourseGroup struct {
	name string
	rows []app.LectureRow
}

func (m model) downloadGroups() []downloadCourseGroup {
	groups := make([]downloadCourseGroup, 0)
	indexByName := make(map[string]int)
	for _, row := range m.downloadRows {
		name := strings.TrimSpace(row.CourseName)
		if name == "" {
			name = "과목 확인 필요"
		}
		index, ok := indexByName[name]
		if !ok {
			index = len(groups)
			indexByName[name] = index
			groups = append(groups, downloadCourseGroup{name: name})
		}
		groups[index].rows = append(groups[index].rows, row)
	}
	return groups
}

func (m model) currentDownloadGroup() downloadCourseGroup {
	groups := m.downloadGroups()
	if len(groups) == 0 {
		return downloadCourseGroup{}
	}
	index := m.downloadCourse
	if index < 0 {
		index = 0
	}
	if index >= len(groups) {
		index = len(groups) - 1
	}
	return groups[index]
}

func (m model) currentDownloadRows() []app.LectureRow {
	return m.currentDownloadGroup().rows
}

func (m model) currentDownloadCourseSelected() bool {
	rows := m.currentDownloadRows()
	hasDownloadable := false
	for _, row := range rows {
		if !lectureDownloadable(row) {
			continue
		}
		hasDownloadable = true
		if !m.downloadSelected[row.ID] {
			return false
		}
	}
	return hasDownloadable
}

func (m model) renderDownloadSelectView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Download"))
	b.WriteString("\n")
	if m.loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 강의 목록을 불러오는 중입니다\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	if len(m.downloadRows) == 0 {
		b.WriteString(emptyStyle.Render("다운로드할 온라인 강의가 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	groups := m.downloadGroups()
	group := m.currentDownloadGroup()
	rows := group.rows
	page := m.downloadCourse + 1
	if page < 1 {
		page = 1
	}
	if page > len(groups) {
		page = len(groups)
	}
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%d/%d  %s", page, len(groups), group.name)))
	b.WriteString("\n\n")

	visibleRows := maxInt(5, m.height-10)
	if m.height <= 0 {
		visibleRows = 16
	}
	totalItems := len(rows) + 1
	if visibleRows > totalItems {
		visibleRows = totalItems
	}
	start := m.downloadCursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > totalItems {
		start = maxInt(0, totalItems-visibleRows)
	}
	end := start + visibleRows
	for index := start; index < end; index++ {
		marker := "  "
		if index == m.downloadCursor {
			marker = "› "
		}
		if index == 0 {
			check := "[ ]"
			if m.currentDownloadCourseSelected() {
				check = "[x]"
			}
			line := fmt.Sprintf("%s%s  모두 선택", marker, check)
			if index == m.downloadCursor {
				line = menuSelectedStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		row := rows[index-1]
		check := "[ ]"
		if m.downloadSelected[row.ID] {
			check = "[x]"
		}
		if !lectureDownloadable(row) {
			check = "[-]"
		}
		line := fmt.Sprintf("%s%s  %s", marker, check, truncateText(lectureDownloadLabel(row), maxInt(24, minInt(70, width-16))))
		if index == m.downloadCursor {
			line = menuSelectedStyle.Render(line)
		} else if !lectureDownloadable(row) {
			line = mutedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < totalItems {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, totalItems)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderHelpText("←/→ 과목  |  space 선택  |  a 전체 과목  |  enter 다음  |  b 뒤로  |  q 종료", width))
	return b.String()
}

func (m model) renderDownloadConfirmView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Transcript"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("선택한 강의 %d개를 다운로드한 뒤 전사할까요?", len(m.selectedDownloadIDs())))
	b.WriteString("\n\n")
	yes := mutedStyle.Render("  Yes")
	no := menuSelectedStyle.Render("› No")
	if m.downloadTranscribe {
		yes = menuSelectedStyle.Render("› Yes")
		no = mutedStyle.Render("  No")
	}
	b.WriteString(yes)
	b.WriteString("    ")
	b.WriteString(no)
	b.WriteString("\n\n")
	b.WriteString(renderHelpText("←/→ 선택  |  y/n  |  enter 다운로드  |  b 뒤로  |  q 종료", width))
	return b.String()
}

func (m model) renderDownloadLanguageView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Transcript Language"))
	b.WriteString("\n")
	b.WriteString("전사 주 언어를 선택하세요.")
	b.WriteString("\n\n")
	for index, language := range transcriptLanguages {
		marker := "  "
		if index == m.downloadLanguage {
			marker = "› "
		}
		line := fmt.Sprintf("%s%s  %s", marker, language.label, mutedStyle.Render(language.locale))
		if index == m.downloadLanguage {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderHelpText("↑↓ 선택  |  enter 다운로드  |  b 뒤로  |  q 종료", width))
	return b.String()
}
