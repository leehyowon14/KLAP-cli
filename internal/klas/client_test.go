package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEvaluationCourseJSONOmitsRawPayload(t *testing.T) {
	course := EvaluationCourse{
		Name:         "정규화 과목",
		Professor:    "정규화 교수",
		OpenGwamokNo: "subject-1",
		Evaluated:    true,
		ThisYear:     "2026",
		Hakgi:        "1",
		Raw:          evaluationCourseItem{Name: "raw-secret-course"},
	}
	payload, err := json.Marshal(course)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("EvaluationCourse JSON contains Raw payload: %s", payload)
	}
	var roundTrip EvaluationCourse
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.Name != course.Name || roundTrip.Professor != course.Professor || roundTrip.OpenGwamokNo != course.OpenGwamokNo || !roundTrip.Evaluated || roundTrip.ThisYear != course.ThisYear || roundTrip.Hakgi != course.Hakgi {
		t.Fatalf("EvaluationCourse round trip = %+v", roundTrip)
	}
}

func TestSyllabusJSONOmitsRawContactPayload(t *testing.T) {
	syllabus := Syllabus{
		SubjectID:  "subject-1",
		KoreanName: "정규화 과목",
		Professor:  "정규화 교수",
		Purpose:    "정규화 목표",
		Times:      []SyllabusTime{{Weekday: "월", Periods: []int{1, 2}, Room: "R101"}},
		Raw: syllabusDataItem{
			Email:   "raw-secret@example.com",
			PhoneNo: "raw-secret-phone",
			TelNo:   "raw-secret-tel",
		},
		RawTimeResponse: []syllabusTimeItem{{Room: "raw-secret-room"}},
	}
	payload, err := json.Marshal(syllabus)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) || bytes.Contains(payload, []byte(`"RawTimeResponse"`)) {
		t.Fatalf("Syllabus JSON contains raw contact payload: %s", payload)
	}
	var roundTrip Syllabus
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.SubjectID != syllabus.SubjectID || roundTrip.KoreanName != syllabus.KoreanName || roundTrip.Professor != syllabus.Professor || roundTrip.Purpose != syllabus.Purpose || len(roundTrip.Times) != 1 || roundTrip.Times[0].Room != "R101" {
		t.Fatalf("Syllabus round trip = %+v", roundTrip)
	}
}

func TestLectureJSONOmitsRawPayload(t *testing.T) {
	lecture := Lecture{
		ContentID:    "content-1",
		LearningSeq:  "42",
		Title:        "정규화 강의",
		Progress:     "30",
		RequiredTime: "60",
		Raw: lectureListItem{
			GroupCode: "raw-secret-group",
			SubjectID: "raw-secret-subject",
		},
	}
	payload, err := json.Marshal(lecture)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("Lecture JSON contains Raw payload: %s", payload)
	}
	var roundTrip Lecture
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.ContentID != lecture.ContentID || roundTrip.LearningSeq != lecture.LearningSeq || roundTrip.Title != lecture.Title || roundTrip.Progress != lecture.Progress || roundTrip.RequiredTime != lecture.RequiredTime {
		t.Fatalf("Lecture round trip = %+v", roundTrip)
	}
	if _, err := lectureLearningStatusPayload(roundTrip, "Y"); err == nil {
		t.Fatal("cached Lecture unexpectedly retained action payload")
	}
}

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

