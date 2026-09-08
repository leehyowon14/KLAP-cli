package klas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Syllabus struct {
	SubjectID       string
	CourseCode      string
	KoreanName      string
	EnglishName     string
	FullName        string
	CourseType      string
	Credits         string
	CurrentNum      string
	Professor       string
	ProfessorTitle  string
	Summary         string
	Purpose         string
	Outcome         string
	Competency      string
	BookName        string
	Operation       string
	Evaluation      SyllabusEvaluation
	Schedule        []SyllabusWeek
	Times           []SyllabusTime
	Raw             syllabusDataItem   `json:"-"`
	RawTimeResponse []syllabusTimeItem `json:"-"`
}

type SyllabusEvaluation struct {
	Attendance int
	Learning   int
	Midterm    int
	Final      int
	Report     int
	Quiz       int
	Other      int
}

type SyllabusWeek struct {
	Week    int
	Topic   string
	SubNote string
}

type SyllabusTime struct {
	Weekday string
	Periods []int
	Room    string
}

type SyllabusListItem struct {
	ThisYear      string
	Hakgi         string
	OpenMajorCode string
	OpenGrade     string
	OpenGwamokNo  string
	BunbanNo      string
	KoreanName    string
	Professor     string
	CourseType    string
	CreditHours   string
	Credits       string
	Summary       string
	CloseOpt      string
	VideoURL      string
}

type syllabusListItem struct {
	ThisYear      string         `json:"thisYear"`
	Hakgi         string         `json:"hakgi"`
	OpenMajorCode string         `json:"openMajorCode"`
	OpenGrade     string         `json:"openGrade"`
	OpenGwamokNo  string         `json:"openGwamokNo"`
	BunbanNo      string         `json:"bunbanNo"`
	KoreanName    string         `json:"gwamokKname"`
	Professor     string         `json:"memberName"`
	CourseType    string         `json:"codeName1"`
	CreditHours   flexibleString `json:"sisuNum"`
	Credits       flexibleString `json:"hakjumNum"`
	Summary       string         `json:"summary"`
	CloseOpt      string         `json:"closeOpt"`
	VideoURL      string         `json:"videoUrl"`
}

func syllabusListModel(item syllabusListItem) SyllabusListItem {
	return SyllabusListItem{
		ThisYear:      item.ThisYear,
		Hakgi:         item.Hakgi,
		OpenMajorCode: item.OpenMajorCode,
		OpenGrade:     item.OpenGrade,
		OpenGwamokNo:  item.OpenGwamokNo,
		BunbanNo:      item.BunbanNo,
		KoreanName:    item.KoreanName,
		Professor:     item.Professor,
		CourseType:    item.CourseType,
		CreditHours:   item.CreditHours.String(),
		Credits:       item.Credits.String(),
		Summary:       item.Summary,
		CloseOpt:      item.CloseOpt,
		VideoURL:      item.VideoURL,
	}
}

