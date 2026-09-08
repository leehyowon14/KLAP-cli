package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

func TestLectureAttendShortcutOpensConfirmation(t *testing.T) {
	m := model{
		active: screenLectures,
		width:  96, lectures: lectureScreenModel{lectureRows: []app.LectureRow{{
			ID:         "1:video",
			CourseName: "운영체제",
			Lecture: app.Lecture{
				ContentID: "video",
				Title:     "프로세스",
				Progress:  "25",
			},
		}}},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(model)
	if cmd != nil {
		t.Fatal("attend confirmation should not start attendance")
	}
	if got.active != screenAttendConfirm || got.attend.attendRow.ID != "1:video" {
		t.Fatalf("active=%v attendRow=%+v", got.active, got.attend.attendRow)
	}
	if !strings.Contains(got.View(), "이 강의를 수강할까요?") {
		t.Fatalf("confirmation view = %q", got.View())
	}
}

func TestLectureAttendShortcutAcceptsKoreanKeyboardKey(t *testing.T) {
	m := model{
		active: screenLectures, lectures: lectureScreenModel{lectureRows: []app.LectureRow{{
			ID:      "1:video",
			Lecture: app.Lecture{ContentID: "video", Progress: "25"},
		}}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅁ")})
	if got := updated.(model); got.active != screenAttendConfirm {
		t.Fatalf("active = %v, want screenAttendConfirm", got.active)
	}
}
