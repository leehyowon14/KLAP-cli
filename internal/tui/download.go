package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	bubblesprogress "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type LectureDownloadRequest struct {
	LectureIDs            []string
	Concurrency           int
	Transcribe            bool
	TranscriptLocale      string
	TranscriptConcurrency int
}

type lectureDownloadModel struct {
	ctx       context.Context
	cancel    context.CancelFunc
	pipeline  lectureDownloadRun
	request   LectureDownloadRequest
	updates   chan tea.Msg
	width     int
	height    int
	startedAt time.Time
	items     []downloadStatusLine
	cursor    int
	done      bool
	canceling bool
	err       error
}

type lectureDownloadRun interface {
	Run() (app.LectureDownloadPipelineResult, error)
	CancelAndCleanup() error
}

type lectureDownloadCleanupMsg struct {
	updates chan tea.Msg
	err     error
}
type lectureDownloadStoppedMsg struct{ updates chan tea.Msg }

type downloadStatusLine struct {
	id             string
	label          string
	status         string
	path           string
	downloadPath   string
	transcriptPath string
	bytes          int64
	total          int64
	percent        float64
	skipped        bool
	err            error
}

type lectureTranscriptProgressMsg struct {
	progress app.LectureTranscriptProgress
}

type lectureDownloadProgressMsg struct {
	progress app.LectureDownloadProgress
}

type lectureDownloadDoneMsg struct {
	all        app.LectureDownloadAllResult
	transcript app.LectureTranscriptResult
	err        error
}

func (m lectureDownloadModel) Init() tea.Cmd {
	return tea.Batch(m.runDownload(), waitLectureDownloadProgress(m.ctx, m.updates))
}

func (m lectureDownloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		switch msg.(type) {
		case lectureDownloadProgressMsg, lectureTranscriptProgressMsg, lectureDownloadDoneMsg:
			return m, nil
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || keyMatches(key, "q", "ㅂ") {
			m.canceling = true
			m.cancel()
			return m, nil
		}
		if key == "up" || keyMatches(key, "k", "ㅏ") {
			if len(m.items) == 0 {
				return m, nil
			}
			if m.cursor <= 0 {
				m.cursor = len(m.items) - 1
			} else {
				m.cursor--
			}
			return m, nil
		}
		if key == "down" || keyMatches(key, "j", "ㅓ") {
			if len(m.items) == 0 {
				return m, nil
			}
			if m.cursor >= len(m.items)-1 {
				m.cursor = 0
			} else {
				m.cursor++
			}
			return m, nil
		}
	case lectureDownloadProgressMsg:
		m.upsertStatusLine(msg.progress)
		return m, waitLectureDownloadProgress(m.ctx, m.updates)
	case lectureTranscriptProgressMsg:
		m.upsertTranscriptStatusLine(msg.progress)
		return m, waitLectureDownloadProgress(m.ctx, m.updates)
	case lectureDownloadDoneMsg:
		m.err = msg.err
		m.applyFinalResult(msg)
		m.applyTranscriptResult(msg.transcript)
		if !m.canceling {
			m.done = true
		}
		return m, nil
	}
	return m, nil
}

func (m lectureDownloadModel) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	contentWidth := tuiContentWidth(width)

	var b strings.Builder
	b.WriteString(renderHeaderTitle(contentWidth, "Download"))
	b.WriteString("\n")
	b.WriteString(renderRule(contentWidth))
	b.WriteString("\n\n")
	b.WriteString(m.renderProgressList(contentWidth))
	b.WriteString("\n")
	if m.done {
		b.WriteString(m.renderSummary())
		if m.err != nil {
			b.WriteString(errorStyle.Render(truncateText(m.err.Error(), contentWidth)))
			b.WriteString("\n")
		}
		b.WriteString(renderHelpText("↑↓ 이동  |  esc 뒤로  |  h 홈  |  q 종료", contentWidth))
		b.WriteString("\n")
	} else if m.canceling {
		b.WriteString(warnBadgeStyle.Render("CANCEL"))
		b.WriteString(" 다운로드를 중단하는 중입니다\n")
		b.WriteString(renderHelpText("esc 삭제/뒤로  |  h 홈  |  q 종료", contentWidth))
		b.WriteString("\n")
	} else {
		b.WriteString(renderHelpText("↑↓ 이동  |  esc 삭제/뒤로  |  h 홈  |  q 종료", contentWidth))
		b.WriteString("\n")
	}
	return appStyle.Render(b.String())
}