type syllabusDataItem struct {
	OpenMajorCode  string         `json:"openMajorCode"`
	OpenGrade      string         `json:"openGrade"`
	OpenGwamokNo   string         `json:"openGwamokNo"`
	BunbanNo       string         `json:"bunbanNo"`
	FullName       string         `json:"gwamokkename"`
	OpenCode       string         `json:"openCode"`
	CourseType     string         `json:"codeName1"`
	Credits        flexibleString `json:"hakjumNum"`
	CreditHours    flexibleString `json:"sisuNum"`
	CurrentNum     flexibleString `json:"currentNum"`
	Professor      string         `json:"memberName"`
	ProfessorTitle string         `json:"jikgeubName"`
	TelNo          string         `json:"telNo"`
	PhoneNo        string         `json:"hpNo"`
	Email          string         `json:"email"`
	KoreanName     string         `json:"gwamokKname"`
	EnglishName    string         `json:"gwamokEname"`
	Summary        string         `json:"summary"`
	Purpose        string         `json:"purpose"`
	Outcome        string         `json:"result1"`
	Competency     string         `json:"gwamokAble"`
	BookName       string         `json:"bookName"`
	AttendanceRate int            `json:"attendBiyul"`
	LearningRate   int            `json:"learnBiyul"`
	MidtermRate    int            `json:"middleBiyul"`
	FinalRate      int            `json:"lastBiyul"`
	ReportRate     int            `json:"reportBiyul"`
	QuizRate       int            `json:"quizBiyul"`
	OtherRate      int            `json:"gitaBiyul"`
	Face100Opt     string         `json:"face100Opt"`
	FaceLiveOpt    string         `json:"faceliveOpt"`
	Live100Opt     string         `json:"live100Opt"`
	FaceRecOpt     string         `json:"facerecOpt"`
	RecLiveOpt     string         `json:"recliveOpt"`
	Rec100Opt      string         `json:"rec100Opt"`
	FaceLiveRecOpt string         `json:"faceliverecOpt"`
	Week1Lecture   string         `json:"week1Lecture"`
	Week2Lecture   string         `json:"week2Lecture"`
	Week3Lecture   string         `json:"week3Lecture"`
	Week4Lecture   string         `json:"week4Lecture"`
	Week5Lecture   string         `json:"week5Lecture"`
	Week6Lecture   string         `json:"week6Lecture"`
	Week7Lecture   string         `json:"week7Lecture"`
	Week8Lecture   string         `json:"week8Lecture"`
	Week9Lecture   string         `json:"week9Lecture"`
	Week10Lecture  string         `json:"week10Lecture"`
	Week11Lecture  string         `json:"week11Lecture"`
	Week12Lecture  string         `json:"week12Lecture"`
	Week13Lecture  string         `json:"week13Lecture"`
	Week14Lecture  string         `json:"week14Lecture"`
	Week15Lecture  string         `json:"week15Lecture"`
	Week16Lecture  string         `json:"week16Lecture"`
	Week1Subs      string         `json:"week1Subs"`
	Week2Subs      string         `json:"week2Subs"`
	Week3Subs      string         `json:"week3Subs"`
	Week4Subs      string         `json:"week4Subs"`
	Week5Subs      string         `json:"week5Subs"`
	Week6Subs      string         `json:"week6Subs"`
	Week7Subs      string         `json:"week7Subs"`
	Week8Subs      string         `json:"week8Subs"`
	Week9Subs      string         `json:"week9Subs"`
	Week10Subs     string         `json:"week10Subs"`
	Week11Subs     string         `json:"week11Subs"`
	Week12Subs     string         `json:"week12Subs"`
	Week13Subs     string         `json:"week13Subs"`
	Week14Subs     string         `json:"week14Subs"`
	Week15Subs     string         `json:"week15Subs"`
	Week16Subs     string         `json:"week16Subs"`
}

type syllabusTimeItem struct {
	Weekday string         `json:"dayname1"`
	Time1   flexibleString `json:"timeNo1"`
	Time2   flexibleString `json:"timeNo2"`
	Time3   flexibleString `json:"timeNo3"`
	Time4   flexibleString `json:"timeNo4"`
	Room    string         `json:"locHname"`
	Room2   string         `json:"locHname2"`
	Code    string         `json:"code"`
}

func (c *Client) SyllabusList(ctx context.Context, yearHakgi string, name string, professor string) ([]SyllabusListItem, error) {
	year, hakgi := splitYearHakgi(yearHakgi)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/LectrePlanStdList.do", map[string]any{
		"list":              []any{},
		"selectSubj":        "",
		"selectYear":        year,
		"selecthakgi":       hakgi,
		"isSearch":          "N",
		"randomNum":         "",
		"numText":           "",
		"selectYearList":    []any{},
		"selectRadio":       "all",
		"selectText":        strings.TrimSpace(name),
		"selectProfsr":      strings.TrimSpace(professor),
		"cmmnGamok":         "",
		"selectCmGamokList": []any{},
		"selecthakgwa":      "",
		"selectHwaGwakList": []any{},
		"selectMajor":       "",
		"selectMajorList":   []any{},
		"stopFlag":          "",
		"selectYearHakgi":   year + "," + hakgi,
	})
	if err != nil {
		return nil, err
	}

	var response []syllabusListItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, fmt.Errorf("강의계획서 목록 응답 파싱 실패: %w", err)
	}
	if response == nil {
		return nil, nil
	}
	items := make([]SyllabusListItem, len(response))
	for i, item := range response {
		items[i] = syllabusListModel(item)
	}
	return items, nil
}

