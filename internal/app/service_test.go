package app

import (
	"strings"
	"testing"
	"time"

	"github.com/kw-klap/klap-cli/internal/klas"
)

func TestSelectedCoursesByNumber(t *testing.T) {
	term := klas.Term{Courses: []klas.Course{
		{Name: "프로그래밍기초", Value: "c1"},
		{Name: "자료구조", Value: "c2"},
	}}

	selected, err := selectedCourses(term, "2")
	if err != nil {
		t.Fatalf("selectedCourses() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Index != 2 || selected[0].Course.Name != "자료구조" {
		t.Fatalf("selectedCourses() = %+v", selected)
	}
}

func TestSelectedCoursesByName(t *testing.T) {
	term := klas.Term{Courses: []klas.Course{
		{Name: "프로그래밍기초", Value: "c1"},
		{Name: "자료구조", Value: "c2"},
	}}

	selected, err := selectedCourses(term, "자료")
	if err != nil {
		t.Fatalf("selectedCourses() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Index != 2 {
		t.Fatalf("selectedCourses() = %+v", selected)
	}
}

func TestSelectTermRow(t *testing.T) {
	rows := []TermRow{
		{Index: 1, Term: klas.Term{Label: "2026년도 1학기", Value: "2026,1"}},
		{Index: 2, Term: klas.Term{Label: "2025년도 겨울학기", Value: "2025,4"}},
	}

	selected, err := selectTermRow(rows, "2")
	if err != nil {
		t.Fatalf("selectTermRow() by number error = %v", err)
	}
	if selected.Term.Value != "2025,4" {
		t.Fatalf("selectTermRow() by number = %+v", selected)
	}

	selected, err = selectTermRow(rows, "2026,1")
	if err != nil {
		t.Fatalf("selectTermRow() by value error = %v", err)
	}
	if selected.Term.Label != "2026년도 1학기" {
		t.Fatalf("selectTermRow() by value = %+v", selected)
	}

	selected, err = selectTermRow(rows, "겨울")
	if err != nil {
		t.Fatalf("selectTermRow() by label error = %v", err)
	}
	if selected.Term.Value != "2025,4" {
		t.Fatalf("selectTermRow() by label = %+v", selected)
	}
}

func TestParseAssignmentID(t *testing.T) {
	courseIndex, ordSeq, err := ParseAssignmentID("3:7")
	if err != nil {
		t.Fatalf("ParseAssignmentID() error = %v", err)
	}
	if courseIndex != 3 || ordSeq != "7" {
		t.Fatalf("ParseAssignmentID() = %d, %q", courseIndex, ordSeq)
	}
}

func TestParseNoticeID(t *testing.T) {
	courseIndex, boardNo, masterNo, err := ParseNoticeID("7:1161280:1000000")
	if err != nil {
		t.Fatalf("ParseNoticeID() error = %v", err)
	}
	if courseIndex != 7 || boardNo != "1161280" || masterNo != "1000000" {
		t.Fatalf("ParseNoticeID() = %d, %q, %q", courseIndex, boardNo, masterNo)
	}
}

func TestParseLectureID(t *testing.T) {
	courseIndex, contentID, err := ParseLectureID("7:content-123")
	if err != nil {
		t.Fatalf("ParseLectureID() error = %v", err)
	}
	if courseIndex != 7 || contentID != "content-123" {
		t.Fatalf("ParseLectureID() = %d, %q", courseIndex, contentID)
	}
}

func TestLectureFilenameSanitizesPathComponents(t *testing.T) {
	got := lectureFilename("오픈소스/실습", klas.Lecture{
		ModuleTitle: "1주차: 소개",
		Title:       "Git? GitHub* 시작",
	}, "https://media.example.com/video.mp4?token=1")
	want := "오픈소스_실습_1주차_ 소개_Git_ GitHub_ 시작.mp4"
	if got != want {
		t.Fatalf("lectureFilename() = %q, want %q", got, want)
	}
}

func TestLectureNeedsAttendance(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	if !lectureNeedsAttendance(klas.Lecture{ContentID: "content", Progress: "20", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected true")
	}
	if lectureNeedsAttendance(klas.Lecture{ContentID: "content", Progress: "100", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false for completed lecture")
	}
	if lectureNeedsAttendance(klas.Lecture{ContentID: "", Progress: "20", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false without content id")
	}
	if !lectureNeedsAttendance(klas.Lecture{LearningSeq: "15", AchievedTime: "0", RequiredTime: "10", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected true for incomplete learning activity")
	}
	if lectureNeedsAttendance(klas.Lecture{LearningSeq: "15", AchievedTime: "10", RequiredTime: "10", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false for completed learning activity")
	}
}

func TestBuildReminderNotesIncludesBodyAndMarker(t *testing.T) {
	dueAt := time.Date(2026, 5, 5, 23, 59, 0, 0, time.FixedZone("KST", 9*60*60))
	notes := buildReminderNotes(AssignmentDetailResult{
		ID:         "3:1",
		TermValue:  "2026,1",
		CourseName: "컴퓨터그래픽스",
		Detail: klas.AssignmentDetail{
			Title:          "과제1",
			ContentText:    "과제 skeleton code 구성 가이드",
			DueAt:          &dueAt,
			Submitted:      true,
			ReportType:     "개인",
			SubmitFileType: "zip",
			FileLimitMB:    "10",
		},
	})

	for _, want := range []string{
		"과제 skeleton code 구성 가이드",
		"--- KLAP ---",
		"ID: 3:1",
		"과목: 컴퓨터그래픽스",
		"제목: 과제1",
		"마감: 2026-05-05 23:59",
		"상태: 제출",
		"#2026-1 #컴퓨터그래픽스",
		"[This reminder is created by KLAP.]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes does not contain %q:\n%s", want, notes)
		}
	}
}

func TestReminderHashtags(t *testing.T) {
	got := reminderHashtags("2026,1", "오픈소스 소프트웨어 실습")
	want := "#2026-1 #오픈소스소프트웨어실습"
	if got != want {
		t.Fatalf("reminderHashtags() = %q, want %q", got, want)
	}
}

func TestTermHashtagSeasonSemesters(t *testing.T) {
	if got := termHashtag("2026,3"); got != "#2026-여름학기" {
		t.Fatalf("termHashtag() summer = %q", got)
	}
	if got := termHashtag("2026,4"); got != "#2026-겨울학기" {
		t.Fatalf("termHashtag() winter = %q", got)
	}
}

func TestParseAcademicEvents(t *testing.T) {
	body := []byte(`
		<!--
		<h3>2025 학사일정</h3>
		<table><tbody><tr><td>3월</td><td>1(토)</td><td>오래된 일정</td><td></td></tr></tbody></table>
		-->
		<h3>2026 학사일정</h3>
		<table><tbody>
			<tr><td rowspan="2">3월</td><td>3(화)</td><td>2026학년도 1학기 개강(학기개시일)</td><td>3월2일(월) - 대체공휴일</td></tr>
			<tr><td>28(토)</td><td>수업일수 4분의 1</td><td>15주 기준</td></tr>
			<tr><td>8월</td><td>3(월)~31(월)</td><td>2학기 복학신청</td></tr>
		</tbody></table>
		<h3>2027 학사일정</h3>
		<table><tbody>
			<tr><td>3월</td><td>2(화)</td><td>2027학년도 1학기 개강</td><td></td></tr>
		</tbody></table>
	`)

	events, err := parseAcademicEvents(body, "2026")
	if err != nil {
		t.Fatalf("parseAcademicEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("parseAcademicEvents() len = %d, want 3: %+v", len(events), events)
	}
	if events[0].Month != "3월" || events[0].Date != "3(화)" || events[0].Title != "2026학년도 1학기 개강(학기개시일)" {
		t.Fatalf("parseAcademicEvents()[0] = %+v", events[0])
	}
	if events[1].Month != "3월" || events[1].Date != "28(토)" || events[1].Note != "15주 기준" {
		t.Fatalf("parseAcademicEvents()[1] = %+v", events[1])
	}
	if events[2].Month != "8월" || events[2].Date != "3(월)~31(월)" || events[2].Title != "2학기 복학신청" {
		t.Fatalf("parseAcademicEvents()[2] = %+v", events[2])
	}
}
