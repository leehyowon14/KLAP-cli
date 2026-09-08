package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type downloadScreenModel struct {
	downloadRows       []app.LectureRow
	downloadSelected   map[string]bool
	downloadCourse     int
	downloadCursor     int
	downloadTranscribe bool
	downloadLanguage   int
	downloadProgress   *lectureDownloadModel
	preparing          bool
	generation         uint64
}
type downloadScreenService interface {
	LectureList(context.Context, app.LectureListOptions) ([]app.LectureRow, error)
	DownloadSettings() (app.DownloadSettings, error)
	TranscriptSettings() (app.TranscriptSettings, error)
	NewLectureDownloadRun(context.Context, app.LectureDownloadPipelineOptions) *app.LectureDownloadRun
}
type downloadPreparedMsg struct {
	generation uint64
	progress   *lectureDownloadModel
	err        error
}

func (m *downloadScreenModel) Loaded(rows []app.LectureRow) {
	m.downloadRows = rows
	m.downloadSelected = make(map[string]bool, len(rows))
	m.downloadCourse, m.downloadCursor = 0, 0
}
func (m *downloadScreenModel) prepare(ctx context.Context, service downloadScreenService) childAction {
	m.preparing = true
	m.generation++
	generation := m.generation
	rows := m.selectedDownloadRows()
	request := LectureDownloadRequest{LectureIDs: m.selectedDownloadIDs(), Transcribe: m.downloadTranscribe, TranscriptLocale: m.selectedTranscriptLocale()}
	return childAction{setError: true, cmd: func() tea.Msg {
		settings, err := service.DownloadSettings()
		if err != nil {
			return downloadPreparedMsg{generation: generation, err: err}
		}
		transcriptSettings, err := service.TranscriptSettings()
		if err != nil {
			return downloadPreparedMsg{generation: generation, err: err}
		}
		if err := ctx.Err(); err != nil {
			return downloadPreparedMsg{generation: generation, err: err}
		}
		request.Concurrency, request.TranscriptConcurrency = settings.Concurrency, transcriptSettings.Concurrency
		runCtx, cancel := context.WithCancel(ctx)
		progress := &lectureDownloadModel{ctx: runCtx, cancel: cancel, request: request, updates: make(chan tea.Msg, 64), items: initialDownloadStatusLines(rows), startedAt: time.Now()}
		progress.pipeline = service.NewLectureDownloadRun(runCtx, progress.pipelineOptions())
		return downloadPreparedMsg{generation: generation, progress: progress}
	}}
}
func (m *downloadScreenModel) Lifecycle(msg tea.Msg) (childAction, bool) {
	switch msg := msg.(type) {
	case downloadPreparedMsg:
		if !m.preparing || msg.generation != m.generation {
			if msg.progress != nil {
				msg.progress.cancel()
			}
			return childAction{}, true
		}
		m.preparing = false
		if msg.err != nil {
			return childAction{setError: true, err: msg.err}, true
		}
		m.downloadProgress = msg.progress
		return childAction{navigate: true, routeOnly: true, target: screenDownloadProgress, setError: true, cmd: msg.progress.Init()}, true
	case lectureDownloadCleanupMsg:
		if m.downloadProgress == nil || m.downloadProgress.updates != msg.updates {
			return childAction{setError: msg.err != nil, err: msg.err}, true
		}
		if msg.err != nil {
			m.downloadProgress.err = msg.err
			m.downloadProgress.done = true
			m.downloadProgress.canceling = false
			return childAction{}, true
		}
		m.downloadProgress = nil
		return childAction{navigate: true, reload: true, refresh: true, target: screenLectures}, true
	case lectureDownloadStoppedMsg:
		if m.downloadProgress != nil && m.downloadProgress.updates == msg.updates {
			m.downloadProgress = nil
			return childAction{navigate: true, target: screenHome, setError: true}, true
		}
		return childAction{}, true
	}
	return childAction{}, false
}
func (m *downloadScreenModel) Update(msg tea.Msg, route screen, loading bool, ctx context.Context, service downloadScreenService) (childAction, bool) {
	if route == screenDownloadProgress {
		return m.updateProgress(msg), true
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	if m.preparing {
		if key.String() == "ctrl+c" || keyMatches(key.String(), "q", "ㅂ") {
			return childAction{cmd: tea.Quit}, true
		}
		return childAction{}, true
	}
	switch route {
	case screenDownloadSelect:
		return m.updateDownloadSelect(key, loading, ctx, service), true
	case screenDownloadConfirm:
		return m.updateDownloadConfirm(key, loading, ctx, service), true
	case screenDownloadLanguage:
		return m.updateDownloadLanguage(key, loading, ctx, service), true
	}
	return childAction{}, false
}
func (m *downloadScreenModel) updateProgress(msg tea.Msg) childAction {
	if m.downloadProgress == nil {
		return childAction{navigate: true, routeOnly: true, target: screenLectures}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.downloadProgress.canceling {
			return childAction{}
		}
		if key.String() == "ctrl+c" || keyMatches(key.String(), "q", "ㅂ") {
			return childAction{cmd: m.downloadProgress.cancelAndWait(true)}
		}
		if keyMatches(key.String(), "h", "ㅗ") {
			return childAction{cmd: m.downloadProgress.cancelAndWait(false)}
		}
		if m.downloadProgress.done && (key.String() == "esc" || keyMatches(key.String(), "b", "ㅠ")) {
			m.downloadProgress = nil
			return childAction{navigate: true, reload: true, refresh: true, target: screenLectures}
		}
		if key.String() == "esc" && !m.downloadProgress.done {
			return childAction{cmd: m.downloadProgress.cancelAndCleanup()}
		}
	}
	updated, cmd := m.downloadProgress.Update(msg)
	if progress, ok := updated.(lectureDownloadModel); ok {
		m.downloadProgress = &progress
	}
	return childAction{cmd: cmd}
}
func (m downloadScreenModel) View(width, height int, route screen, loading bool, err error) string {
	switch route {
	case screenDownloadSelect:
		return m.renderDownloadSelectView(width, height, loading, err)
	case screenDownloadConfirm:
		return m.renderDownloadConfirmView(width) + m.preparationStatus(err)
	case screenDownloadLanguage:
		return m.renderDownloadLanguageView(width) + m.preparationStatus(err)
	default:
		if m.downloadProgress == nil {
			return appStyle.Render(errorStyle.Render("다운로드 상태가 없습니다"))
		}
		return m.downloadProgress.View()
	}
}

func (m downloadScreenModel) preparationStatus(err error) string {
	if err != nil {
		return "\n\n" + errorStyle.Render(err.Error())
	}
	if m.preparing {
		return "\n\n" + mutedStyle.Render("다운로드를 준비하고 있습니다")
	}
	return ""
}