func (c *Client) SyllabusTimeInfo(ctx context.Context, subjectID string) ([]SyllabusTime, error) {
	response, err := c.syllabusTimeInfoRaw(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	return syllabusTimes(response), nil
}

func (c *Client) syllabusTimeInfoRaw(ctx context.Context, subjectID string) ([]syllabusTimeItem, error) {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return nil, errors.New("강의계획서 과목 ID가 없습니다")
	}
	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/LectreTimeInfo.do", map[string]any{"selectSubj": subjectID})
	if err != nil {
		return nil, err
	}
	var response []syllabusTimeItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, fmt.Errorf("강의계획서 시간표 응답 파싱 실패: %w", err)
	}
	return response, nil
}

func (c *Client) SyllabusBySubjectID(ctx context.Context, subjectID string) (Syllabus, error) {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return Syllabus{}, errors.New("강의계획서 과목 ID가 없습니다")
	}

	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/LectrePlanData.do", map[string]any{"selectSubj": subjectID})
	if err != nil {
		return Syllabus{}, err
	}

	var response []syllabusDataItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return Syllabus{}, fmt.Errorf("강의계획서 상세 응답 파싱 실패: %w", err)
	}
	if len(response) == 0 {
		return Syllabus{}, schemaError(errors.New("강의계획서 상세 응답이 비어 있습니다"))
	}

	timeResponse, err := c.syllabusTimeInfoRaw(ctx, subjectID)
	if err != nil {
		return Syllabus{}, err
	}

	return buildSyllabus(subjectID, response[0], timeResponse), nil
}

func buildSyllabus(subjectID string, item syllabusDataItem, timeItems []syllabusTimeItem) Syllabus {
	return Syllabus{
		SubjectID:       subjectID,
		CourseCode:      SyllabusCourseCode(item.OpenMajorCode, item.OpenGrade, item.OpenGwamokNo, item.BunbanNo),
		KoreanName:      strings.TrimSpace(item.KoreanName),
		EnglishName:     strings.TrimSpace(item.EnglishName),
		FullName:        strings.TrimSpace(item.FullName),
		CourseType:      strings.TrimSpace(item.CourseType),
		Credits:         firstNonEmpty(item.Credits.String(), item.CreditHours.String()),
		CurrentNum:      item.CurrentNum.String(),
		Professor:       strings.TrimSpace(item.Professor),
		ProfessorTitle:  strings.TrimSpace(item.ProfessorTitle),
		Summary:         cleanSyllabusText(item.Summary),
		Purpose:         cleanSyllabusText(item.Purpose),
		Outcome:         cleanSyllabusText(item.Outcome),
		Competency:      strings.TrimSpace(item.Competency),
		BookName:        strings.TrimSpace(item.BookName),
		Operation:       syllabusOperation(item),
		Evaluation:      syllabusEvaluation(item),
		Schedule:        syllabusSchedule(item),
		Times:           syllabusTimes(timeItems),
		Raw:             item,
		RawTimeResponse: timeItems,
	}
}

func syllabusEvaluation(item syllabusDataItem) SyllabusEvaluation {
	return SyllabusEvaluation{
		Attendance: item.AttendanceRate,
		Learning:   item.LearningRate,
		Midterm:    item.MidtermRate,
		Final:      item.FinalRate,
		Report:     item.ReportRate,
		Quiz:       item.QuizRate,
		Other:      item.OtherRate,
	}
}

