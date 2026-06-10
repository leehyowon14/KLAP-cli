package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/settings"
)

var errDashboardTest = errors.New("dashboard test error")

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

func TestNormalizeTermValue(t *testing.T) {
	got, err := normalizeTermValue("2026-1")
	if err != nil {
		t.Fatalf("normalizeTermValue() error = %v", err)
	}
	if got != "2026,1" {
		t.Fatalf("normalizeTermValue() = %q", got)
	}
	if _, err := normalizeTermValue("2026-5"); err == nil {
		t.Fatal("normalizeTermValue() expected error for invalid semester")
	}
}

func TestCurrentAcademicTermValue(t *testing.T) {
	tests := []struct {
		now  time.Time
		want string
	}{
		{now: time.Date(2026, time.June, 7, 0, 0, 0, 0, time.Local), want: "2026,1"},
		{now: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.Local), want: "2026,3"},
		{now: time.Date(2026, time.December, 20, 0, 0, 0, 0, time.Local), want: "2026,4"},
		{now: time.Date(2027, time.January, 10, 0, 0, 0, 0, time.Local), want: "2026,4"},
	}
	for _, tt := range tests {
		if got := currentAcademicTermValue(tt.now); got != tt.want {
			t.Fatalf("currentAcademicTermValue(%s) = %q, want %q", tt.now, got, tt.want)
		}
	}
}

func TestFutureTime(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
	past := now.Add(-time.Minute)
	same := now
	future := now.Add(time.Minute)

	if futureTime(nil, now) {
		t.Fatal("futureTime(nil) should be false")
	}
	if futureTime(&past, now) {
		t.Fatal("past deadline should not be synced")
	}
	if !futureTime(&same, now) || !futureTime(&future, now) {
		t.Fatal("current or future deadline should be synced")
	}
}

func TestTermLabel(t *testing.T) {
	if got := termLabel("2026,3"); got != "2026년도 여름학기" {
		t.Fatalf("termLabel() = %q", got)
	}
	if got := termLabel("2026,4"); got != "2026년도 겨울학기" {
		t.Fatalf("termLabel() = %q", got)
	}
}

func TestLooksLikeSyllabusCourseCode(t *testing.T) {
	if !looksLikeSyllabusCourseCode("I040-3-3951-01") {
		t.Fatal("looksLikeSyllabusCourseCode() expected true")
	}
	if looksLikeSyllabusCourseCode("컴퓨터그래픽스") {
		t.Fatal("looksLikeSyllabusCourseCode() expected false")
	}
}

func TestCopyWithProgressReportsOffsetAndTotal(t *testing.T) {
	var dst bytes.Buffer
	var events []int64
	written, err := copyWithProgress(&dst, strings.NewReader("abcdef"), 4, 10, func(bytesWritten int64, totalBytes int64) {
		if totalBytes != 10 {
			t.Fatalf("totalBytes = %d, want 10", totalBytes)
		}
		events = append(events, bytesWritten)
	})
	if err != nil {
		t.Fatalf("copyWithProgress() error = %v", err)
	}
	if written != 6 || dst.String() != "abcdef" {
		t.Fatalf("copyWithProgress() = %d, %q", written, dst.String())
	}
	if len(events) == 0 || events[len(events)-1] != 10 {
		t.Fatalf("progress events = %v, want final 10", events)
	}
}

