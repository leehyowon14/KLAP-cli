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

func (m model) updateGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
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

func (m model) isDetailScreen() bool {
	return m.active == screenAssignmentDetail || m.active == screenNoticeDetail
}

func (m *model) moveDetailCursor(delta int) {
	if m.active == screenAssignmentDetail {
		m.assignments.moveDetailCursor(delta, m.width, m.height)
	} else {
		m.notices.moveDetailCursor(delta, m.width, m.height)
	}
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
