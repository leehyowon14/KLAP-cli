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

type model struct {
	ctx             context.Context
	service         *app.Service
	home            homeModel
	active          screen
	loading         bool
	loadedScreens   map[screen]bool
	loadingScreens  map[screen]bool
	screenErrors    map[screen]error
	prefetchQueue   []screen
	prefetchCurrent screen
	prefetchActive  bool
	err             error
	content         string
	dashboard       dashboardScreenModel
	assignments     assignmentScreenModel
	notices         noticeScreenModel
	lectures        lectureScreenModel
	due             dueScreenModel
	academic        academicScreenModel
	syllabus        syllabusScreenModel
	detailBack      screen
	syncStatus      string
	width           int
	height          int
	loadedAt        time.Time
	lastSyncAt      time.Time
	config          configScreenModel
	download        downloadScreenModel
	attend          attendScreenModel
	room            roomScreenModel
	sync            syncScreenModel
	auth            authScreenModel
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
	if action, handled := m.download.Lifecycle(msg); handled {
		return m.applyChildAction(action)
	}
	switch value := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = value.Width, value.Height
		if action, handled := m.updateActiveChild(msg); handled {
			return m.applyChildAction(action)
		}
		return m, nil
	case loadMsg:
		m.applyLoadMsg(value)
		if value.prefetch {
			m.prefetchActive = false
			m.prefetchCurrent = 0
			return m.startNextPrefetch()
		}
		return m, nil
	}
	// Progress children own their run events and cancellation. Window size and
	// background loads above remain global even while a run is visible.
	if m.active == screenDownloadProgress || m.active == screenAttendProgress {
		action, _ := m.updateActiveChild(msg)
		return m.applyChildAction(action)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if action, handled := m.updateActiveChild(msg); handled {
			return m.applyChildAction(action)
		}
		return m.updateGlobalKey(msg)
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
		m.download.Loaded(msg.rows)
	case roomAvailableResultsMsg:
		if m.active != screenRoomResult {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.room.Loaded(msg.results)
		m.loadedAt = time.Now()
	default:
		if action, handled := m.updateActiveChild(msg); handled {
			return m.applyChildAction(action)
		}
	}
	return m, nil
}
