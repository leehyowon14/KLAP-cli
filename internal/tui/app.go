package tui

import (
	"context"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type screen int

const (
	screenHome screen = iota
	screenAuth
	screenDashboard
	screenDue
	screenAssignments
	screenNotices
	screenLectures
	screenAssignmentDetail
	screenNoticeDetail
	screenSyllabus
	screenAcademic
	screenConfig
	screenRoomDay
	screenRoomPeriod
	screenRoomResult
	screenSyncConflict
	screenConfigChoice
	screenConfigInput
	screenDownloadSelect
	screenDownloadConfirm
	screenDownloadLanguage
	screenDownloadProgress
	screenAttendConfirm
	screenAttendProgress
)

type model struct {
	ctx                context.Context
	service            *app.Service
	home               homeModel
	active             screen
	loading            bool
	loadedScreens      map[screen]bool
	loadingScreens     map[screen]bool
	screenErrors       map[screen]error
	prefetchQueue      []screen
	prefetchCurrent    screen
	prefetchActive     bool
	err                error
	content            string
	dashboard          dashboardScreenModel
	assignments        assignmentScreenModel
	notices            noticeScreenModel
	lectures           lectureScreenModel
	due                dueScreenModel
	academic           academicScreenModel
	syllabus           syllabusScreenModel
	detailBack         screen
	syncStatus         string
	width              int
	height             int
	loadedAt           time.Time
	lastSyncAt         time.Time
	config             configScreenModel
	downloadRows       []app.LectureRow
	downloadSelected   map[string]bool
	downloadCourse     int
	downloadCursor     int
	downloadTranscribe bool
	downloadLanguage   int
	downloadProgress   *lectureDownloadModel
	attendRow          app.LectureRow
	attendProgress     *lectureAttendModel
	room               roomScreenModel
	sync               syncScreenModel
	auth               authScreenModel
}

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

type loadMsg struct {
	screen      screen
	prefetch    bool
	content     string
	err         error
	assignments []app.AssignmentRow
	notices     []app.NoticeRow
	lectures    []app.LectureRow
	due         app.DueResult
	academic    app.AcademicListResult
	dashboard   app.DashboardResult
	config      app.ConfigSettings
	categories  app.CategoryOptions
	users       []app.UserRow
	terms       []app.TermRow
}

type downloadRowsMsg struct {
	rows []app.LectureRow
	err  error
}

type statusMsg struct {
	status string
	err    error
}

type detailMsg struct {
	screen     screen
	assignment app.AssignmentDetailResult
	notice     app.NoticeDetailResult
	err        error
}

func Run(ctx context.Context, service *app.Service) error {
	if service == nil {
		return errors.New("TUI service가 없습니다")
	}
	initial := model{
		ctx:            ctx,
		service:        service,
		active:         screenAuth,
		loading:        true,
		loadedScreens:  map[screen]bool{},
		loadingScreens: map[screen]bool{},
		screenErrors:   map[screen]error{},

		home: newHomeModel(),
		auth: authScreenModel{authInputs: newAuthInputs()},
	}
	_, err := tea.NewProgram(initial, tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd {
	return checkAuthUsers(m.ctx, m.service)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cleaned, ok := msg.(lectureDownloadCleanupMsg); ok {
		if m.downloadProgress == nil || m.downloadProgress.updates != cleaned.updates {
			if cleaned.err != nil {
				m.err = cleaned.err
			}
			return m, nil
		}
		if cleaned.err != nil {
			m.downloadProgress.err = cleaned.err
			m.downloadProgress.done = true
			m.downloadProgress.canceling = false
			return m, nil
		}
		m.active = screenLectures
		m.loading = true
		m.downloadProgress = nil
		return m, m.load(screenLectures, true)
	}
	if stopped, ok := msg.(lectureDownloadStoppedMsg); ok {
		if m.downloadProgress != nil && m.downloadProgress.updates == stopped.updates {
			m.active = screenHome
			m.loading = false
			m.err = nil
			m.content = ""
			m.downloadProgress = nil
		}
		return m, nil
	}

	if m.active == screenDownloadProgress {
		return m.updateDownloadProgress(msg)
	}
	if m.active == screenAttendProgress {
		return m.updateAttendProgress(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.active == screenAuth {
			action, _ := m.auth.Update(msg, m.ctx, m.service)
			return m.applyChildAction(action)
		}
		if m.active == screenDownloadSelect {
			return m.updateDownloadSelect(msg)
		}
		if m.active == screenDownloadConfirm {
			return m.updateDownloadConfirm(msg)
		}
		if m.active == screenDownloadLanguage {
			return m.updateDownloadLanguage(msg)
		}
		if m.active == screenAttendConfirm {
			return m.updateAttendConfirm(msg)
		}
		if m.active == screenConfig || m.active == screenConfigChoice || m.active == screenConfigInput {
			if action, handled := m.config.Update(msg, m.active, m.loading, m.ctx, m.service); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenRoomDay || m.active == screenRoomPeriod || m.active == screenRoomResult {
			if action, handled := m.room.Update(msg, m.active, m.loading, m.ctx, m.service); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenSyncConflict {
			action, _ := m.sync.Update(msg, m.ctx, m.service, m.due.duePage == 3)
			return m.applyChildAction(action)
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
		if m.active == screenHome {
			if action, handled := m.home.Update(msg); handled {
				return m.applyChildAction(action)
			}
		}
		if (m.active == screenAssignments || m.active == screenAssignmentDetail) && !m.loading {
			if action, handled := m.assignments.Update(msg, m.width, m.height, m.active == screenAssignmentDetail, m.ctx, m.service); handled {
				return m.applyChildAction(action)
			}
		}
		if (m.active == screenNotices || m.active == screenNoticeDetail) && !m.loading {
			if action, handled := m.notices.Update(msg, m.width, m.height, m.active == screenNoticeDetail, m.ctx, m.service); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenDashboard && !m.loading {
			if action, handled := m.dashboard.Update(msg, m.width, m.height, m.lastSyncAt); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenAcademic && !m.loading {
			if action, handled := m.academic.Update(msg); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenDue && !m.loading {
			if action, handled := m.due.Update(msg, m.width); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenSyllabus {
			if action, handled := m.syllabus.Update(msg, m.width, m.height, m.loading, m.ctx, m.service, m.dashboard.dashboardResult.Term.Value); handled {
				return m.applyChildAction(action)
			}
		}
		if m.active == screenLectures {
			if action, handled := m.lectures.Update(msg, m.width, m.loading); handled {
				return m.applyChildAction(action)
			}
		}
		key := msg.String()
		switch {
		case keyMatches(key, "q", "ㅂ"):
			return m, tea.Quit
		case key == "esc" || keyMatches(key, "b", "ㅠ"):
			if m.active == screenSyllabus {
				m.active = screenDashboard
				m.err = nil
				m.loading = false
				m.syllabus.resetCursor()
			} else if m.isDetailScreen() {
				if m.active == screenAssignmentDetail {
					m.assignments.resetDetailCursor()
				} else {
					m.notices.resetDetailCursor()
				}
				m.active = m.detailBack
				m.err = nil
				m.loading = false
			} else if m.active != screenHome {
				m.active = screenHome
				m.err = nil
				m.content = ""
				m.loading = false
			}
		case key == "up" || (keyMatches(key, "k", "ㅏ") && !m.canOpenKlasURL()):
			if m.isDetailScreen() && !m.loading {
				m.moveDetailCursor(-1)
			}
		case key == "down" || keyMatches(key, "j", "ㅓ"):
			if m.isDetailScreen() && !m.loading {
				m.moveDetailCursor(1)
			}
		case keyMatches(key, "r", "ㄱ"):
			if m.active != screenHome {
				m.loading = true
				m.err = nil
				m.content = ""
				m.syncStatus = ""
				m.markScreenLoading(m.active)
				return m, m.load(m.active, true)
			}
		case keyMatches(key, "s", "ㄴ"):
			if cmd := syncForScreen(m.ctx, m.service, m.active, m.due.duePage == 3, nil); cmd != nil {
				m.loading = false
				m.err = nil
				m.sync.Start(m.active)
				m.syncStatus = ""
				return m, cmd
			}
		case keyMatches(key, "k", "ㅏ"):
			if cmd := m.openCurrentKlasURL(); cmd != nil {
				m.err = nil
				return m, cmd
			}
		}
	case configSavedMsg:
		if !m.config.Accepts(msg) {
			return m, nil
		}
		action := m.config.Saved(msg)
		if msg.invalidate {
			m.resetLoadedMainScreens()
		}
		if msg.loaded != nil && msg.loaded.err == nil {
			m.loadedAt = time.Now()
		}
		if m.active != screenConfig && m.active != screenConfigChoice && m.active != screenConfigInput {
			return m, nil
		}
		return m.applyChildAction(action)
	case loadMsg:
		m.applyLoadMsg(msg)
		if msg.prefetch {
			m.prefetchActive = false
			m.prefetchCurrent = 0
			return m.startNextPrefetch()
		}
	case authCheckMsg:
		m.loading = false
		action, _ := m.auth.Update(msg, m.ctx, m.service)
		m.err = action.err
		if msg.err != nil {
			m.active = screenAuth
			m.err = msg.err
			return m, action.cmd
		}
		if len(msg.users) == 0 {
			m.active = screenAuth
			m.err = nil
			return m, action.cmd
		}
		m.config.UsersLoaded(msg.users)
		m.active = screenHome
		m = m.preparePrefetch(mainPrefetchScreens())
		if !m.prefetchActive {
			return m, nil
		}
		return m, m.loadPrefetch(m.prefetchCurrent, false)
	case authSubmitMsg:
		action, _ := m.auth.Update(msg, m.ctx, m.service)
		m.err = action.err
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.active = screenHome
		m = m.preparePrefetch(mainPrefetchScreens())
		if !m.prefetchActive {
			return m, nil
		}
		return m, m.loadPrefetch(m.prefetchCurrent, false)
	case syncMsg:
		m.loading = false
		if len(msg.conflicts) > 0 {
			m.active = screenSyncConflict
			m.sync.Loaded(msg)
			m.syncStatus = ""
			m.err = nil
			return m, nil
		}
		m.active = screenDashboard
		m.dashboard.resetPage()
		m.syncStatus = msg.status
		m.sync.Loaded(msg)
		if msg.err == nil {
			m.lastSyncAt = time.Now()
		}
		m.err = nil
		m.loadedAt = time.Now()
		return m, tea.Batch(m.load(screenDashboard, false), syncDoneTimeout())
	case syncDoneTimeoutMsg:
		m.sync.ClearPhase()
		m.syncStatus = ""
		m.err = nil
	case statusMsg:
		m.loading = false
		if msg.err != nil {
			status := strings.TrimSpace(msg.status)
			if status == "" {
				status = "작업 실패"
			}
			m.err = fmt.Errorf("%s: %w", status, msg.err)
			m.syncStatus = ""
		} else {
			m.err = nil
			m.syncStatus = ""
		}
		m.loadedAt = time.Now()
	case detailMsg:
		if msg.screen != m.active {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.screen == screenAssignmentDetail {
			m.assignments.DetailLoaded(msg.assignment)
		}
		if msg.screen == screenNoticeDetail {
			m.notices.DetailLoaded(msg.notice)
		}
		m.syncStatus = ""
		m.loadedAt = time.Now()
	case syllabusMsg:
		if m.active != screenSyllabus {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.syllabus.Loaded(msg.result)
		m.loadedAt = time.Now()
	case downloadRowsMsg:
		if m.active != screenDownloadSelect {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.downloadRows = msg.rows
		m.downloadSelected = make(map[string]bool, len(msg.rows))
		m.downloadCourse = 0
		m.downloadCursor = 0
	case roomAvailableResultsMsg:
		if m.active != screenRoomResult {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.room.Loaded(msg.results)
		m.loadedAt = time.Now()
	}
	return m, nil
}

func mainPrefetchScreens() []screen {
	return []screen{
		screenAssignments,
		screenNotices,
		screenLectures,
		screenAcademic,
		screenConfig,
		screenDashboard,
		screenDue,
	}
}

func (m model) preparePrefetch(targets []screen) model {
	if len(targets) == 0 {
		return m
	}
	m.prefetchCurrent = targets[0]
	m.prefetchActive = true
	m.prefetchQueue = append([]screen(nil), targets[1:]...)
	m.markScreenLoading(targets[0])
	return m
}

func (m model) startNextPrefetch() (model, tea.Cmd) {
	for len(m.prefetchQueue) > 0 {
		target := m.prefetchQueue[0]
		m.prefetchQueue = m.prefetchQueue[1:]
		if m.isScreenLoaded(target) || m.isScreenLoading(target) {
			continue
		}
		m.prefetchCurrent = target
		m.prefetchActive = true
		m.markScreenLoading(target)
		return m, m.loadPrefetch(target, false)
	}
	m.prefetchActive = false
	return m, nil
}

func (m model) enterScreen(target screen) (tea.Model, tea.Cmd) {
	m.active = target
	m.err = nil
	m.content = ""
	if m.isScreenLoaded(target) {
		m.loading = false
		if err := m.screenError(target); err != nil {
			m.err = err
		}
		return m, nil
	}
	m.loading = true
	if m.isScreenLoading(target) {
		return m, nil
	}
	m.markScreenLoading(target)
	return m, m.load(target, false)
}

func (m *model) applyLoadMsg(msg loadMsg) {
	m.markScreenLoaded(msg.screen, msg.err)
	switch msg.screen {
	case screenDashboard:
		m.dashboard.Loaded(msg.dashboard, msg.screen == m.active)
	case screenDue:
		m.due.Loaded(msg.due, msg.screen == m.active)
	case screenAssignments:
		m.assignments.Loaded(msg.assignments)
	case screenNotices:
		m.notices.Loaded(msg.notices)
	case screenLectures:
		m.lectures.Loaded(msg.lectures)
	case screenAcademic:
		m.academic.Loaded(msg.academic, msg.screen == m.active, time.Now())
	case screenConfig:
		m.config.Loaded(msg)
	}

	active := msg.screen == m.active
	if active {
		m.loading = false
		m.err = msg.err
		m.content = msg.content
		if msg.screen == screenConfig {
			m.config.clampConfigCursor()
		}
		m.activePager().contentCourse = 0
		m.activePager().contentCursor = 0
		if m.sync.syncPhase == "" {
			m.syncStatus = ""
		}
		m.loadedAt = time.Now()
		return
	}

	if msg.screen == screenConfig {
		m.config.clampConfigCursor()
	}
	if msg.err == nil {
		m.loadedAt = time.Now()
	}
}

func (m model) isScreenLoaded(target screen) bool {
	return m.loadedScreens != nil && m.loadedScreens[target]
}

func (m model) isScreenLoading(target screen) bool {
	return m.loadingScreens != nil && m.loadingScreens[target]
}

func (m model) screenError(target screen) error {
	if m.screenErrors == nil {
		return nil
	}
	return m.screenErrors[target]
}

func (m *model) markScreenLoading(target screen) {
	if m.loadingScreens == nil {
		m.loadingScreens = map[screen]bool{}
	}
	m.loadingScreens[target] = true
}

func (m *model) markScreenLoaded(target screen, err error) {
	if m.loadingScreens == nil {
		m.loadingScreens = map[screen]bool{}
	}
	if m.loadedScreens == nil {
		m.loadedScreens = map[screen]bool{}
	}
	if m.screenErrors == nil {
		m.screenErrors = map[screen]error{}
	}
	delete(m.loadingScreens, target)
	m.screenErrors[target] = err
	m.loadedScreens[target] = true
}

func (m model) isDetailScreen() bool {
	return m.active == screenAssignmentDetail || m.active == screenNoticeDetail
}

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

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
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

func (m *model) moveDetailCursor(delta int) {
	if m.active == screenAssignmentDetail {
		m.assignments.moveDetailCursor(delta, m.width, m.height)
	} else {
		m.notices.moveDetailCursor(delta, m.width, m.height)
	}
}

func (m *model) resetLoadedMainScreens() {
	for _, target := range []screen{screenDashboard, screenDue, screenAssignments, screenNotices, screenLectures, screenAcademic} {
		delete(m.loadedScreens, target)
		delete(m.loadingScreens, target)
		delete(m.screenErrors, target)
	}
	m.dashboard.Reset()
	m.due.Reset()
	m.assignments.Reset()
	m.notices.Reset()
	m.lectures.Reset()
	m.academic.Reset()
	m.activePager().contentCourse = 0
	m.activePager().contentCursor = 0
}

func (m model) View() string {
	width := m.width
	if width <= 0 {
		width = 96
	}
	contentWidth := tuiContentWidth(width)
	switch m.active {
	case screenAuth:
		return appStyle.Render(m.auth.View(contentWidth, m.loading, m.err))
	case screenDownloadSelect:
		return appStyle.Render(m.renderDownloadSelectView(contentWidth))
	case screenDownloadConfirm:
		return appStyle.Render(m.renderDownloadConfirmView(contentWidth))
	case screenDownloadLanguage:
		return appStyle.Render(m.renderDownloadLanguageView(contentWidth))
	case screenDownloadProgress:
		if m.downloadProgress == nil {
			return appStyle.Render(errorStyle.Render("다운로드 상태가 없습니다"))
		}
		return m.downloadProgress.View()
	case screenAttendConfirm:
		return appStyle.Render(m.renderAttendConfirmView(contentWidth))
	case screenAttendProgress:
		if m.attendProgress == nil {
			return appStyle.Render(errorStyle.Render("수강 상태가 없습니다"))
		}
		return m.attendProgress.View()
	case screenConfigChoice:
		return appStyle.Render(m.config.View(contentWidth, m.active, m.err))
	case screenConfigInput:
		return appStyle.Render(m.config.View(contentWidth, m.active, m.err))
	case screenRoomDay:
		return appStyle.Render(m.room.View(contentWidth, m.height, m.active, m.err))
	case screenRoomPeriod:
		return appStyle.Render(m.room.View(contentWidth, m.height, m.active, m.err))
	}
	if m.active == screenHome {
		return appStyle.Render(m.home.View(contentWidth))
	}

	header := m.renderHeader(contentWidth)
	rule := renderRule(contentWidth)
	panel := panelStyle.Width(contentWidth).Render(m.renderPanel(contentWidth))
	footer := m.renderFooterHelp(contentWidth)
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, rule, "", panel, "", footer))
}

func (m model) footerHelp() string {
	if m.sync.syncPhase != "" {
		return "동기화 진행 중  q 종료"
	}
	if m.active == screenDashboard {
		if m.dashboard.dashboardPage > 0 {
			return "←/→ 페이지  ↑↓ 스크롤  p 강의계획서  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←/→ 페이지  ↑↓ 스크롤  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenSyllabus {
		return "↑↓ 스크롤  b/esc Dashboard  r 새로고침  q 종료"
	}
	if m.active == screenDue {
		if m.due.duePage == 3 {
			return "←/→ 페이지  ↑↓ 스크롤  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←/→ 페이지  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenAcademic {
		return "←/→ 월 이동  ↑↓ 일정  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenRoomResult {
		return "←/→ 건물  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenSyncConflict {
		return "←/→ 선택  enter 확정/다음  b/esc 취소  q 종료"
	}
	if m.isDetailScreen() {
		return "↑↓ 스크롤  k KLAS  b/esc 목록  q 종료"
	}
	if m.active == screenLectures {
		return "←/→ 과목  ↑↓ 스크롤  a 수강  k KLAS  d 다운로드  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenConfig {
		return "←/→ 페이지  ↑↓ 선택  enter 선택  [] 변경  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.isCoursePagedScreen() {
		if m.active == screenAssignments {
			return "←/→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
		}
		if m.active == screenNotices {
			return "←/→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  b/esc 뒤로  r 새로고침  q 종료"
		}
		return "←/→ 과목  ↑↓ 스크롤  b/esc 뒤로  r 새로고침  q 종료"
	}
	if m.active == screenRoomResult {
		return "b/esc 뒤로  r 새로고침  q 종료"
	}
	return "b/esc 뒤로  r 새로고침  q 종료"
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

func (m model) renderPanel(width int) string {
	var b strings.Builder
	if subtitle := strings.TrimSpace(screenSubtitle(m.active)); subtitle != "" {
		b.WriteString(mutedStyle.Render(subtitle))
		b.WriteString("\n\n")
	}
	if m.sync.syncPhase != "" {
		b.WriteString(m.sync.StatusView(m.syncStatus))
		return b.String()
	}
	if m.loading {
		b.WriteString(warnBadgeStyle.Render("LOADING"))
		b.WriteString(" 데이터를 불러오는 중입니다\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n")
		return b.String()
	}
	if m.active == screenDashboard {
		b.WriteString(m.renderDashboardPagedPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if m.active == screenSyllabus {
		b.WriteString(m.renderSyllabusPanel(width))
		return b.String()
	}
	if m.active == screenDue {
		b.WriteString(m.renderDuePagedPanel(width))
		return b.String()
	}
	if m.isDetailScreen() {
		b.WriteString(m.renderDetailPanel(width))
		return b.String()
	}
	if m.active == screenAcademic {
		b.WriteString(m.renderAcademicCalendarPanel(width))
		return b.String()
	}
	if m.active == screenRoomResult {
		b.WriteString(m.room.View(width, m.height, m.active, m.err))
		return b.String()
	}
	if m.active == screenSyncConflict {
		b.WriteString(m.sync.View(width))
		return b.String()
	}
	if m.active == screenConfig {
		b.WriteString(m.config.View(width, m.active, m.err))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if m.isCoursePagedScreen() {
		b.WriteString(m.renderCoursePagedPanel(width))
		if m.syncStatus != "" {
			b.WriteString("\n")
			b.WriteString(footerStyle.Render(m.syncStatus))
			b.WriteString("\n")
		}
		return b.String()
	}
	if strings.TrimSpace(m.content) == "" {
		b.WriteString(emptyStyle.Render("표시할 내용이 없습니다"))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(m.content)
	if !strings.HasSuffix(m.content, "\n") {
		b.WriteString("\n")
	}
	if m.syncStatus != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(m.syncStatus))
		b.WriteString("\n")
	}
	return b.String()
}

func conflictLabel(conflict app.SyncConflict) string {
	switch conflict.Scope {
	case "assignment":
		return "과제 미리알림"
	case "lecture":
		return "강의 미리알림"
	case "academic":
		return "학사일정 캘린더"
	case "timetable":
		return "시간표 캘린더"
	default:
		return conflict.Scope
	}
}

func (m model) renderDetailPanel(width int) string {
	if m.active == screenAssignmentDetail {
		return m.assignments.View(width, m.height, true, m.syncStatus)
	}
	return m.notices.View(width, m.height, true, m.syncStatus)
}

func (m model) visibleBodyRows(reserved int) int { return visibleBodyRows(m.height, reserved) }

func visibleBodyRows(height, reserved int) int {
	if height <= 0 {
		return 16
	}
	return maxInt(4, height-11-reserved)
}

func (m model) detailLines(width int) []string {
	switch m.active {
	case screenAssignmentDetail:
		return assignmentDetailLines(m.assignments.assignmentDetail, width)
	case screenNoticeDetail:
		return noticeDetailLines(m.notices.noticeDetail, width)
	default:
		return nil
	}
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func appendWrappedLines(lines []string, text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return lines
	}
	wrapWidth := maxInt(24, minInt(100, width-6))
	for _, paragraph := range strings.Split(text, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		rendered := lipgloss.NewStyle().Width(wrapWidth).Render(paragraph)
		lines = append(lines, strings.Split(rendered, "\n")...)
	}
	return lines
}

func splitRenderedLines(value string) []string {
	return strings.Split(strings.TrimRight(value, "\n"), "\n")
}

func (m model) load(target screen, refresh bool) tea.Cmd {
	return m.loadWithPrefetch(target, refresh, false)
}

func (m model) loadPrefetch(target screen, refresh bool) tea.Cmd {
	return m.loadWithPrefetch(target, refresh, true)
}

func (m model) loadWithPrefetch(target screen, refresh bool, prefetch bool) tea.Cmd {
	if target == screenLectures {
		return loadLectures(m.ctx, m.service, refresh, prefetch)
	}
	if target == screenAcademic {
		return loadAcademic(m.ctx, m.service, refresh, prefetch)
	}
	if target == screenDue {
		return loadDue(m.ctx, m.service, refresh, prefetch)
	}
	if target == screenDashboard {
		return loadDashboard(m.ctx, m.service, refresh, prefetch)
	}
	if target == screenNotices {
		return loadNotices(m.ctx, m.service, refresh, prefetch)
	}
	if target == screenAssignments {
		return loadAssignments(m.ctx, m.service, refresh, prefetch)
	}
	return func() tea.Msg {
		switch target {
		case screenConfig:
			msg := loadConfigMsg(m.ctx, m.service)
			msg.screen = target
			msg.prefetch = prefetch
			return msg
		}
		content, err := m.loadContent(target, refresh)
		return loadMsg{screen: target, prefetch: prefetch, content: content, err: err}
	}
}

func (m model) loadContent(target screen, refresh bool) (string, error) {
	switch target {
	case screenDashboard:
		return "", errors.New("Dashboard는 structured loader를 사용해야 합니다")
	default:
		return "", errors.New("지원하지 않는 TUI 화면입니다")
	}
}

func screenTitle(value screen) string {
	switch value {
	case screenAuth:
		return "Account"
	case screenDashboard:
		return "Dashboard"
	case screenDue:
		return "Due"
	case screenAssignments:
		return "Assignments"
	case screenNotices:
		return "Notices"
	case screenLectures:
		return "Lectures"
	case screenAssignmentDetail:
		return "Assignment"
	case screenNoticeDetail:
		return "Notice"
	case screenSyllabus:
		return "Syllabus"
	case screenAcademic:
		return "Academic"
	case screenConfig:
		return "Config"
	case screenConfigChoice:
		return "Config"
	case screenConfigInput:
		return "Config"
	case screenRoomDay, screenRoomPeriod, screenRoomResult:
		return "Rooms"
	case screenSyncConflict:
		return "Sync"
	case screenDownloadSelect:
		return "Download"
	case screenDownloadConfirm:
		return "Transcript"
	case screenDownloadLanguage:
		return "Transcript"
	case screenDownloadProgress:
		return "Download"
	case screenAttendConfirm, screenAttendProgress:
		return "Attend"
	default:
		return "Home"
	}
}

func screenSubtitle(value screen) string {
	switch value {
	case screenDashboard:
		return "과제, 온라인 강의, 공지, 출석, 수업평가 요약"
	case screenDue:
		return "다가오는 과제, 온라인 강의, 학사일정"
	case screenAssignments:
		return "현재 학기 과제 목록"
	case screenNotices:
		return "최근 강의 공지"
	case screenLectures:
		return "온라인 강의와 학습활동 상태"
	case screenAssignmentDetail:
		return "과제 상세"
	case screenNoticeDetail:
		return "공지 상세"
	case screenSyllabus:
		return "선택한 과목의 강의계획서"
	case screenAcademic:
		return "학사일정 달력"
	case screenConfig:
		return "현재 유저 설정"
	case screenConfigChoice:
		return "설정 항목 선택"
	case screenConfigInput:
		return "설정 값 입력"
	case screenRoomResult:
		return "선택한 요일과 교시에 비어 있는 강의실"
	case screenSyncConflict:
		return "KLAS 변경사항 반영 여부 선택"
	default:
		return ""
	}
}

func enabledLabel(value bool) string {
	if value {
		return "켜짐"
	}
	return "꺼짐"
}

func formatMinutes(minutes int) string {
	if minutes%1440 == 0 {
		days := minutes / 1440
		if days == 1 {
			return "1일"
		}
		return fmt.Sprintf("%d일", days)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%d시간", minutes/60)
	}
	return fmt.Sprintf("%d분", minutes)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}

func renderSection(title string, lines []string) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(title))
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func formatSectionErrors(errors []app.DashboardSectionError) string {
	lines := make([]string, 0, len(errors))
	for _, sectionError := range errors {
		lines = append(lines, errorStyle.Render(sectionError.Section)+" "+sectionError.Err.Error())
	}
	return renderSection("확인 실패", lines)
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
