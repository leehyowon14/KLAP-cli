package tui

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestEnterScreenUsesPrefetchedData(t *testing.T) {
	m := model{
		loadedScreens: map[screen]bool{screenAssignments: true},
		assignments: assignmentScreenModel{assignmentRows: []app.AssignmentRow{
			{ID: "1", CourseName: "오픈소스소프트웨어실습"},
		}},
	}
	updated, cmd := m.enterScreen(screenAssignments)
	got := updated.(model)
	if cmd != nil {
		t.Fatal("enterScreen() returned load command for prefetched screen")
	}
	if got.active != screenAssignments || got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
	if len(got.assignments.assignmentRows) != 1 {
		t.Fatalf("assignmentRows = %+v", got.assignments.assignmentRows)
	}
}

func TestEnterScreenShowsLoadingForPendingPrefetch(t *testing.T) {
	m := model{
		loadingScreens: map[screen]bool{screenLectures: true},
	}
	updated, cmd := m.enterScreen(screenLectures)
	got := updated.(model)
	if cmd != nil {
		t.Fatal("enterScreen() started duplicate load for pending prefetch")
	}
	if got.active != screenLectures || !got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
}

func TestInactiveLoadMsgCachesWithoutClobberingOtherScreens(t *testing.T) {
	m := model{
		active: screenHome,
		assignments: assignmentScreenModel{assignmentRows: []app.AssignmentRow{
			{ID: "1", CourseName: "컴퓨터그래픽스"},
		}},
	}
	m.applyLoadMsg(loadMsg{
		screen: screenLectures,
		lectures: []app.LectureRow{
			{ID: "lecture-1", CourseName: "오픈소스소프트웨어실습"},
		},
	})
	if !m.loadedScreens[screenLectures] || m.loading {
		t.Fatalf("loadedScreens=%+v loading=%t", m.loadedScreens, m.loading)
	}
	if len(m.lectures.lectureRows) != 1 {
		t.Fatalf("lectureRows = %+v", m.lectures.lectureRows)
	}
	if len(m.assignments.assignmentRows) != 1 || m.assignments.assignmentRows[0].ID != "1" {
		t.Fatalf("assignmentRows clobbered: %+v", m.assignments.assignmentRows)
	}
}

func TestPreparePrefetchStartsOnlyFirstScreen(t *testing.T) {
	m := (model{loadingScreens: map[screen]bool{}}).preparePrefetch([]screen{screenAssignments, screenLectures, screenDashboard})
	if !m.prefetchActive || m.prefetchCurrent != screenAssignments {
		t.Fatalf("prefetch active=%t current=%v", m.prefetchActive, m.prefetchCurrent)
	}
	if !m.loadingScreens[screenAssignments] || m.loadingScreens[screenLectures] || m.loadingScreens[screenDashboard] {
		t.Fatalf("loadingScreens = %+v", m.loadingScreens)
	}
	if len(m.prefetchQueue) != 2 || m.prefetchQueue[0] != screenLectures || m.prefetchQueue[1] != screenDashboard {
		t.Fatalf("prefetchQueue = %+v", m.prefetchQueue)
	}
}

func TestPrefetchLoadMsgStartsNextQueuedScreen(t *testing.T) {
	m := model{
		active:          screenHome,
		loadedScreens:   map[screen]bool{},
		loadingScreens:  map[screen]bool{screenAssignments: true},
		screenErrors:    map[screen]error{},
		prefetchActive:  true,
		prefetchCurrent: screenAssignments,
		prefetchQueue:   []screen{screenLectures},
	}
	updated, cmd := m.Update(loadMsg{screen: screenAssignments, prefetch: true,
		assignments: []app.AssignmentRow{{ID: "1"}}})
	got := updated.(model)
	if cmd == nil {
		t.Fatal("next prefetch command is nil")
	}
	if !got.loadedScreens[screenAssignments] || got.loadingScreens[screenAssignments] {
		t.Fatalf("assignments loaded/loading = %+v/%+v", got.loadedScreens, got.loadingScreens)
	}
	if !got.prefetchActive || got.prefetchCurrent != screenLectures || !got.loadingScreens[screenLectures] {
		t.Fatalf("next prefetch active=%t current=%v loading=%+v", got.prefetchActive, got.prefetchCurrent, got.loadingScreens)
	}
}

func TestPrefetchErrorIsStoredAsLoadedError(t *testing.T) {
	errBoom := errors.New("boom")
	m := model{}
	m.markScreenLoaded(screenLectures, errBoom)
	if !m.isScreenLoaded(screenLectures) {
		t.Fatal("failed screen should be considered loaded")
	}
	if got := m.screenError(screenLectures); got == nil || got.Error() != "boom" {
		t.Fatalf("screenError() = %v", got)
	}

	updated, cmd := m.enterScreen(screenLectures)
	got := updated.(model)
	if cmd != nil || got.loading || got.err == nil {
		t.Fatalf("enter failed screen cmd=%v loading=%t err=%v", cmd, got.loading, got.err)
	}
}
