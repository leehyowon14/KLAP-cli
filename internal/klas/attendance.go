package klas

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type AttendanceCourse struct {
	CourseCode  string
	SubjectID   string
	Name        string
	Professor   string
	CourseType  string
	Credits     string
	CreditHours string
	CurrentNum  string
	Weekday     string
	Raw         attendanceCourseItem `json:"-"`
}

type AttendanceSession struct {
	Week  string
	Slots []AttendanceSlot
	Raw   attendanceSessionItem `json:"-"`
}

type AttendanceSlot struct {
	Index  int
	Status string
	Mark   string
	Date   string
}

type CdpAttendanceReport struct {
	TotalCount string
	Rows       []CdpAttendance
}

type CdpAttendance struct {
	Date    string
	Seq     string
	Title   string
	Speaker string
	Raw     cdpAttendanceItem `json:"-"`
}

type attendanceCourseItem struct {
	ThisYear      string         `json:"thisYear"`
	Hakgi         string         `json:"hakgi"`
	OpenMajorCode string         `json:"openMajorCode"`
	OpenGrade     string         `json:"openGrade"`
	OpenGwamokNo  string         `json:"openGwamokNo"`
	BunbanNo      string         `json:"bunbanNo"`
	KoreanName    string         `json:"gwamokKname"`
	Professor     string         `json:"memberName"`
	CourseType    string         `json:"codeName1"`
	Credits       flexibleString `json:"hakjumNum"`
	CreditHours   flexibleString `json:"sisuNum"`
	CurrentNum    flexibleString `json:"currentNum"`
	Weekday       string         `json:"yoil"`
}

type attendanceSessionItem struct {
	WeeklySeq       flexibleString `json:"weeklyseq"`
	AttendanceDiv1  string         `json:"attendancediv1"`
	AttendanceDiv2  string         `json:"attendancediv2"`
	AttendanceDiv3  string         `json:"attendancediv3"`
	AttendanceDiv4  string         `json:"attendancediv4"`
	AttendanceDate1 string         `json:"attendancedate1"`
	AttendanceDate2 string         `json:"attendancedate2"`
	AttendanceDate3 string         `json:"attendancedate3"`
	AttendanceDate4 string         `json:"attendancedate4"`
}

type cdpAttendanceItem struct {
	Date    string         `json:"cdpDate"`
	Seq     flexibleString `json:"cdpSeq"`
	Title   string         `json:"title"`
	Speaker string         `json:"memberName"`
	Count   flexibleString `json:"cnt"`
}

