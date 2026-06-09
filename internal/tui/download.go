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
	Target      string
	User        app.UserOption
	Dir         string
	All         bool
	LectureIDs  []string
	Concurrency int
}

type lectureDownloadModel struct {
	ctx       context.Context
	cancel    context.CancelFunc
	service   *app.Service
	request   LectureDownloadRequest
	updates   chan tea.Msg
	progress  bubblesprogress.Model
	width     int
	startedAt time.Time
	current   app.LectureDownloadProgress
	items     []downloadStatusLine
	done      bool
	canceling bool
	err       error
}

type downloadStatusLine struct {
	label   string
	status  string
	path    string
	bytes   int64
	skipped bool
	err     error
}

type lectureDownloadProgressMsg struct {
	progress app.LectureDownloadProgress
}

type lectureDownloadDoneMsg struct {
	single app.LectureDownloadResult
	all    app.LectureDownloadAllResult
	err    error
}

func RunLectureDownload(ctx context.Context, service *app.Service, request LectureDownloadRequest) error {
	if service == nil {
		return errors.New("다운로드 service가 없습니다")
	}
	runCtx, cancel := context.WithCancel(ctx)
	model := lectureDownloadModel{
		ctx:       runCtx,
		cancel:    cancel,
		service:   service,
		request:   request,
		updates:   make(chan tea.Msg, 64),
		progress:  bubblesprogress.New(bubblesprogress.WithWidth(36), bubblesprogress.WithFillCharacters('█', '░')),
		startedAt: time.Now(),
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
		m.progress.Width = maxInt(20, minInt(52, msg.Width-28))
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || keyMatches(key, "q", "ㅂ") {
			m.canceling = true
			m.cancel()
			return m, nil
		}
	case lectureDownloadProgressMsg:
		m.current = msg.progress
		m.upsertStatusLine(msg.progress)
		return m, waitLectureDownloadProgress(m.updates)
	case lectureDownloadDoneMsg:
		m.done = true
		m.err = msg.err
		m.applyFinalResult(msg)
		return m, tea.Quit
	}
	return m, nil
}

func (m lectureDownloadModel) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	m.progress.Width = maxInt(20, minInt(52, width-28))

	var b strings.Builder
	b.WriteString(headerStyle.Render("KLAP"))
	b.WriteString(mutedStyle.Render(" download"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxInt(24, minInt(width-2, 96)))))
	b.WriteString("\n\n")
	b.WriteString(m.renderCurrent())
	b.WriteString("\n")
	b.WriteString(m.renderQueue(width))
	b.WriteString("\n")
	if m.done {
		b.WriteString(m.renderSummary())
	} else if m.canceling {
		b.WriteString(warnBadgeStyle.Render("CANCEL"))
		b.WriteString(" 다운로드를 중단하는 중입니다\n")
	} else {
		b.WriteString(footerStyle.Render("q 종료"))
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
		label:   label,
		status:  progress.Stage,
		path:    progress.Path,
		bytes:   progress.Bytes,
		skipped: progress.Skipped,
		err:     progress.Err,
	}
	for index := range m.items {
		if m.items[index].label == label {
			m.items[index] = line
			return
		}
	}
	m.items = append(m.items, line)
}

func (m *lectureDownloadModel) applyFinalResult(msg lectureDownloadDoneMsg) {
	if msg.single.Path != "" {
		m.upsertStatusLine(app.LectureDownloadProgress{
			Lecture: msg.single.Lecture,
			Path:    msg.single.Path,
			Stage:   "done",
			Bytes:   msg.single.Bytes,
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
			Lecture: item.Lecture,
			Path:    item.Path,
			Stage:   stage,
			Bytes:   item.Bytes,
			Skipped: item.Skipped,
			Err:     item.Err,
		})
	}
}

func (m lectureDownloadModel) renderCurrent() string {
	progress := m.current
	percent := lectureDownloadPercent(progress)
	stage := downloadStageLabel(progress.Stage)
	if m.done {
		stage = "완료"
		if m.err != nil {
			stage = "오류"
		}
	}
	label := lectureDownloadLabel(progress.Lecture)
	if label == "" {
		label = m.request.Target
	}
	count := ""
	if progress.TotalItems > 0 {
		count = fmt.Sprintf(" %d/%d", progress.CurrentIndex, progress.TotalItems)
	}

	var b strings.Builder
	b.WriteString(successBadgeStyle.Render(stage))
	b.WriteString(count)
	b.WriteString("  ")
	b.WriteString(truncateText(label, 48))
	b.WriteString("\n")
	b.WriteString(m.progress.ViewAs(percent))
	b.WriteString("  ")
	b.WriteString(formatDownloadProgress(progress))
	b.WriteString("\n")
	return b.String()
}

func (m lectureDownloadModel) renderQueue(width int) string {
	if len(m.items) == 0 {
		return renderSection("QUEUE", []string{emptyStyle.Render("다운로드 준비 중")})
	}
	lines := make([]string, 0, len(m.items))
	for _, item := range m.items {
		label := truncateText(item.label, maxInt(20, minInt(44, width-38)))
		status := downloadStageLabel(item.status)
		switch {
		case item.err != nil:
			status = "실패"
		case item.skipped:
			status = "건너뜀"
		}
		detail := formatDownloadBytes(item.bytes)
		if item.path != "" {
			detail = filepath.Base(item.path)
		}
		if item.err != nil {
			detail = item.err.Error()
		}
		lines = append(lines, fmt.Sprintf("%s  %s  %s", mutedStyle.Render(status), label, truncateText(detail, 34)))
	}
	return renderSection("QUEUE", lines)
}

func (m lectureDownloadModel) renderSummary() string {
	done, skipped, failed := 0, 0, 0
	for _, item := range m.items {
		switch {
		case item.err != nil:
			failed++
		case item.skipped:
			skipped++
		case item.status == "done":
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