func syllabusSchedule(item syllabusDataItem) []SyllabusWeek {
	lectures := []string{
		item.Week1Lecture, item.Week2Lecture, item.Week3Lecture, item.Week4Lecture,
		item.Week5Lecture, item.Week6Lecture, item.Week7Lecture, item.Week8Lecture,
		item.Week9Lecture, item.Week10Lecture, item.Week11Lecture, item.Week12Lecture,
		item.Week13Lecture, item.Week14Lecture, item.Week15Lecture, item.Week16Lecture,
	}
	subNotes := []string{
		item.Week1Subs, item.Week2Subs, item.Week3Subs, item.Week4Subs,
		item.Week5Subs, item.Week6Subs, item.Week7Subs, item.Week8Subs,
		item.Week9Subs, item.Week10Subs, item.Week11Subs, item.Week12Subs,
		item.Week13Subs, item.Week14Subs, item.Week15Subs, item.Week16Subs,
	}

	weeks := make([]SyllabusWeek, 0, len(lectures))
	for index, lecture := range lectures {
		lecture = cleanSyllabusText(lecture)
		subNote := cleanSyllabusText(subNotes[index])
		if lecture == "" && subNote == "" {
			continue
		}
		weeks = append(weeks, SyllabusWeek{
			Week:    index + 1,
			Topic:   lecture,
			SubNote: subNote,
		})
	}
	return weeks
}

func cleanSyllabusText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

func syllabusTimes(items []syllabusTimeItem) []SyllabusTime {
	times := make([]SyllabusTime, 0, len(items))
	for _, item := range items {
		periods := syllabusPeriods(item)
		if strings.TrimSpace(item.Weekday) == "" && len(periods) == 0 && strings.TrimSpace(item.Room) == "" {
			continue
		}
		times = append(times, SyllabusTime{
			Weekday: strings.TrimSpace(item.Weekday),
			Periods: periods,
			Room:    firstNonEmpty(strings.TrimSpace(item.Room), strings.TrimSpace(item.Room2)),
		})
	}
	return times
}

func syllabusPeriods(item syllabusTimeItem) []int {
	values := []string{item.Time1.String(), item.Time2.String(), item.Time3.String(), item.Time4.String()}
	periods := make([]int, 0, len(values))
	for _, value := range values {
		period, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || period <= 0 {
			continue
		}
		periods = append(periods, period)
	}
	return periods
}

func syllabusOperation(item syllabusDataItem) string {
	options := []struct {
		value string
		label string
	}{
		{item.Face100Opt, "100%대면강의"},
		{item.FaceLiveOpt, "대면+실시간 화상강의"},
		{item.FaceRecOpt, "대면+사전녹화강의"},
		{item.FaceLiveRecOpt, "대면+실시간 화상강의+사전녹화강의"},
		{item.RecLiveOpt, "실시간 화상강의+사전녹화강의"},
		{item.Live100Opt, "100%실시간 화상강의"},
		{item.Rec100Opt, "100%사전녹화강의"},
	}
	for _, option := range options {
		if strings.EqualFold(strings.TrimSpace(option.value), "Y") {
			return option.label
		}
	}
	return ""
}

func SyllabusCourseCode(openMajorCode string, openGrade string, openGwamokNo string, bunbanNo string) string {
	parts := []string{
		strings.TrimSpace(openMajorCode),
		strings.TrimSpace(openGrade),
		strings.TrimSpace(openGwamokNo),
		strings.TrimSpace(bunbanNo),
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return strings.Join(parts, "-")
	}
	return strings.Join(parts, "-")
}

func (item SyllabusListItem) CourseCode() string {
	return SyllabusCourseCode(item.OpenMajorCode, item.OpenGrade, item.OpenGwamokNo, item.BunbanNo)
}

func (item SyllabusListItem) SubjectID() (string, error) {
	return SyllabusSubjectIDFromCourseCode(item.ThisYear+","+item.Hakgi, item.CourseCode())
}

func SyllabusSubjectIDFromCourseCode(yearHakgi string, courseCode string) (string, error) {
	year, hakgi := splitYearHakgi(yearHakgi)
	parts := strings.Split(strings.TrimSpace(courseCode), "-")
	if len(parts) != 4 {
		return "", errors.New("학정번호는 <전공코드>-<학년>-<과목번호>-<분반> 형식이어야 합니다")
	}
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
		if parts[index] == "" {
			return "", errors.New("학정번호에 빈 구간이 있습니다")
		}
	}
	if year == "" || hakgi == "" {
		return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
	}
	return "U" + year + hakgi + parts[2] + parts[0] + parts[3] + parts[1], nil
}