func (c *Client) AttendanceCourses(ctx context.Context, yearHakgi string) ([]AttendanceCourse, error) {
	year, hakgi := splitYearHakgi(yearHakgi)
	body, err := c.do(ctx, http.MethodPost, "/std/ads/admst/KwAttendStdGwakmokList.do", map[string]any{
		"selectYear":  year,
		"selectHakgi": hakgi,
	})
	if err != nil {
		return nil, err
	}

	var response []attendanceCourseItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, fmt.Errorf("출석 현황 응답 파싱 실패: %w", err)
	}

	courses := make([]AttendanceCourse, 0, len(response))
	for _, item := range response {
		course := buildAttendanceCourse(item)
		if strings.TrimSpace(course.Name) == "" {
			continue
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func (c *Client) AttendanceSessions(ctx context.Context, yearHakgi string, course AttendanceCourse) ([]AttendanceSession, error) {
	year, hakgi := splitYearHakgi(yearHakgi)
	payload := map[string]any{
		"selectYear":    year,
		"selectHakgi":   hakgi,
		"openMajorCode": course.Raw.OpenMajorCode,
		"openGrade":     course.Raw.OpenGrade,
		"openGwamokNo":  course.Raw.OpenGwamokNo,
		"bunbanNo":      course.Raw.BunbanNo,
		"gwamokKname":   course.Raw.KoreanName,
		"codeName1":     course.Raw.CourseType,
		"hakjumNum":     course.Raw.Credits.String(),
		"sisuNum":       course.Raw.CreditHours.String(),
		"memberName":    course.Raw.Professor,
		"currentNum":    course.Raw.CurrentNum.String(),
		"yoil":          course.Raw.Weekday,
		"subj":          course.SubjectID,
	}
	body, err := c.do(ctx, http.MethodPost, "/std/ads/admst/KwAttendStdAttendList.do", payload)
	if err != nil {
		return nil, err
	}

	var response []attendanceSessionItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, fmt.Errorf("출석 상세 응답 파싱 실패: %w", err)
	}

	sessions := make([]AttendanceSession, 0, len(response))
	for _, item := range response {
		session := buildAttendanceSession(item)
		if len(session.Slots) == 0 {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (c *Client) CdpAttendance(ctx context.Context) (CdpAttendanceReport, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/CdpAtendInfo.do", map[string]any{})
	if err != nil {
		return CdpAttendanceReport{}, err
	}

	var response []cdpAttendanceItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return CdpAttendanceReport{}, fmt.Errorf("CDP 출석내역 응답 파싱 실패: %w", err)
	}

	rows := make([]CdpAttendance, 0, len(response))
	for _, item := range response {
		row := buildCdpAttendance(item)
		if strings.TrimSpace(row.Date+row.Seq+row.Title+row.Speaker) == "" {
			continue
		}
		rows = append(rows, row)
	}

	totalCount := ""
	if len(response) > 0 {
		totalCount = response[0].Count.String()
	}
	if strings.TrimSpace(totalCount) == "" {
		totalCount = strconv.Itoa(len(rows))
	}
	return CdpAttendanceReport{TotalCount: totalCount, Rows: rows}, nil
}

func buildAttendanceCourse(item attendanceCourseItem) AttendanceCourse {
	courseCode := SyllabusCourseCode(item.OpenMajorCode, item.OpenGrade, item.OpenGwamokNo, item.BunbanNo)
	subjectID, _ := SyllabusSubjectIDFromCourseCode(item.ThisYear+","+item.Hakgi, courseCode)
	return AttendanceCourse{
		CourseCode:  courseCode,
		SubjectID:   subjectID,
		Name:        strings.TrimSpace(item.KoreanName),
		Professor:   strings.TrimSpace(item.Professor),
		CourseType:  strings.TrimSpace(item.CourseType),
		Credits:     item.Credits.String(),
		CreditHours: item.CreditHours.String(),
		CurrentNum:  item.CurrentNum.String(),
		Weekday:     strings.TrimSpace(item.Weekday),
		Raw:         item,
	}
}

func buildAttendanceSession(item attendanceSessionItem) AttendanceSession {
	slots := []AttendanceSlot{
		attendanceSlot(1, item.AttendanceDiv1, item.AttendanceDate1),
		attendanceSlot(2, item.AttendanceDiv2, item.AttendanceDate2),
		attendanceSlot(3, item.AttendanceDiv3, item.AttendanceDate3),
		attendanceSlot(4, item.AttendanceDiv4, item.AttendanceDate4),
	}
	filtered := slots[:0]
	for _, slot := range slots {
		if strings.TrimSpace(slot.Status) == "" {
			continue
		}
		filtered = append(filtered, slot)
	}
	return AttendanceSession{
		Week:  item.WeeklySeq.String(),
		Slots: filtered,
		Raw:   item,
	}
}

func attendanceSlot(index int, status string, date string) AttendanceSlot {
	status = strings.TrimSpace(status)
	return AttendanceSlot{
		Index:  index,
		Status: status,
		Mark:   AttendanceStatusMark(status),
		Date:   strings.TrimSpace(date),
	}
}

func AttendanceStatusMark(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "AT":
		return "O"
	case "AB":
		return "X"
	case "LT":
		return "L"
	case "LE":
		return "R"
	case "OA":
		return "A"
	default:
		return strings.TrimSpace(status)
	}
}

func buildCdpAttendance(item cdpAttendanceItem) CdpAttendance {
	return CdpAttendance{
		Date:    strings.TrimSpace(item.Date),
		Seq:     item.Seq.String(),
		Title:   strings.TrimSpace(item.Title),
		Speaker: strings.TrimSpace(item.Speaker),
		Raw:     item,
	}
}
