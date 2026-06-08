package klas

import (
	"encoding/json"
	"testing"
)

func TestLooksLikeLoginHTML(t *testing.T) {
	cases := []struct {
		name string
		body []byte
		want bool
	}{
		{name: "html", body: []byte("<html><body>login</body></html>"), want: true},
		{name: "login form path", body: []byte("<script>location='/usr/cmn/login/LoginForm.do'</script>"), want: true},
		{name: "json", body: []byte(`{"loginRequired":true}`), want: false},
		{name: "empty", body: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeLoginHTML(tc.body); got != tc.want {
				t.Fatalf("looksLikeLoginHTML() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLooksLikeLoginPageHTML(t *testing.T) {
	if looksLikeLoginPageHTML([]byte("<html><body>viewer</body></html>")) {
		t.Fatal("looksLikeLoginPageHTML() should allow ordinary viewer html")
	}
	if !looksLikeLoginPageHTML([]byte("<html><script>location='/usr/cmn/login/LoginForm.do'</script></html>")) {
		t.Fatal("looksLikeLoginPageHTML() should detect login page")
	}
}

func TestFirstFieldError(t *testing.T) {
	got := firstFieldError([]fieldError{{Message: ""}, {Message: "개인번호 또는 비밀번호가 일치하지 않습니다."}}, "fallback")
	if got != "개인번호 또는 비밀번호가 일치하지 않습니다." {
		t.Fatalf("firstFieldError() = %q", got)
	}

	got = firstFieldError(nil, "fallback")
	if got != "fallback" {
		t.Fatalf("firstFieldError() fallback = %q", got)
	}
}

func TestAssignmentListItemAcceptsNumericIDs(t *testing.T) {
	var item assignmentListItem
	err := json.Unmarshal([]byte(`{
		"ordseq": 7,
		"weeklyseq": 15,
		"weeklysubseq": 1,
		"title": "과제명",
		"expiredate": "2026-06-17 23:59:59",
		"submityn": "N"
	}`), &item)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.OrdSeq.String() != "7" || item.WeeklySeq.String() != "15" || item.WeeklySubSeq.String() != "1" {
		t.Fatalf("numeric ids were not preserved: %+v", item)
	}
}

func TestNoticeItemAcceptsNumericIDs(t *testing.T) {
	var item noticeItem
	err := json.Unmarshal([]byte(`{
		"boardNo": 1161280,
		"masterNo": 1000000,
		"title": "공지",
		"readCnt": 12,
		"fileCnt": 1
	}`), &item)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.BoardNo.String() != "1161280" || item.MasterNo.String() != "1000000" {
		t.Fatalf("numeric notice ids were not preserved: %+v", item)
	}
	if item.ReadCount.String() != "12" || item.FileCount.String() != "1" {
		t.Fatalf("numeric notice counts were not preserved: %+v", item)
	}
}

func TestParseTimetableEntries(t *testing.T) {
	entries := parseTimetableEntries([]timetableRow{
		{
			"wtTime":        "1",
			"wtHasSchedule": "Y",
			"wtSpan_2":      "2",
			"wtSubj_2":      "SUBJ001",
			"wtSubjNm_2":    "자료구조",
			"wtLocHname_2":  "새빛관 101",
			"wtProfNm_2":    "홍길동",
		},
		{
			"wtTime":        float64(9),
			"wtHasSchedule": "Y",
			"wtSubj_3":      "SUBJ002",
			"wtSubjNm_3":    "온라인강의",
			"wtSpan_3":      "bad",
		},
		{
			"wtTime":        "2",
			"wtHasSchedule": "N",
			"wtSubjNm_1":    "빈 교시",
		},
	})

	if len(entries) != 2 {
		t.Fatalf("parseTimetableEntries() len = %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].Weekday != 2 || entries[0].Period != 1 || entries[0].Span != 2 {
		t.Fatalf("regular entry was not parsed correctly: %+v", entries[0])
	}
	if entries[0].SubjectName != "자료구조" || entries[0].Room != "새빛관 101" || entries[0].Online {
		t.Fatalf("regular entry fields were not parsed correctly: %+v", entries[0])
	}
	if entries[1].Weekday != 3 || entries[1].Period != 9 || entries[1].Span != 1 || !entries[1].Online {
		t.Fatalf("online entry was not parsed correctly: %+v", entries[1])
	}
}

func TestSyllabusSubjectIDFromCourseCode(t *testing.T) {
	got, err := SyllabusSubjectIDFromCourseCode("2026,1", "I040-3-3951-01")
	if err != nil {
		t.Fatalf("SyllabusSubjectIDFromCourseCode() error = %v", err)
	}
	if got != "U202613951I040013" {
		t.Fatalf("SyllabusSubjectIDFromCourseCode() = %q", got)
	}
}

func TestSyllabusListItemSubjectID(t *testing.T) {
	item := SyllabusListItem{
		ThisYear:      "2026",
		Hakgi:         "1",
		OpenMajorCode: "I040",
		OpenGrade:     "3",
		OpenGwamokNo:  "3951",
		BunbanNo:      "01",
	}
	subjectID, err := item.SubjectID()
	if err != nil {
		t.Fatalf("SubjectID() error = %v", err)
	}
	if item.CourseCode() != "I040-3-3951-01" || subjectID != "U202613951I040013" {
		t.Fatalf("item ids = %q, %q", item.CourseCode(), subjectID)
	}
}

func TestBuildAttendanceCourse(t *testing.T) {
	course := buildAttendanceCourse(attendanceCourseItem{
		OpenMajorCode: "I040",
		OpenGrade:     "3",
		OpenGwamokNo:  "3951",
		BunbanNo:      "01",
		KoreanName:    " 컴퓨터그래픽스 ",
		Professor:     " 김동준 ",
		CourseType:    "전선",
		Credits:       flexibleString("3"),
		CreditHours:   flexibleString("3"),
		CurrentNum:    flexibleString("40"),
		Weekday:       "화3,목4",
	})
	if course.CourseCode != "I040-3-3951-01" || course.Name != "컴퓨터그래픽스" || course.Professor != "김동준" {
		t.Fatalf("buildAttendanceCourse() = %+v", course)
	}
	if course.Credits != "3" || course.CreditHours != "3" || course.CurrentNum != "40" || course.Weekday != "화3,목4" {
		t.Fatalf("buildAttendanceCourse() details = %+v", course)
	}
}

func TestBuildSyllabus(t *testing.T) {
	syllabus := buildSyllabus("U202613951I040013", syllabusDataItem{
		OpenMajorCode:  "I040",
		OpenGrade:      "3",
		OpenGwamokNo:   "3951",
		BunbanNo:       "01",
		KoreanName:     "컴퓨터그래픽스",
		EnglishName:    "Computer Graphics",
		Professor:      "김동준",
		CourseType:     "전선",
		Credits:        flexibleString("3"),
		Face100Opt:     "Y",
		AttendanceRate: 10,
		MidtermRate:    25,
		Week1Lecture:   "오리엔테이션",
		Week10Lecture:  "Global Illumination",
		Week10Subs:     "동영상 보강",
	}, []syllabusTimeItem{
		{Weekday: "화", Time1: flexibleString("3"), Room: "새빛103"},
	})

	if syllabus.CourseCode != "I040-3-3951-01" || syllabus.Operation != "100%대면강의" {
		t.Fatalf("buildSyllabus() = %+v", syllabus)
	}
	if len(syllabus.Schedule) != 2 || syllabus.Schedule[1].Week != 10 || syllabus.Schedule[1].SubNote != "동영상 보강" {
		t.Fatalf("buildSyllabus() schedule = %+v", syllabus.Schedule)
	}
	if len(syllabus.Times) != 1 || syllabus.Times[0].Periods[0] != 3 || syllabus.Times[0].Room != "새빛103" {
		t.Fatalf("buildSyllabus() times = %+v", syllabus.Times)
	}
	if syllabus.Evaluation.Attendance != 10 || syllabus.Evaluation.Midterm != 25 {
		t.Fatalf("buildSyllabus() evaluation = %+v", syllabus.Evaluation)
	}
}

func TestSplitYearHakgi(t *testing.T) {
	year, hakgi := splitYearHakgi("2026,1")
	if year != "2026" || hakgi != "1" {
		t.Fatalf("splitYearHakgi() = %q, %q", year, hakgi)
	}

	year, hakgi = splitYearHakgi("2026-2")
	if year != "2026" || hakgi != "2" {
		t.Fatalf("splitYearHakgi() hyphen = %q, %q", year, hakgi)
	}
}

func TestNormalizeTermLabelSeasonSemesters(t *testing.T) {
	if got := normalizeTermLabel("2026년도 3학기", "2026,3"); got != "2026년도 여름학기" {
		t.Fatalf("normalizeTermLabel() summer = %q", got)
	}
	if got := normalizeTermLabel("", "2026,4"); got != "2026년도 겨울학기" {
		t.Fatalf("normalizeTermLabel() winter = %q", got)
	}
	if got := normalizeTermLabel("2026년도 1학기", "2026,1"); got != "2026년도 1학기" {
		t.Fatalf("normalizeTermLabel() regular = %q", got)
	}
}

func TestExtractKWCommonsContentID(t *testing.T) {
	got := ExtractKWCommonsContentID("https://kwcommons.kw.ac.kr/em/content-123&contents=abc", "")
	if got != "content-123" {
		t.Fatalf("ExtractKWCommonsContentID() = %q", got)
	}

	got = ExtractKWCommonsContentID("", "https://kwcommons.kw.ac.kr/em/fallback-456")
	if got != "fallback-456" {
		t.Fatalf("ExtractKWCommonsContentID() fallback = %q", got)
	}
}

func TestExtractMediaURLDesktop(t *testing.T) {
	body := []byte(`<content><desktop><media_uri>https://media.example.com/video.mp4</media_uri></desktop></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	if got != "https://media.example.com/video.mp4" {
		t.Fatalf("ExtractMediaURL() = %q", got)
	}
}

func TestExtractMediaURLPreservesExternalDesktopHost(t *testing.T) {
	body := []byte(`<content><desktop><media_uri>https://professor-media.example.edu/lecture/path/video.m3u8?token=signed</media_uri></desktop></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	want := "https://professor-media.example.edu/lecture/path/video.m3u8?token=signed"
	if got != want {
		t.Fatalf("ExtractMediaURL() = %q, want %q", got, want)
	}
}

func TestExtractMediaURLFallbackMainMedia(t *testing.T) {
	body := []byte(`<content><media_uri target="all">https://media.example.com/path/[MEDIA_FILE]</media_uri><main_media media_id="m1">video.mp4</main_media></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	if got != "https://media.example.com/path/video.mp4" {
		t.Fatalf("ExtractMediaURL() = %q", got)
	}
}

func TestLectureKeyPatternSupportsQuoteVariants(t *testing.T) {
	cases := []string{
		`"lecKey": 'abc-123'`,
		`'lecKey': "abc-123"`,
	}

	for _, body := range cases {
		match := lectureKeyPattern.FindStringSubmatch(body)
		if len(match) < 2 || match[1] != "abc-123" {
			t.Fatalf("lectureKeyPattern did not parse %q: %+v", body, match)
		}
	}
}

func TestParseLectureProgress(t *testing.T) {
	progress, err := parseLectureProgress([]byte(`{"data":{"totalTime":"10","ptime":"50","prog":20}}`))
	if err != nil {
		t.Fatalf("parseLectureProgress() error = %v", err)
	}
	if progress.TotalTime != "10" || progress.PTime != "50" || progress.Progress != 20 || progress.Completed {
		t.Fatalf("parseLectureProgress() = %+v", progress)
	}

	progress, err = parseLectureProgress([]byte(`{"totalTime":"50","ptime":"50","prog":100}`))
	if err != nil {
		t.Fatalf("parseLectureProgress() root error = %v", err)
	}
	if !progress.Completed {
		t.Fatalf("parseLectureProgress() completed = %+v", progress)
	}
}

func TestLectureViewerForm(t *testing.T) {
	form, err := lectureViewerForm(Lecture{Raw: lectureListItem{
		GroupCode: "G",
		SubjectID: "S",
		Year:      "2026",
		Hakgi:     "1",
		Bunban:    "01",
		Module:    flexibleString("M"),
		OID:       "OID",
		PTime:     flexibleString("50"),
		WeekNo:    flexibleString("3"),
		WeeklySeq: flexibleString("2"),
		TotalTime: flexibleString("0"),
		Progress:  flexibleString("0"),
		Lesson:    flexibleString("001"),
		IsPreview: "N",
	}})
	if err != nil {
		t.Fatalf("lectureViewerForm() error = %v", err)
	}
	if form.Get("weeklyseq") != "3" || form.Get("weeklysubseq") != "2" || form.Get("profYN") != "Y" {
		t.Fatalf("lectureViewerForm() = %v", form)
	}
}

func TestLectureLearningStatusPayload(t *testing.T) {
	payload, err := lectureLearningStatusPayload(Lecture{Raw: lectureListItem{
		GroupCode:   "G",
		SubjectID:   "S",
		Year:        "2026",
		Hakgi:       "1",
		Bunban:      "01",
		LearningSeq: flexibleString("15"),
	}}, "Y")
	if err != nil {
		t.Fatalf("lectureLearningStatusPayload() error = %v", err)
	}
	if payload["lrnSn"] != "15" || payload["lrnStatus"] != "Y" {
		t.Fatalf("lectureLearningStatusPayload() = %v", payload)
	}
}

func TestLectureListItemLearningTimeFields(t *testing.T) {
	var item lectureListItem
	if err := json.Unmarshal([]byte(`{"rcognTime":10,"achivTime":0,"learnTime":"0","totRcognTime":"60","totAchivTime":"50"}`), &item); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if item.RcognTime.String() != "10" || item.AchivTime.String() != "0" || item.TotRcognTime.String() != "60" {
		t.Fatalf("learning time fields were not preserved: %+v", item)
	}
}

func TestHTMLToTextKeepsBlockBreaks(t *testing.T) {
	input := `<p>과제 설명</p><p><a href="https://example.com">https://example.com</a></p><p>제출 내용</p><ol><li>GitHub repository 주소</li><li>youtube 링크</li></ol>`

	got := htmlToText(input)
	want := "과제 설명\n\nhttps://example.com\n\n제출 내용\n\n- GitHub repository 주소\n- youtube 링크"
	if got != want {
		t.Fatalf("htmlToText() = %q, want %q", got, want)
	}
}

func TestHTMLToTextKeepsNumberedListCompact(t *testing.T) {
	input := `<p>제출 내용</p><p>1. GitHub repository 주소</p><p>2. youtube 링크</p>`

	got := htmlToText(input)
	want := "제출 내용\n\n1. GitHub repository 주소\n2. youtube 링크"
	if got != want {
		t.Fatalf("htmlToText() = %q, want %q", got, want)
	}
}