func (m lectureDownloadModel) pipelineOptions() app.LectureDownloadPipelineOptions {
	return app.LectureDownloadPipelineOptions{
		Download:   app.LectureDownloadAllOptions{Concurrency: m.request.Concurrency, LectureIDs: m.request.LectureIDs},
		Transcribe: m.request.Transcribe, TranscriptLocale: m.request.TranscriptLocale, TranscriptConcurrency: m.request.TranscriptConcurrency,
		OnEvent: func(event app.LecturePipelineEvent) {
			var message tea.Msg
			if event.Download != nil {
				message = lectureDownloadProgressMsg{progress: *event.Download}
			} else if event.Transcript != nil {
				message = lectureTranscriptProgressMsg{progress: *event.Transcript}
			} else {
				return
			}
			select {
			case m.updates <- message:
			default:
			}
		},
	}
}

func (m lectureDownloadModel) runDownload() tea.Cmd {
	return func() tea.Msg {
		result, err := m.pipeline.Run()
		// The terminal result follows all accepted progress in the same FIFO stream.
		select {
		case m.updates <- lectureDownloadDoneMsg{all: result.Download, transcript: result.Transcript, err: err}:
		case <-m.ctx.Done():
		}
		return nil
	}
}

func waitLectureDownloadProgress(ctx context.Context, updates chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		select {
		case message := <-updates:
			return message
		case <-ctx.Done():
			return nil
		}
	}
}

func (m *lectureDownloadModel) upsertStatusLine(progress app.LectureDownloadProgress) {
	label := lectureDownloadLabel(progress.Lecture)
	if label == "" {
		return
	}
	line := downloadStatusLine{
		id:      lectureDownloadKey(progress.Lecture),
		label:   label,
		status:  progress.Stage,
		path:    progress.Path,
		bytes:   progress.Bytes,
		total:   progress.TotalBytes,
		skipped: progress.Skipped,
		err:     progress.Err,
	}
	if strings.TrimSpace(progress.Path) != "" {
		line.downloadPath = progress.Path
	}
	for index := range m.items {
		if m.items[index].matches(line) {
			if preservesTranscriptStatus(m.items[index].status, line.status) {
				m.items[index].bytes = line.bytes
				m.items[index].total = line.total
				if strings.TrimSpace(line.downloadPath) != "" {
					m.items[index].downloadPath = line.downloadPath
				}
				return
			}
			if strings.TrimSpace(line.downloadPath) == "" {
				line.downloadPath = m.items[index].downloadPath
			}
			if strings.TrimSpace(line.transcriptPath) == "" {
				line.transcriptPath = m.items[index].transcriptPath
			}
			m.items[index] = line
			return
		}
	}
	m.items = append(m.items, line)
}

func (m *lectureDownloadModel) upsertTranscriptStatusLine(progress app.LectureTranscriptProgress) {
	label := lectureDownloadLabel(progress.Lecture)
	if label == "" {
		label = progress.InputPath
	}
	line := downloadStatusLine{
		id:      lectureDownloadKey(progress.Lecture),
		label:   label,
		status:  progress.Stage,
		path:    progress.OutputPath,
		percent: progress.Progress,
		err:     progress.Err,
	}
	if strings.TrimSpace(progress.OutputPath) != "" {
		line.transcriptPath = progress.OutputPath
	}
	for index := range m.items {
		if m.items[index].matches(line) {
			m.items[index].status = line.status
			m.items[index].path = line.path
			if strings.TrimSpace(line.transcriptPath) != "" {
				m.items[index].transcriptPath = line.transcriptPath
			}
			m.items[index].percent = line.percent
			m.items[index].err = line.err
			m.items[index].skipped = false
			return
		}
	}
	m.items = append(m.items, line)
}

func (m *lectureDownloadModel) applyFinalResult(msg lectureDownloadDoneMsg) {
	for _, item := range msg.all.Items {
		stage := "done"
		if item.Skipped {
			stage = "skip"
		}
		if item.Err != nil {
			stage = "error"
		}
		m.upsertStatusLine(app.LectureDownloadProgress{
			Lecture:    item.Lecture,
			Path:       item.Path,
			Stage:      stage,
			Bytes:      item.Bytes,
			TotalBytes: item.Bytes,
			Skipped:    item.Skipped,
			Err:        item.Err,
		})
	}
}

func (m *lectureDownloadModel) applyTranscriptResult(result app.LectureTranscriptResult) {
	for _, item := range result.Items {
		m.applyTranscriptItem(item)
	}
}

