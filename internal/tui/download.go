package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	bubblesprogress "github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kw-klap/klap-cli/internal/app"
)

type LectureDownloadRequest struct {
	Target                string
	User                  app.UserOption
	Dir                   string
	All                   bool
	Rows                  []app.LectureRow
	LectureIDs            []string
	Concurrency           int
	Transcribe            bool
	TranscriptLocale      string
	TranscriptConcurrency int
}

type lectureDownloadModel struct {
	ctx               context.Context
	cancel            context.CancelFunc
	service           *app.Service
	request           LectureDownloadRequest
	updates           chan tea.Msg
	width             int
	height            int
	startedAt         time.Time
	items             []downloadStatusLine
	cursor            int
	downloadsDone     bool
	done              bool
	canceling         bool
	err               error
	quitOnDone        bool
	transcriptStarted map[string]bool
	transcriptRunning map[string]bool
	transcriptQueue   []app.LectureDownloadItem
	transcriptActive  int
}

type downloadStatusLine struct {
	id      string
	label   string
	status  string
	path    string
	bytes   int64
	total   int64
	percent float64
	skipped bool
	err     error
}

type lectureTranscriptProgressMsg struct {
	progress app.LectureTranscriptProgress
}

type lectureDownloadProgressMsg struct {
	progress app.LectureDownloadProgress
}

type lectureDownloadDoneMsg struct {
	single      app.LectureDownloadResult
	all         app.LectureDownloadAllResult
	transcripts app.LectureTranscriptResult
	err         error
}

type lectureTranscriptDoneMsg struct {
	key    string
	result app.LectureTranscriptResult
}

func RunLectureDownload(ctx context.Context, service *app.Service, request LectureDownloadRequest) error {
	if service == nil {
		return errors.New("다운로드 service가 없습니다")
	}
	runCtx, cancel := context.WithCancel(ctx)
	model := lectureDownloadModel{
		ctx:               runCtx,
		cancel:            cancel,
		service:           service,
		request:           request,
		updates:           make(chan tea.Msg, 64),
		items:             initialDownloadStatusLines(request.Rows),
		startedAt:         time.Now(),
		quitOnDone:        true,
		transcriptStarted: make(map[string]bool),
		transcriptRunning: make(map[string]bool),
	}
	finalModel, err := tea.NewProgram(model).Run()
	cancel()
	if err != nil {
		return err
	}
	if model, ok := finalModel.(lectureDownloadModel); ok && model.err != nil && !errors.Is(model.err, context.Canceled) {
		return model.err
	}
	return nil
}

func (m lectureDownloadModel) Init() tea.Cmd {
	return tea.Batch(m.runDownload(), waitLectureDownloadProgress(m.updates))
}

func (m lectureDownloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		cmds := []tea.Cmd{waitLectureDownloadProgress(m.updates)}
		cmds = append(cmds, m.enqueueTranscriptForDownloadProgress(msg.progress)...)
		return m, tea.Batch(cmds...)
	case lectureTranscriptProgressMsg:
		m.upsertTranscriptStatusLine(msg.progress)
		return m, waitLectureDownloadProgress(m.updates)
	case lectureDownloadDoneMsg:
		m.downloadsDone = true
		m.err = msg.err
		m.applyFinalResult(msg)
		cmds := m.enqueueTranscriptsForResult(msg)
		m.markDoneIfIdle()
		if m.done && m.quitOnDone {
			return m, tea.Quit
		}
		return m, tea.Batch(cmds...)
	case lectureTranscriptDoneMsg:
		delete(m.transcriptRunning, msg.key)
		if m.transcriptActive > 0 {
			m.transcriptActive--
		}
		m.applyTranscriptResult(msg.result)
		cmds := m.startTranscriptWorkers()
		m.markDoneIfIdle()
		if m.done && m.quitOnDone {
			return m, tea.Quit
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m lectureDownloadModel) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("KLAP"))
	b.WriteString(mutedStyle.Render(" download"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 96)))))
	b.WriteString("\n\n")
	b.WriteString(m.renderProgressList(width))
	b.WriteString("\n")
	if m.done {
		b.WriteString(m.renderSummary())
	} else if m.canceling {
		b.WriteString(warnBadgeStyle.Render("CANCEL"))
		b.WriteString(" 다운로드를 중단하는 중입니다\n")
	} else {
		b.WriteString(footerStyle.Render("↑↓ 이동  |  q 종료"))
		b.WriteString("\n")
	}
	return appStyle.Render(b.String())
}

func (m lectureDownloadModel) runDownload() tea.Cmd {
	return func() tea.Msg {
		onProgress := func(progress app.LectureDownloadProgress) {
			select {
			case m.updates <- lectureDownloadProgressMsg{progress: progress}:
			default:
			}
		}
		if m.request.All {
			result, err := m.service.DownloadAllLectures(m.ctx, app.LectureDownloadAllOptions{
				User:         m.request.User,
				CourseFilter: m.request.Target,
				Dir:          m.request.Dir,
				OnProgress:   onProgress,
				Concurrency:  m.request.Concurrency,
				LectureIDs:   m.request.LectureIDs,
			})
			return lectureDownloadDoneMsg{all: result, err: err}
		}
		result, err := m.service.DownloadLecture(m.ctx, m.request.Target, app.LectureDownloadOptions{
			User:       m.request.User,
			Dir:        m.request.Dir,
			OnProgress: onProgress,
		})
		return lectureDownloadDoneMsg{single: result, err: err}
	}
}