func TestResetConfigSettingsRestoresDefaults(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.SetConfigValue("reminder.name", "To-do"); err != nil {
		t.Fatalf("SetConfigValue(reminder.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.concurrency", "9"); err != nil {
		t.Fatalf("SetConfigValue(download.concurrency) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.caffeinate", "false"); err != nil {
		t.Fatalf("SetConfigValue(download.caffeinate) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.keep-partial", "true"); err != nil {
		t.Fatalf("SetConfigValue(download.keep-partial) error = %v", err)
	}
	if _, err := service.SetConfigValue("transcript.concurrency", "3"); err != nil {
		t.Fatalf("SetConfigValue(transcript.concurrency) error = %v", err)
	}

	got, err := service.ResetConfigSettings()
	if err != nil {
		t.Fatalf("ResetConfigSettings() error = %v", err)
	}
	if got.Reminder.ListName != settings.DefaultReminderListName {
		t.Fatalf("Reminder.ListName = %q", got.Reminder.ListName)
	}
	if got.Download.Dir != settings.DefaultDownloadDir() || got.Download.Concurrency != settings.DefaultDownloadConcurrency {
		t.Fatalf("Download = %+v", got.Download)
	}
	if !got.Download.Caffeinate {
		t.Fatal("Download.Caffeinate should reset to true")
	}
	if got.Download.KeepPartial {
		t.Fatal("Download.KeepPartial should reset to false")
	}
	if got.Transcript.Concurrency != settings.DefaultTranscriptConcurrency {
		t.Fatalf("Transcript.Concurrency = %d", got.Transcript.Concurrency)
	}
	if got.Term.Value != "" {
		t.Fatalf("Term.Value = %q", got.Term.Value)
	}
}

func TestSetConfigValueRejectsInvalidTranscriptConcurrency(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.SetConfigValue("transcript.concurrency", "4"); err == nil {
		t.Fatal("SetConfigValue(transcript.concurrency) expected error")
	}
}

func TestTranscriptPath(t *testing.T) {
	if got := transcriptPath(filepath.Join("downloads", "컴퓨터그래픽스", "video", "lecture.mp4")); got != filepath.Join("downloads", "컴퓨터그래픽스", "transcription", "lecture.txt") {
		t.Fatalf("transcriptPath() = %q", got)
	}
	if got := transcriptPath(filepath.Join("downloads", "lecture")); got != filepath.Join("downloads", "lecture.txt") {
		t.Fatalf("transcriptPath() without extension = %q", got)
	}
}

func TestLectureDownloadItemNeedsTranscriptForSkippedVideo(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "컴퓨터그래픽스", "video", "lecture.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(video) error = %v", err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	item := LectureDownloadItem{Path: videoPath, Skipped: true}
	if !LectureDownloadItemNeedsTranscript(item) {
		t.Fatal("LectureDownloadItemNeedsTranscript() expected true without transcript")
	}

	outputPath := filepath.Join(root, "컴퓨터그래픽스", "transcription", "lecture.txt")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(transcript) error = %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("text"), 0o644); err != nil {
		t.Fatalf("WriteFile(transcript) error = %v", err)
	}
	if LectureDownloadItemNeedsTranscript(item) {
		t.Fatal("LectureDownloadItemNeedsTranscript() expected false with existing transcript")
	}
}

func TestTranscribeDownloadedLecturesReturnsPreflightFailure(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "컴퓨터그래픽스")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile(blocker) error = %v", err)
	}
	videoPath := filepath.Join(blocker, "video", "lecture.mp4")
	row := LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "a", Title: "소개"}}

	result := (&Service{}).TranscribeDownloadedLectures(context.Background(), []LectureDownloadItem{{
		Lecture: row,
		Path:    videoPath,
	}}, LectureTranscriptOptions{})

	if len(result.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(result.Items))
	}
	if result.Items[0].Err == nil {
		t.Fatalf("Items[0].Err is nil")
	}
	if result.Items[0].InputPath != videoPath || result.Items[0].Lecture.CourseName != "컴퓨터그래픽스" {
		t.Fatalf("Items[0] = %+v", result.Items[0])
	}
}