func (m *lectureDownloadModel) applyTranscriptItem(item app.LectureTranscriptItem) {
	stage := "transcribed"
	if item.Err != nil {
		stage = "transcript-error"
	}
	m.upsertTranscriptStatusLine(app.LectureTranscriptProgress{
		Lecture:    item.Lecture,
		InputPath:  item.InputPath,
		OutputPath: item.OutputPath,
		Stage:      stage,
		Err:        item.Err,
	})
}

func (m lectureDownloadModel) renderProgressList(width int) string {
	if len(m.items) == 0 {
		return emptyStyle.Render("다운로드 준비 중") + "\n"
	}
	lines := make([]string, 0, len(m.items)*2)
	visibleItems := m.visibleDownloadItems()
	start := m.downloadScrollStart(visibleItems)
	end := minInt(len(m.items), start+visibleItems)
	for index := start; index < end; index++ {
		item := m.items[index]
		label := truncateText(item.label, maxInt(20, minInt(64, width-18)))
		marker := "  "
		if index == m.cursor {
			marker = "› "
		}
		lines = append(lines, fmt.Sprintf("%s%s", marker, label))
		lines = append(lines, "  "+m.renderItemProgress(item, maxInt(18, minInt(56, width-26)))+"  "+m.itemProgressText(item))
	}
	if start > 0 || end < len(m.items) {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(m.items))))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m lectureDownloadModel) visibleDownloadItems() int {
	if m.height <= 0 {
		return 8
	}
	return maxInt(3, (m.height-9)/2)
}

func (m lectureDownloadModel) downloadScrollStart(visibleItems int) int {
	start := m.cursor - visibleItems/2
	if start < 0 {
		return 0
	}
	if start+visibleItems > len(m.items) {
		return maxInt(0, len(m.items)-visibleItems)
	}
	return start
}

func (m lectureDownloadModel) renderSummary() string {
	done, skipped, failed := 0, 0, 0
	for _, item := range m.items {
		switch {
		case item.err != nil:
			failed++
		case item.skipped:
			skipped++
		case item.status == "done" || item.status == "transcribed":
			done++
		}
	}
	elapsed := time.Since(m.startedAt).Round(time.Second)
	if m.err != nil && failed == 0 {
		failed = 1
	}
	badge := successBadgeStyle.Render("DONE")
	if m.err != nil {
		badge = errorStyle.Render("ERROR")
	}
	return fmt.Sprintf("%s 완료 %d · 건너뜀 %d · 실패 %d · %s\n", badge, done, skipped, failed, elapsed)
}

func lectureDownloadLabel(row app.LectureRow) string {
	parts := []string{row.CourseName, row.Lecture.ModuleTitle, row.Lecture.Title}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " · ")
}

func initialDownloadStatusLines(rows []app.LectureRow) []downloadStatusLine {
	lines := make([]downloadStatusLine, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		label := lectureDownloadLabel(row)
		if label == "" {
			continue
		}
		id := lectureDownloadKey(row)
		key := id
		if key == "" {
			key = label
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		lines = append(lines, downloadStatusLine{id: id, label: label, status: "pending"})
	}
	return lines
}

func lectureDownloadKey(row app.LectureRow) string {
	if strings.TrimSpace(row.ID) != "" {
		return strings.TrimSpace(row.ID)
	}
	parts := []string{row.TermValue, row.CourseName, row.Lecture.ContentID, row.Lecture.ModuleTitle, row.Lecture.Title}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "\x00")
}

func (line downloadStatusLine) matches(other downloadStatusLine) bool {
	if line.id != "" && other.id != "" {
		return line.id == other.id
	}
	return line.label == other.label
}

func preservesTranscriptStatus(current string, next string) bool {
	switch current {
	case "transcribe", "transcribed", "transcript-error":
		return next == "done"
	default:
		return false
	}
}

func (m lectureDownloadModel) renderItemProgress(item downloadStatusLine, width int) string {
	bar := bubblesprogress.New(
		bubblesprogress.WithWidth(width),
		bubblesprogress.WithFillCharacters('█', '░'),
		bubblesprogress.WithoutPercentage(),
		bubblesprogress.WithSolidFill(itemProgressColor(item)),
	)
	return bar.ViewAs(itemProgressPercent(item))
}

