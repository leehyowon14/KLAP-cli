package tui

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"time"
)

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
		switch msg.screen {
		case screenAssignments:
			m.assignments.ResetListPosition()
		case screenNotices:
			m.notices.ResetListPosition()
		case screenLectures:
			m.lectures.ResetListPosition()
		}
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