func TestBuildEvaluationPayloadUsesRequestedDefaults(t *testing.T) {
	term := EvaluationTerm{Year: "2026", Hakgi: "1", JudgeChasu: "last", JudgeName: "기말평가"}
	course := EvaluationCourse{
		Name:          "오픈소스소프트웨어실습",
		ThisYear:      "2026",
		Hakgi:         "1",
		OpenMajorCode: "I040",
		OpenGrade:     "2",
		OpenGwamokNo:  "1234",
		BunbanNo:      "01",
	}
	form := EvaluationForm{
		Questions: map[string]any{
			"q01":     "강의 문항",
			"q02":     "강의 문항",
			"sq01":    "특수 문항",
			"tq01":    "소규모 문항",
			"fq01":    "원격 문항",
			"chamgo":  "서술형",
			"chamgo2": "기타2",
			"chamgo3": "원격 서술형",
		},
		BunbanOptions: map[string]any{
			"experimentOpt": "Y",
			"eng100Opt":     "Y",
			"recOpt":        "Y",
		},
	}

	payload := BuildEvaluationPayload(term, course, form, EvaluationAnswerOptions{})

	for _, key := range []string{"a01", "a02", "sa01", "fa01", "ta01"} {
		if payload[key] != "5" {
			t.Fatalf("%s = %v, want 5", key, payload[key])
		}
	}
	if payload["chamgo2Opt"] != "N" {
		t.Fatalf("chamgo2Opt = %v, want N", payload["chamgo2Opt"])
	}
	for _, key := range []string{"chamgo", "chamgo3"} {
		if payload[key] != "많은 도움 되었습니다. 한학기동안 감사했습니다." {
			t.Fatalf("%s = %v", key, payload[key])
		}
	}
	if payload["engOpt"] != "N" || payload["ea1"] != "" || payload["chamgoEng"] != "" {
		t.Fatalf("engineering fields should be disabled by default: engOpt=%v ea1=%v chamgoEng=%v", payload["engOpt"], payload["ea1"], payload["chamgoEng"])
	}
}

func TestBuildEvaluationPayloadSkipsEngineeringWithoutExplicitOptIn(t *testing.T) {
	term := EvaluationTerm{Year: "2026", Hakgi: "1", JudgeChasu: "last", JudgeName: "기말평가"}
	course := EvaluationCourse{Name: "공학과목", Engineering: true}
	form := EvaluationForm{
		Questions:       map[string]any{"q01": "강의 문항"},
		EngineeringView: "1",
		Engineering:     map[string]any{"level1": "공학인증 문항", "studyResult1": "학습성과"},
	}

	payload := BuildEvaluationPayload(term, course, form, EvaluationAnswerOptions{})
	if payload["engOpt"] != "Y" {
		t.Fatalf("engOpt = %v, want Y", payload["engOpt"])
	}
	if payload["ea1"] != "" || payload["ea21"] != "" || payload["ea31"] != "" {
		t.Fatalf("engineering answers should be empty without opt-in: ea1=%v ea21=%v ea31=%v", payload["ea1"], payload["ea21"], payload["ea31"])
	}

	payload = BuildEvaluationPayload(term, course, form, EvaluationAnswerOptions{IncludeEngineering: true})
	if payload["ea1"] != "5" || payload["ea21"] != "5" || payload["ea31"] != "5" || payload["chamgoEng"] == "" {
		t.Fatalf("engineering answers were not filled with opt-in: ea1=%v ea21=%v ea31=%v chamgoEng=%v", payload["ea1"], payload["ea21"], payload["ea31"], payload["chamgoEng"])
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
		CurrentNum:     flexibleString("42"),
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
	if syllabus.CurrentNum != "42" {
		t.Fatalf("buildSyllabus() current num = %q", syllabus.CurrentNum)
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

	cases := map[string]string{
		"https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=viewer-789": "viewer-789",
		"https://kwcommons.kw.ac.kr/em/path-123/?contents=ignored":                                       "path-123",
		"https://kwcommons.kw.ac.kr/em/html-123&amp;contents=abc":                                        "html-123",
		"javascript:openPlayer('https://kwcommons.kw.ac.kr/em/script-123?contents=abc')":                 "script-123",
		"https://kwcommons.kw.ac.kr/player?contents=query-123":                                           "query-123",
	}
	for value, want := range cases {
		if got := ExtractKWCommonsContentID(value); got != want {
			t.Fatalf("ExtractKWCommonsContentID(%q) = %q, want %q", value, got, want)
		}
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

func TestExtractMediaURLNestedMainMediaDesktop(t *testing.T) {
	body := []byte(`<?xml version="1.0"?>
<content version="1.0"><content_playing_info version="1.0"><content_id>699bd27c797cb</content_id><main_media><desktop><html5><method>progressive</method><media_uri>https://kwcommons.kw.ac.kr/contents5/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4</media_uri></html5><flash_fallback><method>pseudo</method><media_uri>https://kwcommons.kw.ac.kr/contents5_pseudo/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4</media_uri></flash_fallback></desktop><mobile><html5><method>progressive</method><media_uri>https://kwcommons.kw.ac.kr/contents5/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4</media_uri></html5></mobile></main_media></content_playing_info></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	want := "https://kwcommons.kw.ac.kr/contents5/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4"
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