func (m *lectureDownloadModel) enqueueTranscriptForDownloadProgress(progress app.LectureDownloadProgress) []tea.Cmd {
	if !m.request.Transcribe || progress.Stage != "done" || strings.TrimSpace(progress.Path) == "" {
		return nil
	}
	return m.enqueueTranscriptItem(app.LectureDownloadItem{
		Lecture: progress.Lecture,
		Path:    progress.Path,
		Bytes:   progress.Bytes,
	})
}

func (m *lectureDownloadModel) enqueueTranscriptItem(item app.LectureDownloadItem) []tea.Cmd {
	if !m.request.Transcribe || !app.LectureDownloadItemNeedsTranscript(item) {
		return nil
	}
	key := lectureDownloadKey(item.Lecture)
	if key == "" {
		return nil
	}
	if m.transcriptStarted == nil {
		m.transcriptStarted = make(map[string]bool)
	}
	if m.transcriptRunning == nil {
		m.transcriptRunning = make(map[string]bool)
	}
	if m.transcriptStarted[key] {
		return nil
	}
	m.transcriptStarted[key] = true
	m.upsertTranscriptStatusLine(app.LectureTranscriptProgress{
		Lecture:    item.Lecture,
		InputPath:  item.Path,
		OutputPath: app.TranscriptPathForDownload(item.Path),
		Stage:      "transcribe",
	})
	m.transcriptQueue = append(m.transcriptQueue, item)
	return m.startTranscriptWorkers()
}

func (m *lectureDownloadModel) startTranscriptWorkers() []tea.Cmd {
	if !m.request.Transcribe {
		return nil
	}
	if m.transcriptRunning == nil {
		m.transcriptRunning = make(map[string]bool)
	}
	limit := m.request.TranscriptConcurrency
	if limit <= 0 {
		limit = 1
	}
	cmds := make([]tea.Cmd, 0)
	for m.transcriptActive < limit && len(m.transcriptQueue) > 0 {
		item := m.transcriptQueue[0]
		m.transcriptQueue = m.transcriptQueue[1:]
		key := lectureDownloadKey(item.Lecture)
		if key == "" {
			continue
		}
		m.transcriptRunning[key] = true
		m.transcriptActive++
		cmds = append(cmds, m.transcriptCommandForItem(key, item))
	}
	return cmds
}

func (m lectureDownloadModel) transcriptCommandForItem(key string, item app.LectureDownloadItem) tea.Cmd {
	return func() tea.Msg {
		onProgress := func(transcriptProgress app.LectureTranscriptProgress) {
			select {
			case m.updates <- lectureTranscriptProgressMsg{progress: transcriptProgress}:
			default:
			}
		}
		result := m.service.TranscribeDownloadedLectures(m.ctx, []app.LectureDownloadItem{item}, app.LectureTranscriptOptions{Locale: m.request.TranscriptLocale, OnProgress: onProgress})
		return lectureTranscriptDoneMsg{key: key, result: result}
	}
}

func waitLectureDownloadProgress(updates chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-updates
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
	for index := range m.items {
		if m.items[index].matches(line) {
			if preservesTranscriptStatus(m.items[index].status, line.status) {
				m.items[index].bytes = line.bytes
				m.items[index].total = line.total
				return
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
	for index := range m.items {
		if m.items[index].matches(line) {
			m.items[index].status = line.status
			m.items[index].path = line.path
			m.items[index].percent = line.percent
			m.items[index].err = line.err
			return
		}
	}
	m.items = append(m.items, line)
}

func (m *lectureDownloadModel) applyFinalResult(msg lectureDownloadDoneMsg) {
	if msg.single.Path != "" {
		m.upsertStatusLine(app.LectureDownloadProgress{
			Lecture:    msg.single.Lecture,
			Path:       msg.single.Path,
			Stage:      "done",
			Bytes:      msg.single.Bytes,
			TotalBytes: msg.single.Bytes,
		})
	}
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
	for _, item := range msg.transcripts.Items {
		m.applyTranscriptItem(item)
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

func (m *lectureDownloadModel) enqueueTranscriptsForResult(msg lectureDownloadDoneMsg) []tea.Cmd {
	if !m.request.Transcribe {
		return nil
	}
	cmds := make([]tea.Cmd, 0)
	if msg.single.Path != "" {
		cmds = append(cmds, m.enqueueTranscriptItem(app.LectureDownloadItem{
			Lecture: msg.single.Lecture,
			Path:    msg.single.Path,
			Bytes:   msg.single.Bytes,
		})...)
	}
	for _, item := range msg.all.Items {
		cmds = append(cmds, m.enqueueTranscriptItem(item)...)
	}
	return cmds
}

func (m *lectureDownloadModel) markDoneIfIdle() {
	if !m.downloadsDone {
		return
	}
	if m.request.Transcribe && (len(m.transcriptRunning) > 0 || len(m.transcriptQueue) > 0 || m.transcriptActive > 0) {
		return
	}
	m.done = true
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
	if item.err != nil || item.skipped {
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
	case item.skipped:
		return mutedStyle.Render("건너뜀")
	case item.status == "transcribe":
		if item.percent > 0 {
			return taglineStyle.Render(fmt.Sprintf("전사중 %.0f%%", clampPercent(item.percent)*100))
		}
		return taglineStyle.Render("전사중")
	case item.status == "transcribed":
		return taglineStyle.Render("전사완료")
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