func TestTranscriptBridgeCandidatesPreferBuiltBinary(t *testing.T) {
	got := transcriptBridgeCandidates("bridges/macos")
	want := []string{
		filepath.Join("bridges", "macos", ".build", "release", "TranscriptBridge"),
		filepath.Join("bridges", "macos", ".build", "debug", "TranscriptBridge"),
		filepath.Join("bridges", "macos", "transcribe.swift"),
	}
	if len(got) != len(want) {
		t.Fatalf("transcriptBridgeCandidates() length = %d, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("transcriptBridgeCandidates()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestLectureTranscriptContextIncludesCourseAndCodeSwitching(t *testing.T) {
	got := lectureTranscriptContext(LectureRow{
		CourseName: "컴퓨터그래픽스",
		Lecture: klas.Lecture{
			ModuleTitle: "14주차",
			Title:       "렌더링 파이프라인",
		},
	})
	for _, want := range []string{"광운대학교", "컴퓨터그래픽스", "14주차", "렌더링 파이프라인", "code switching"} {
		if !containsString(got, want) {
			t.Fatalf("lectureTranscriptContext() missing %q: %v", want, got)
		}
	}
}

func TestCleanupPartialDownloadRespectsKeepPartial(t *testing.T) {
	dir := t.TempDir()
	removePath := filepath.Join(dir, "remove.part")
	keepPath := filepath.Join(dir, "keep.part")
	if err := os.WriteFile(removePath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(remove) error = %v", err)
	}
	if err := os.WriteFile(keepPath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(keep) error = %v", err)
	}

	cleanupPartialDownload(removePath, false)
	if _, err := os.Stat(removePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removePath stat error = %v, want not exist", err)
	}
	cleanupPartialDownload(keepPath, true)
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("keepPath stat error = %v", err)
	}
}

func TestDownloadStatusReportsPartialFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4"), []byte("done"), 0o644); err != nil {
		t.Fatalf("WriteFile(done) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4.part"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(partial) error = %v", err)
	}

	got, err := (&Service{}).DownloadStatus(dir)
	if err != nil {
		t.Fatalf("DownloadStatus() error = %v", err)
	}
	if got.Files != 1 || got.PartialFiles != 1 || got.PartialBytes != int64(len("partial")) {
		t.Fatalf("DownloadStatus() = %+v", got)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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

func TestLectureRowIDMatchesLearningSeqFallback(t *testing.T) {
	lecture := klas.Lecture{LearningSeq: "39769"}
	if got := lectureRowID(1, lecture); got != "1:lrn-39769" {
		t.Fatalf("lectureRowID() = %q", got)
	}
	lecture.ContentID = "content-123"
	if got := lectureRowID(1, lecture); got != "1:content-123" {
		t.Fatalf("lectureRowID() content = %q", got)
	}
}

func TestLectureFilenameSanitizesPathComponents(t *testing.T) {
	got := lectureVideoPath("root", "오픈소스/실습", klas.Lecture{
		ModuleTitle: "1주차: 소개",
		Title:       "Git? GitHub* 시작",
	}, "https://media.example.com/video.mp4?token=1", 1)
	want := filepath.Join("root", "오픈소스_실습", "video", "1-1. Git_ GitHub_ 시작.mp4")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureFilenameUsesKlasWeekSequence(t *testing.T) {
	got := lectureVideoPath("root", "컴퓨터그래픽스", klas.Lecture{
		WeekNo:      "14",
		WeeklySeq:   "2",
		ModuleTitle: "보강",
		Title:       "기말/정리",
	}, "https://media.example.com/final.mov", 2)
	want := filepath.Join("root", "컴퓨터그래픽스", "video", "14-2. 기말_정리.mov")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureWeekOrderFallsBackToCourseOrder(t *testing.T) {
	rows := []LectureRow{
		{ID: "1:a", Lecture: klas.Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "첫번째"}},
		{ID: "1:b", Lecture: klas.Lecture{ContentID: "b", ModuleTitle: "1주차", Title: "두번째"}},
		{ID: "1:c", Lecture: klas.Lecture{ContentID: "c", ModuleTitle: "2주차", Title: "첫번째"}},
	}
	orders := lectureRowWeekOrders(rows)
	if orders["1:a"] != 1 || orders["1:b"] != 2 || orders["1:c"] != 1 {
		t.Fatalf("lectureRowWeekOrders() = %#v", orders)
	}
}

func TestAcademicEventDueAt(t *testing.T) {
	got, ok := academicEventDueAt(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.17(수)",
		Title: "기말고사",
	})
	if !ok {
		t.Fatal("academicEventDueAt() expected ok")
	}
	if got.Year() != 2026 || got.Month() != time.June || got.Day() != 17 {
		t.Fatalf("academicEventDueAt() = %s", got)
	}
}

func TestParseConfigBool(t *testing.T) {
	got, err := parseConfigBool("yes")
	if err != nil {
		t.Fatalf("parseConfigBool() error = %v", err)
	}
	if !got {
		t.Fatal("parseConfigBool() expected true")
	}
	got, err = parseConfigBool("off")
	if err != nil {
		t.Fatalf("parseConfigBool() off error = %v", err)
	}
	if got {
		t.Fatal("parseConfigBool() expected false")
	}
	if _, err := parseConfigBool("maybe"); err == nil {
		t.Fatal("parseConfigBool() expected error")
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

func TestDashboardAssignmentsKeepsUpcomingUnsubmitted(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	rows := []AssignmentRow{
		{ID: "1:past", Assignment: klas.Assignment{Title: "지난 과제", DueAt: &past}},
		{ID: "1:done", Assignment: klas.Assignment{Title: "제출 과제", DueAt: &future, Submitted: true}},
		{ID: "1:todo", Assignment: klas.Assignment{Title: "할 과제", DueAt: &future}},
	}

	got := dashboardAssignments(rows, 5)
	if len(got) != 1 || got[0].ID != "1:todo" {
		t.Fatalf("dashboardAssignments() = %+v", got)
	}
}

func TestDashboardAttendanceCountsMarks(t *testing.T) {
	got := dashboardAttendance([]AttendanceRow{
		{
			Sessions: []klas.AttendanceSession{
				{Slots: []klas.AttendanceSlot{
					{Mark: "O"},
					{Mark: "X"},
					{Mark: "L"},
					{Mark: "R"},
					{Mark: "A"},
					{Mark: "??"},
				}},
			},
		},
		{Err: errDashboardTest},
	})

	if got.TotalCourses != 2 || got.Completed != 1 || got.Absent != 1 || got.Late != 1 || got.LeaveEarly != 1 || got.Excused != 1 || got.Unknown != 1 || got.DetailErrors != 1 {
		t.Fatalf("dashboardAttendance() = %+v", got)
	}
	if len(got.Rows) != 2 || got.Rows[0].Completed != 1 || got.Rows[0].Unknown != 1 || got.Rows[1].Err == nil {
		t.Fatalf("dashboardAttendance() rows = %+v", got.Rows)
	}
}

func TestDashboardNoticesSortsByRecentDate(t *testing.T) {
	oldDate := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	newDate := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	rows := []NoticeRow{
		{ID: "old", Notice: klas.Notice{Registered: &oldDate, Top: true}},
		{ID: "new", Notice: klas.Notice{Registered: &newDate}},
	}

	got := dashboardNotices(rows, 2)
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("dashboardNotices() = %+v", got)
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

func TestBuildLectureReminderNotesIncludesMarker(t *testing.T) {
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	notes := buildLectureReminderNotes(LectureRow{
		ID:         "7:lecture",
		TermValue:  "2026,1",
		CourseName: "오픈소스소프트웨어실습",
		Lecture: klas.Lecture{
			Title:        "HuggingFace",
			ModuleTitle:  "14주차",
			EndAt:        &dueAt,
			AchievedTime: "0",
			RequiredTime: "10",
		},
	})

	for _, want := range []string{
		"ID: lecture:7:lecture",
		"과목: 오픈소스소프트웨어실습",
		"제목: HuggingFace",
		"#2026-1 #오픈소스소프트웨어실습",
		"[This reminder is created by KLAP.]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("buildLectureReminderNotes() missing %q: %q", want, notes)
		}
	}
}

func TestBuildAcademicCalendarNotesIncludesMarker(t *testing.T) {
	event := AcademicEvent{Year: "2026", Month: "3월", Date: "3(화)", Title: "개강", Note: "비고"}
	notes := buildAcademicCalendarNotes(event)
	for _, want := range []string{
		"ID: academic:2026:3월:3(화):개강",
		"학년도: 2026",
		"비고: 비고",
		"[This calendar event is created by KLAP.]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("buildAcademicCalendarNotes() missing %q: %q", want, notes)
		}
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
