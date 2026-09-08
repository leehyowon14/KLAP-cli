package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func TestRootProgressScreensPreserveGlobalResizeAndPrefetch(t *testing.T) {
	for _, route := range []screen{screenDownloadProgress, screenAttendProgress} {
		m := model{active: route, prefetchActive: true, prefetchCurrent: screenAssignments, prefetchQueue: []screen{screenNotices},
			download: downloadScreenModel{downloadProgress: &lectureDownloadModel{}},
			attend:   attendScreenModel{attendProgress: &lectureAttendModel{}},
		}
		next, _ := m.Update(tea.WindowSizeMsg{Width: 101, Height: 33})
		m = next.(model)
		if m.width != 101 || m.height != 33 {
			t.Fatal("root missed resize")
		}
		if route == screenDownloadProgress && (m.download.downloadProgress.width != 101 || m.download.downloadProgress.height != 33) {
			t.Fatal("download child missed resize")
		}
		if route == screenAttendProgress && (m.attend.attendProgress.width != 101 || m.attend.attendProgress.height != 33) {
			t.Fatal("attend child missed resize")
		}
		next, cmd := m.Update(loadMsg{screen: screenAssignments, prefetch: true, assignments: []app.AssignmentRow{{ID: "background"}}})
		m = next.(model)
		if m.active != route || len(m.assignments.assignmentRows) != 1 || !m.isScreenLoaded(screenAssignments) || m.prefetchCurrent != screenNotices || cmd == nil {
			t.Fatal("progress screen swallowed background load or changed route")
		}
	}
}

func TestRootGlobalQuitAndChildKeyPrecedence(t *testing.T) {
	m := model{active: screenHome}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("global quit lost")
	}
	m = model{active: screenDue, due: dueScreenModel{duePage: 1}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if next.(model).due.duePage != 0 {
		t.Fatal("child key not delegated")
	}
	next, _ = next.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(model).active != screenHome {
		t.Fatal("global back lost")
	}
}