func itemProgressColor(item downloadStatusLine) string {
	switch {
	case item.err != nil || item.status == "error" || item.status == "transcript-error":
		return "#CF222E"
	case item.status == "transcribe" || item.status == "transcribed":
		return "#9ACD32"
	case item.status == "skip" || item.skipped:
		return "#6E7781"
	default:
		return "#58A6FF"
	}
}

func itemProgressPercent(item downloadStatusLine) float64 {
	if item.err != nil {
		return 1
	}
	switch item.status {
	case "done", "skip", "error", "transcribed", "transcript-error":
		return 1
	case "transcribe":
		if item.percent > 0 {
			return clampPercent(item.percent)
		}
		return 0.05
	case "download":
		if item.total > 0 {
			percent := float64(item.bytes) / float64(item.total)
			if percent < 0 {
				return 0
			}
			if percent > 1 {
				return 1
			}
			return percent
		}
		if item.bytes > 0 {
			return 0.05
		}
	}
	if item.skipped {
		return 1
	}
	return 0
}

func clampPercent(percent float64) float64 {
	if percent < 0 {
		return 0
	}
	if percent > 1 {
		return 1
	}
	return percent
}

func (m lectureDownloadModel) itemProgressText(item downloadStatusLine) string {
	status := downloadStageLabel(item.status)
	switch {
	case item.err != nil:
		return errorStyle.Render(truncateText(item.err.Error(), 34))
	case item.status == "transcribe":
		if item.percent > 0 {
			return taglineStyle.Render(fmt.Sprintf("전사중 %.0f%%", clampPercent(item.percent)*100))
		}
		return taglineStyle.Render("전사중")
	case item.status == "transcribed":
		return taglineStyle.Render("전사완료")
	case item.skipped:
		return mutedStyle.Render("건너뜀")
	case item.status == "done":
		if strings.TrimSpace(item.path) != "" {
			return mutedStyle.Render(truncateText(filepath.Base(item.path), 34))
		}
		return mutedStyle.Render("완료")
	case item.status == "download":
		if item.total > 0 {
			return mutedStyle.Render(fmt.Sprintf("%s/%s", formatDownloadBytes(item.bytes), formatDownloadBytes(item.total)))
		}
		if item.bytes > 0 {
			return mutedStyle.Render(formatDownloadBytes(item.bytes))
		}
	}
	return mutedStyle.Render(status)
}

func lectureDownloadPercent(progress app.LectureDownloadProgress) float64 {
	if progress.TotalBytes > 0 {
		percent := float64(progress.Bytes) / float64(progress.TotalBytes)
		if percent < 0 {
			return 0
		}
		if percent > 1 {
			return 1
		}
		return percent
	}
	if progress.TotalItems > 0 {
		percent := float64(maxInt(0, progress.CurrentIndex-1)) / float64(progress.TotalItems)
		if progress.Stage == "done" || progress.Stage == "skip" || progress.Stage == "error" {
			percent = float64(progress.CurrentIndex) / float64(progress.TotalItems)
		}
		return percent
	}
	return 0
}

func downloadStageLabel(stage string) string {
	switch stage {
	case "resolve":
		return "확인"
	case "download":
		return "받는중"
	case "done":
		return "완료"
	case "skip":
		return "건너뜀"
	case "error":
		return "실패"
	case "transcribe":
		return "전사중"
	case "transcribed":
		return "전사완료"
	case "transcript-error":
		return "전사실패"
	default:
		return "대기"
	}
}

func formatDownloadProgress(progress app.LectureDownloadProgress) string {
	if progress.TotalBytes > 0 {
		return fmt.Sprintf("%s/%s", formatDownloadBytes(progress.Bytes), formatDownloadBytes(progress.TotalBytes))
	}
	if progress.Bytes > 0 {
		return formatDownloadBytes(progress.Bytes)
	}
	return downloadStageLabel(progress.Stage)
}

func formatDownloadBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KB", "MB", "GB", "TB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f PB", value/unit)
}

func (m *lectureDownloadModel) cancelAndCleanup() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.canceling = true
	pipeline, updates := m.pipeline, m.updates
	return func() tea.Msg { return lectureDownloadCleanupMsg{updates: updates, err: pipeline.CancelAndCleanup()} }
}

func (m *lectureDownloadModel) cancelAndWait(quit bool) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.canceling = true
	pipeline, updates := m.pipeline, m.updates
	return func() tea.Msg {
		_, _ = pipeline.Run()
		if quit {
			return tea.Quit()
		}
		return lectureDownloadStoppedMsg{updates: updates}
	}
}
