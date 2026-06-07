package klas

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://klas.kw.ac.kr"

var ErrSessionExpired = errors.New("KLAS 세션이 만료되었습니다")

type Client struct {
	httpClient *http.Client
	jar        http.CookieJar
	baseURL    *url.URL
}

type Session struct {
	UserID    string            `json:"userId"`
	Cookies   map[string]string `json:"cookies"`
	CreatedAt time.Time         `json:"createdAt"`
}

type Term struct {
	Label   string   `json:"label"`
	Value   string   `json:"value"`
	Courses []Course `json:"subjList"`
}

type Course struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Assignment struct {
	OrdSeq       string
	WeeklySeq    string
	WeeklySubSeq string
	Title        string
	StartAt      *time.Time
	DueAt        *time.Time
	Submitted    bool
	Raw          assignmentListItem
}

type Notice struct {
	BoardNo    string
	MasterNo   string
	Title      string
	Author     string
	Registered *time.Time
	Top        bool
	ReadCount  string
	FileCount  string
	Raw        noticeItem
}

type TimetableEntry struct {
	SubjectID   string
	SubjectName string
	Weekday     int
	Period      int
	Span        int
	Room        string
	Professor   string
	Online      bool
	Raw         map[string]any
}

type Lecture struct {
	ContentID   string
	PlayURL     string
	ModuleTitle string
	Title       string
	Progress    string
	StartAt     *time.Time
	EndAt       *time.Time
	Raw         lectureListItem
}

type LectureProgress struct {
	TotalTime string
	PTime     string
	Progress  float64
	Completed bool
}

type AssignmentDetail struct {
	OrdSeq         string
	Title          string
	ContentText    string
	StartAt        *time.Time
	DueAt          *time.Time
	Submitted      bool
	ReportType     string
	SubmitFileType string
	FileLimitMB    string
	SubmittedTitle string
	SubmittedText  string
	FinalScore     string
	TutorText      string
	Raw            assignmentDetailResponse
}

type NoticeDetail struct {
	BoardNo     string
	MasterNo    string
	Title       string
	ContentText string
	Author      string
	Registered  *time.Time
	Top         bool
	ReadCount   string
	Attachment  string
	Raw         noticeDetailResponse
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type commonErrorResponse struct {
	Redirect      bool         `json:"redirect"`
	RedirectURL   string       `json:"redirectUrl"`
	FieldErrors   []fieldError `json:"fieldErrors"`
	ResponseText  string       `json:"responseText"`
	LoginRequired bool         `json:"loginRequired"`
	ErrorCount    int          `json:"errorCount"`
}

type loginSecurityResponse struct {
	PublicKey string `json:"publicKey"`
}

type loginConfirmResponse struct {
	FieldErrors   []fieldError `json:"fieldErrors"`
	Response      *loginUser   `json:"response"`
	ErrorCount    int          `json:"errorCount"`
	LoginRequired bool         `json:"loginRequired"`
}

type loginUser struct {
	UserID string `json:"userId"`
}

type assignmentListItem struct {
	OrdSeq       flexibleString `json:"ordseq"`
	WeeklySeq    flexibleString `json:"weeklyseq"`
	WeeklySubSeq flexibleString `json:"weeklysubseq"`
	Title        string         `json:"title"`
	StartDate    string         `json:"startdate"`
	ExpireDate   string         `json:"expiredate"`
	SubmitYN     string         `json:"submityn"`
}

type assignmentDetailResponse struct {
	Report     assignmentReport     `json:"rpt"`
	Submission assignmentSubmission `json:"smt"`
}

type assignmentReport struct {
	OrdSeq         flexibleString `json:"ordseq"`
	Title          string         `json:"title"`
	Contents       string         `json:"contents"`
	StartDate      string         `json:"startdate"`
	ExpireDate     string         `json:"expiredate"`
	ReportType     string         `json:"reptype"`
	SubmitYN       string         `json:"submityn"`
	SubmitFileType string         `json:"submitfiletype"`
	FileLimit      flexibleString `json:"filelimit"`
}

type assignmentSubmission struct {
	Title         string      `json:"title"`
	Contents      string      `json:"contents"`
	FinalScore    interface{} `json:"finalscore"`
	TutorContents string      `json:"tutorcontents"`
}

type noticeListResponse struct {
	List []noticeItem `json:"list"`
}

type noticeDetailResponse struct {
	Board *noticeItem `json:"board"`
}

type noticeItem struct {
	BoardNo    flexibleString `json:"boardNo"`
	MasterNo   flexibleString `json:"masterNo"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	TopAt      string         `json:"topAt"`
	AtchFileID flexibleString `json:"atchFileId"`
	ReadCount  flexibleString `json:"readCnt"`
	UserName   string         `json:"userNm"`
	RegistDt   string         `json:"registDt"`
	FileCount  flexibleString `json:"fileCnt"`
}

type timetableRow map[string]any

type lectureListItem struct {
	GroupCode   string         `json:"grcode"`
	SubjectID   string         `json:"subj"`
	Year        string         `json:"year"`
	Hakgi       string         `json:"hakgi"`
	Bunban      string         `json:"bunban"`
	Module      flexibleString `json:"module"`
	Lesson      flexibleString `json:"lesson"`
	OID         string         `json:"oid"`
	PTime       flexibleString `json:"ptime"`
	TotalTime   flexibleString `json:"totalTime"`
	WeekNo      flexibleString `json:"weekNo"`
	WeeklySeq   flexibleString `json:"weeklyseq"`
	IsPreview   string         `json:"ispreview"`
	Evaluation  string         `json:"evltnSe"`
	Title       string         `json:"sbjt"`
	ModuleTitle string         `json:"moduletitle"`
	Progress    flexibleString `json:"prog"`
	StartDate   string         `json:"startDate"`
	EndDate     string         `json:"endDate"`
	StartY      string         `json:"sdateY"`
	StartH      string         `json:"sdateH"`
	StartM      string         `json:"sdateM"`
	EndY        string         `json:"edateY"`
	EndH        string         `json:"edateH"`
	EndM        string         `json:"edateM"`
	Starting    string         `json:"starting"`
	MVPLink     string         `json:"mvpLink"`
}

type mediaCandidate struct {
	URL    string
	Target string
	Scope  string
}

type flexibleString string

func (s *flexibleString) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*s = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*s = flexibleString(text)
		return nil
	}

	var number json.Number
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err == nil {
		*s = flexibleString(number.String())
		return nil
	}

	return fmt.Errorf("문자열 변환 실패: %s", trimmed)
}

func (s flexibleString) String() string {
	return string(s)
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar 생성 실패: %w", err)
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
		jar:     jar,
		baseURL: parsedBaseURL,
	}, nil
}

func (c *Client) Login(ctx context.Context, studentID string, password string) (Session, error) {
	publicKey, err := c.loginSecurity(ctx)
	if err != nil {
		return Session{}, err
	}

	if !c.hasCookie("SESSION") && !c.hasCookie("WMONID") {
		return Session{}, errors.New("LoginSecurity 응답에 세션 쿠키가 없습니다")
	}

	loginToken, err := buildLoginToken(publicKey, studentID, password)
	if err != nil {
		return Session{}, err
	}

	userID, err := c.loginConfirm(ctx, loginToken)
	if err != nil {
		return Session{}, err
	}

	cookies := c.cookiesMap()
	if cookies["SESSION"] == "" && cookies["WMONID"] == "" {
		return Session{}, errors.New("LoginConfirm 이후 저장할 세션 쿠키가 없습니다")
	}

	return Session{
		UserID:    userID,
		Cookies:   cookies,
		CreatedAt: time.Now(),
	}, nil
}

func (c *Client) SetSession(session Session) {
	var cookies []*http.Cookie
	for name, value := range session.Cookies {
		if strings.TrimSpace(value) == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{Name: name, Value: value, Path: "/"})
	}
	c.jar.SetCookies(c.baseURL, cookies)
}

func (c *Client) Courses(ctx context.Context) ([]Term, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cmn/frame/YearhakgiAtnlcSbjectList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var terms []Term
	if err := json.Unmarshal(body, &terms); err != nil {
		return nil, fmt.Errorf("수업 목록 응답 파싱 실패: %w", err)
	}

	for termIndex := range terms {
		filtered := terms[termIndex].Courses[:0]
		for _, course := range terms[termIndex].Courses {
			if strings.TrimSpace(course.Name) == "" || strings.TrimSpace(course.Value) == "" {
				continue
			}
			filtered = append(filtered, course)
		}
		terms[termIndex].Courses = filtered
	}

	return terms, nil
}

func (c *Client) Assignments(ctx context.Context, yearHakgi string, course Course) ([]Assignment, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return nil, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/TaskStdList.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"currentPage":     0,
	})
	if err != nil {
		return nil, err
	}

	var response []assignmentListItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("과제 목록 응답 파싱 실패: %w", err)
	}

	assignments := make([]Assignment, 0, len(response))
	for _, item := range response {
		ordSeq := item.OrdSeq.String()
		if strings.TrimSpace(ordSeq) == "" {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "제목 없음"
		}
		assignments = append(assignments, Assignment{
			OrdSeq:       ordSeq,
			WeeklySeq:    item.WeeklySeq.String(),
			WeeklySubSeq: item.WeeklySubSeq.String(),
			Title:        title,
			StartAt:      parseKoreanDateTime(item.StartDate),
			DueAt:        parseKoreanDateTime(item.ExpireDate),
			Submitted:    strings.EqualFold(item.SubmitYN, "Y"),
			Raw:          item,
		})
	}
	return assignments, nil
}

func (c *Client) AssignmentDetail(ctx context.Context, yearHakgi string, course Course, ordSeq string) (AssignmentDetail, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return AssignmentDetail{}, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/TaskStdView.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"ordseq":          ordSeq,
	})
	if err != nil {
		return AssignmentDetail{}, err
	}

	var response assignmentDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return AssignmentDetail{}, fmt.Errorf("과제 상세 응답 파싱 실패: %w", err)
	}
	if strings.TrimSpace(response.Report.Title) == "" && strings.TrimSpace(response.Report.Contents) == "" {
		return AssignmentDetail{}, errors.New("과제 상세 응답에 rpt 본문이 없습니다")
	}

	return AssignmentDetail{
		OrdSeq:         firstNonEmpty(response.Report.OrdSeq.String(), ordSeq),
		Title:          firstNonEmpty(strings.TrimSpace(response.Report.Title), "제목 없음"),
		ContentText:    htmlToText(response.Report.Contents),
		StartAt:        parseKoreanDateTime(response.Report.StartDate),
		DueAt:          parseKoreanDateTime(response.Report.ExpireDate),
		Submitted:      strings.EqualFold(response.Report.SubmitYN, "Y"),
		ReportType:     reportTypeLabel(response.Report.ReportType),
		SubmitFileType: response.Report.SubmitFileType,
		FileLimitMB:    response.Report.FileLimit.String(),
		SubmittedTitle: response.Submission.Title,
		SubmittedText:  htmlToText(response.Submission.Contents),
		FinalScore:     fmt.Sprint(response.Submission.FinalScore),
		TutorText:      htmlToText(response.Submission.TutorContents),
		Raw:            response,
	}, nil
}

func (c *Client) Notices(ctx context.Context, yearHakgi string, course Course) ([]Notice, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return nil, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdList.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"currentPage":     0,
		"searchCondition": "ALL",
		"searchKeyword":   nil,
	})
	if err != nil {
		return nil, err
	}

	var response noticeListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("공지 목록 응답 파싱 실패: %w", err)
	}

	notices := make([]Notice, 0, len(response.List))
	for _, item := range response.List {
		boardNo := item.BoardNo.String()
		masterNo := item.MasterNo.String()
		if strings.TrimSpace(boardNo) == "" || strings.TrimSpace(masterNo) == "" {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "제목 없음"
		}
		notices = append(notices, Notice{
			BoardNo:    boardNo,
			MasterNo:   masterNo,
			Title:      title,
			Author:     strings.TrimSpace(item.UserName),
			Registered: parseKlasDateTime(item.RegistDt),
			Top:        strings.EqualFold(item.TopAt, "Y"),
			ReadCount:  item.ReadCount.String(),
			FileCount:  item.FileCount.String(),
			Raw:        item,
		})
	}
	return notices, nil
}

func (c *Client) NoticeDetail(ctx context.Context, yearHakgi string, course Course, boardNo string, masterNo string) (NoticeDetail, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return NoticeDetail{}, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdView.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"boardNo":         strings.TrimSpace(boardNo),
		"masterNo":        strings.TrimSpace(masterNo),
		"cmd":             "select",
	})
	if err != nil {
		return NoticeDetail{}, err
	}

	var response noticeDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return NoticeDetail{}, fmt.Errorf("공지 상세 응답 파싱 실패: %w", err)
	}
	if response.Board == nil {
		return NoticeDetail{}, errors.New("공지 상세 응답에 board 본문이 없습니다")
	}
	if strings.TrimSpace(response.Board.Title) == "" && strings.TrimSpace(response.Board.Content) == "" {
		return NoticeDetail{}, errors.New("공지 상세 응답에 제목과 본문이 없습니다")
	}

	board := response.Board
	return NoticeDetail{
		BoardNo:     firstNonEmpty(board.BoardNo.String(), boardNo),
		MasterNo:    firstNonEmpty(board.MasterNo.String(), masterNo),
		Title:       firstNonEmpty(strings.TrimSpace(board.Title), "제목 없음"),
		ContentText: htmlToText(board.Content),
		Author:      strings.TrimSpace(board.UserName),
		Registered:  parseKlasDateTime(board.RegistDt),
		Top:         strings.EqualFold(board.TopAt, "Y"),
		ReadCount:   board.ReadCount.String(),
		Attachment:  board.AtchFileID.String(),
		Raw:         response,
	}, nil
}

func (c *Client) Timetable(ctx context.Context, yearHakgi string) ([]TimetableEntry, error) {
	year, hakgi := splitYearHakgi(yearHakgi)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/TimetableStdList.do", map[string]any{
		"searchYear":  year,
		"searchHakgi": hakgi,
		"searchPgmNo": "",
	})
	if err != nil {
		return nil, err
	}

	var response []timetableRow
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("시간표 목록 응답 파싱 실패: %w", err)
	}

	return parseTimetableEntries(response), nil
}

func (c *Client) Lectures(ctx context.Context, yearHakgi string, course Course) ([]Lecture, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return nil, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/SelectOnlineCntntsStdList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var response []lectureListItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("강의 목록 응답 파싱 실패: %w", err)
	}

	lectures := make([]Lecture, 0, len(response))
	for _, item := range response {
		if strings.EqualFold(strings.TrimSpace(item.Evaluation), "proj") {
			continue
		}
		if strings.TrimSpace(item.SubjectID) != "" && strings.TrimSpace(course.Value) != "" && strings.TrimSpace(item.SubjectID) != strings.TrimSpace(course.Value) {
			continue
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "제목 없음"
		}
		contentID := ExtractKWCommonsContentID(item.MVPLink, item.Starting)
		lectures = append(lectures, Lecture{
			ContentID:   contentID,
			PlayURL:     normalizeKWCommonsPlayURL(firstNonEmpty(item.MVPLink, item.Starting), contentID),
			ModuleTitle: strings.TrimSpace(item.ModuleTitle),
			Title:       title,
			Progress:    item.Progress.String(),
			StartAt:     parseLectureDateTime(item.StartDate, item.StartY, item.StartH, item.StartM),
			EndAt:       parseLectureDateTime(item.EndDate, item.EndY, item.EndH, item.EndM),
			Raw:         item,
		})
	}
	return lectures, nil
}

func (c *Client) ResolveLectureMediaURL(ctx context.Context, contentID string) (string, error) {
	contentID = strings.TrimSpace(contentID)
	if contentID == "" {
		return "", errors.New("KWCommons 콘텐츠 ID가 없습니다")
	}

	endpoint := "https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=" + url.QueryEscape(contentID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("KWCommons 요청 실패: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("KWCommons 응답 읽기 실패: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("KWCommons HTTP 오류: %s", response.Status)
	}

	mediaURL, err := ExtractMediaURL(body)
	if err != nil {
		return "", err
	}
	return mediaURL, nil
}

func (c *Client) LectureKey(ctx context.Context, lecture Lecture) (string, error) {
	form, err := lectureViewerForm(lecture)
	if err != nil {
		return "", err
	}

	body, err := c.doForm(ctx, "/spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do", form, true)
	if err != nil {
		return "", err
	}

	match := lectureKeyPattern.FindSubmatch(body)
	if len(match) < 2 {
		return "", errors.New("강의 뷰어 응답에서 lecKey를 찾지 못했습니다")
	}
	return string(match[1]), nil
}

func (c *Client) CheckLectureView(ctx context.Context, lecture Lecture, lecKey string) error {
	form, err := lectureProgressForm(lecture, lecKey)
	if err != nil {
		return err
	}
	_, err = c.doForm(ctx, "/spv/lis/lctre/viewer/ChkLctreCntntsView.do", form, false)
	return err
}

func (c *Client) UpdateLectureProgress(ctx context.Context, lecture Lecture, lecKey string) (LectureProgress, error) {
	form, err := lectureProgressForm(lecture, lecKey)
	if err != nil {
		return LectureProgress{}, err
	}

	body, err := c.doForm(ctx, "/spv/lis/lctre/viewer/UpdateProgress.do", form, false)
	if err != nil {
		return LectureProgress{}, err
	}
	return parseLectureProgress(body)
}

func parseTimetableEntries(response []timetableRow) []TimetableEntry {
	entries := make([]TimetableEntry, 0)
	for _, row := range response {
		if !strings.EqualFold(rowString(row, "wtHasSchedule"), "Y") {
			continue
		}
		period, err := strconv.Atoi(rowString(row, "wtTime"))
		if err != nil {
			continue
		}

		for weekday := 1; weekday <= 6; weekday++ {
			suffix := "_" + strconv.Itoa(weekday)
			subjectID := rowString(row, "wtSubj"+suffix)
			subjectName := rowString(row, "wtSubjNm"+suffix)
			if subjectID == "" && subjectName == "" {
				continue
			}
			if subjectName == "" {
				subjectName = subjectID
			}

			span := 1
			if parsedSpan, err := strconv.Atoi(rowString(row, "wtSpan"+suffix)); err == nil && parsedSpan > 0 {
				span = parsedSpan
			}

			room := rowString(row, "wtLocHname"+suffix)
			entries = append(entries, TimetableEntry{
				SubjectID:   subjectID,
				SubjectName: subjectName,
				Weekday:     weekday,
				Period:      period,
				Span:        span,
				Room:        room,
				Professor:   rowString(row, "wtProfNm"+suffix),
				Online:      period > 8 || room == "",
				Raw:         map[string]any(row),
			})
		}
	}
	return entries
}

func ExtractKWCommonsContentID(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		index := strings.Index(value, "em/")
		if index < 0 {
			continue
		}
		id := value[index+len("em/"):]
		for _, delimiter := range []string{"&", "?", "#"} {
			if delimiterIndex := strings.Index(id, delimiter); delimiterIndex >= 0 {
				id = id[:delimiterIndex]
			}
		}
		if id = strings.TrimSpace(id); id != "" {
			return id
		}
	}
	return ""
}

func ExtractMediaURL(body []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var stack []string
	var candidates []mediaCandidate
	var mainMedia string
	var allPrefix string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			candidates = append(candidates, regexMediaCandidates(string(body))...)
			if mainMedia == "" {
				mainMedia = regexMainMedia(string(body))
			}
			break
		}
		switch typed := token.(type) {
		case xml.StartElement:
			stack = append(stack, typed.Name.Local)
			switch typed.Name.Local {
			case "media_uri":
				text, readErr := readElementText(decoder)
				if readErr != nil {
					continue
				}
				stack = stack[:len(stack)-1]
				candidate := mediaCandidate{URL: strings.TrimSpace(text), Scope: currentXMLScope(stack)}
				for _, attr := range typed.Attr {
					if attr.Name.Local == "target" {
						candidate.Target = strings.TrimSpace(attr.Value)
					}
				}
				if strings.Contains(candidate.URL, "[MEDIA_FILE]") {
					allPrefix = strings.Split(candidate.URL, "[MEDIA_FILE]")[0]
				}
				candidates = append(candidates, candidate)
			case "main_media":
				text, readErr := readElementText(decoder)
				if readErr != nil {
					continue
				}
				stack = stack[:len(stack)-1]
				mainMedia = strings.TrimSpace(text)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	for _, candidate := range candidates {
		if strings.Contains(candidate.URL, "[MEDIA_FILE]") && allPrefix == "" {
			allPrefix = strings.Split(candidate.URL, "[MEDIA_FILE]")[0]
		}
		if candidate.Scope == "desktop" && candidate.URL != "" && !strings.Contains(candidate.URL, "[MEDIA_FILE]") {
			return resolveKWCommonsURL(candidate.URL)
		}
	}
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Target, "all") && allPrefix != "" && mainMedia != "" {
			return resolveKWCommonsURL(allPrefix + mainMedia)
		}
	}
	for _, candidate := range candidates {
		if candidate.URL != "" && !strings.Contains(candidate.URL, "[MEDIA_FILE]") {
			return resolveKWCommonsURL(candidate.URL)
		}
	}

	return "", errors.New("KWCommons 응답에서 동영상 URL을 찾을 수 없습니다")
}

func lectureViewerForm(lecture Lecture) (url.Values, error) {
	item := lecture.Raw
	required := map[string]string{
		"grcode":    item.GroupCode,
		"subj":      item.SubjectID,
		"year":      item.Year,
		"hakgi":     item.Hakgi,
		"bunban":    item.Bunban,
		"module":    item.Module.String(),
		"oid":       item.OID,
		"ptime":     item.PTime.String(),
		"weeklyseq": item.WeekNo.String(),
		"lesson":    item.Lesson.String(),
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("강의 수강 필드가 없습니다: %s", key)
		}
	}

	values := url.Values{}
	values.Set("grcode", item.GroupCode)
	values.Set("subj", item.SubjectID)
	values.Set("year", item.Year)
	values.Set("hakgi", item.Hakgi)
	values.Set("bunban", item.Bunban)
	values.Set("module", item.Module.String())
	values.Set("oid", item.OID)
	values.Set("ptime", item.PTime.String())
	values.Set("weeklyseq", item.WeekNo.String())
	values.Set("weeklysubseq", item.WeeklySeq.String())
	values.Set("totalTime", item.TotalTime.String())
	values.Set("prog", item.Progress.String())
	values.Set("lesson", item.Lesson.String())
	values.Set("profYN", "Y")
	values.Set("previewYN", firstNonEmpty(item.IsPreview, "N"))
	values.Set("late", "N")
	return values, nil
}

func lectureProgressForm(lecture Lecture, lecKey string) (url.Values, error) {
	lecKey = strings.TrimSpace(lecKey)
	if lecKey == "" {
		return nil, errors.New("lecKey가 없습니다")
	}

	viewerForm, err := lectureViewerForm(lecture)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	for _, key := range []string{"grcode", "subj", "year", "hakgi", "bunban", "module", "oid", "ptime", "weeklyseq", "weeklysubseq", "lesson"} {
		values.Set(key, viewerForm.Get(key))
	}
	values.Set("lecKey", lecKey)
	return values, nil
}

func parseLectureProgress(body []byte) (LectureProgress, error) {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return LectureProgress{}, fmt.Errorf("수강 진도 응답 파싱 실패: %w", err)
	}

	source := root
	if nested, ok := root["data"].(map[string]any); ok {
		source = nested
	}
	progressText := rowString(source, "prog")
	if progressText == "" {
		return LectureProgress{}, errors.New("수강 진도 응답에 prog가 없습니다")
	}
	progress, err := strconv.ParseFloat(progressText, 64)
	if err != nil {
		return LectureProgress{}, fmt.Errorf("수강 진도 prog 파싱 실패: %w", err)
	}
	return LectureProgress{
		TotalTime: rowString(source, "totalTime"),
		PTime:     rowString(source, "ptime"),
		Progress:  progress,
		Completed: progress >= 100,
	}, nil
}

func (c *Client) SetCourseContext(ctx context.Context, yearHakgi string, course Course) error {
	_, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/LctrumHomeStdInfo.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"selectChangeYn":  "Y",
	})
	if err != nil {
		return fmt.Errorf("과목 컨텍스트 변경 실패: %w", err)
	}
	return nil
}

func (c *Client) loginSecurity(ctx context.Context) (string, error) {
	body, err := c.do(ctx, http.MethodPost, "/usr/cmn/login/LoginSecurity.do", nil)
	if err != nil {
		return "", err
	}

	var response loginSecurityResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("LoginSecurity 응답 파싱 실패: %w", err)
	}
	if strings.TrimSpace(response.PublicKey) == "" {
		return "", errors.New("LoginSecurity 응답에 publicKey가 없습니다")
	}
	return response.PublicKey, nil
}

func (c *Client) loginConfirm(ctx context.Context, loginToken string) (string, error) {
	requestBody := map[string]string{
		"loginToken":     loginToken,
		"redirectUrl":    "",
		"redirectTabUrl": "",
	}

	body, err := c.do(ctx, http.MethodPost, "/usr/cmn/login/LoginConfirm.do", requestBody)
	if err != nil {
		return "", err
	}

	var response loginConfirmResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("LoginConfirm 응답 파싱 실패: %w", err)
	}

	if response.LoginRequired {
		return "", errors.New("로그인이 필요하다는 응답을 받았습니다")
	}
	if response.ErrorCount > 0 {
		return "", errors.New(firstFieldError(response.FieldErrors, "KLAS 로그인에 실패했습니다"))
	}
	if response.Response == nil || strings.TrimSpace(response.Response.UserID) == "" {
		return "", errors.New("LoginConfirm 성공 응답에 userId가 없습니다")
	}

	return response.Response.UserID, nil
}

func (c *Client) do(ctx context.Context, method string, path string, payload any) ([]byte, error) {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})

	var requestBody io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("요청 직렬화 실패: %w", err)
		}
		requestBody = bytes.NewReader(payloadBytes)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json;charset=utf-8")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("KLAS 요청 실패: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("KLAS HTTP 오류: %s", response.Status)
	}
	if looksLikeLoginHTML(body) {
		return nil, ErrSessionExpired
	}
	if err := checkCommonAPIError(body); err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) doForm(ctx context.Context, path string, values url.Values, allowHTML bool) ([]byte, error) {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("KLAS 요청 실패: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("KLAS HTTP 오류: %s", response.Status)
	}
	if (!allowHTML && looksLikeLoginHTML(body)) || (allowHTML && looksLikeLoginPageHTML(body)) {
		return nil, ErrSessionExpired
	}
	if err := checkCommonAPIError(body); err != nil {
		return nil, err
	}

	return body, nil
}

func checkCommonAPIError(body []byte) error {
	var response commonErrorResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil
	}

	if response.LoginRequired {
		return ErrSessionExpired
	}
	if response.ErrorCount > 0 {
		return errors.New(firstFieldError(response.FieldErrors, "KLAS API 오류가 발생했습니다"))
	}
	return nil
}

func buildLoginToken(publicKeyBody string, studentID string, password string) (string, error) {
	publicKey, err := parsePublicKey(publicKeyBody)
	if err != nil {
		return "", err
	}

	payload := map[string]string{
		"loginId":   studentID,
		"loginPwd":  password,
		"storeIdYn": "N",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, payloadBytes)
	if err != nil {
		return "", fmt.Errorf("로그인 토큰 암호화 실패: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func parsePublicKey(publicKeyBody string) (*rsa.PublicKey, error) {
	pemText := "-----BEGIN PUBLIC KEY-----\n" +
		strings.TrimSpace(publicKeyBody) +
		"\n-----END PUBLIC KEY-----\n"

	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, errors.New("publicKey PEM 디코딩 실패")
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("publicKey 파싱 실패: %w", err)
	}

	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("publicKey가 RSA 키가 아닙니다")
	}
	return publicKey, nil
}

func looksLikeLoginHTML(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	if !bytes.HasPrefix(trimmed, []byte("<")) {
		return false
	}

	lower := bytes.ToLower(trimmed)
	return bytes.Contains(lower, []byte("<html")) ||
		bytes.Contains(lower, []byte("loginform.do")) ||
		bytes.Contains(lower, []byte("/usr/cmn/login"))
}

func looksLikeLoginPageHTML(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || !bytes.HasPrefix(trimmed, []byte("<")) {
		return false
	}

	lower := bytes.ToLower(trimmed)
	return bytes.Contains(lower, []byte("loginform.do")) ||
		bytes.Contains(lower, []byte("/usr/cmn/login"))
}

func firstFieldError(errors []fieldError, fallback string) string {
	for _, fieldError := range errors {
		if strings.TrimSpace(fieldError.Message) != "" {
			return fieldError.Message
		}
	}
	return fallback
}

func parseKoreanDateTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		location = time.FixedZone("KST", 9*60*60)
	}

	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
		parsed, err := time.ParseInLocation(layout, value, location)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func parseKlasDateTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return &parsed
	}
	return parseKoreanDateTime(value)
}

func parseLectureDateTime(dateTime string, compactDate string, hour string, minute string) *time.Time {
	if parsed := parseKoreanDateTime(dateTime); parsed != nil {
		return parsed
	}

	compactDate = strings.TrimSpace(compactDate)
	if compactDate == "" {
		return nil
	}
	hour = firstNonEmpty(hour, "00")
	minute = firstNonEmpty(minute, "00")
	value := compactDate + hour + minute

	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		location = time.FixedZone("KST", 9*60*60)
	}

	parsed, err := time.ParseInLocation("200601021504", value, location)
	if err != nil {
		return nil
	}
	return &parsed
}

var (
	htmlBreakPattern      = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlBlockClosePattern = regexp.MustCompile(`(?i)</(p|div|section|article|header|footer|h[1-6]|li|ul|ol|table|thead|tbody|tr)>`)
	htmlBlockOpenPattern  = regexp.MustCompile(`(?i)<(p|div|section|article|header|footer|h[1-6]|ul|ol|table|thead|tbody|tr)(\s+[^>]*)?>`)
	htmlListItemPattern   = regexp.MustCompile(`(?i)<li(\s+[^>]*)?>`)
	htmlCellPattern       = regexp.MustCompile(`(?i)</t[dh]>\s*<t[dh](\s+[^>]*)?>`)
	htmlTagPattern        = regexp.MustCompile(`<[^>]+>`)
	htmlBlankLinePattern  = regexp.MustCompile(`\n{3,}`)
	htmlListGapPattern    = regexp.MustCompile(`(?m)(- [^\n]+)\n\n- `)
	htmlNumberGapPattern  = regexp.MustCompile(`(?m)([0-9]+\. [^\n]+)\n\n([0-9]+\. )`)
	htmlLineSpacePattern  = regexp.MustCompile(`[ \t]+\n`)
	mediaURIPattern       = regexp.MustCompile(`(?is)<media_uri(?:\s+[^>]*)?>([^<]+)</media_uri>`)
	desktopMediaPattern   = regexp.MustCompile(`(?is)<desktop\b[^>]*>.*?<media_uri(?:\s+[^>]*)?>([^<]+)</media_uri>.*?</desktop>`)
	mainMediaPattern      = regexp.MustCompile(`(?is)<main_media(?:\s+[^>]*)?>([^<]+)</main_media>`)
	lectureKeyPattern     = regexp.MustCompile(`["']lecKey["']\s*:\s*['"]([^'"]+)['"]`)
)

func htmlToText(value string) string {
	value = htmlBreakPattern.ReplaceAllString(value, "\n")
	value = htmlCellPattern.ReplaceAllString(value, " ")
	value = htmlListItemPattern.ReplaceAllString(value, "\n- ")
	value = htmlBlockOpenPattern.ReplaceAllString(value, "\n")
	value = htmlBlockClosePattern.ReplaceAllString(value, "\n")
	value = htmlTagPattern.ReplaceAllString(value, "")
	value = decodeHTMLEntities(value)
	value = htmlLineSpacePattern.ReplaceAllString(value, "\n")
	value = htmlBlankLinePattern.ReplaceAllString(value, "\n\n")
	for htmlListGapPattern.MatchString(value) {
		value = htmlListGapPattern.ReplaceAllString(value, "$1\n- ")
	}
	for htmlNumberGapPattern.MatchString(value) {
		value = htmlNumberGapPattern.ReplaceAllString(value, "$1\n$2")
	}
	return strings.TrimSpace(value)
}

func decodeHTMLEntities(value string) string {
	replacements := map[string]string{
		"&nbsp;": " ",
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&#39;":  "'",
		"&apos;": "'",
	}
	for entity, replacement := range replacements {
		value = strings.ReplaceAll(value, entity, replacement)
	}
	return value
}

func reportTypeLabel(value string) string {
	switch value {
	case "1", "P":
		return "개인"
	case "2":
		return "팀"
	default:
		return value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeKWCommonsPlayURL(value string, contentID string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if contentID != "" {
		return "https://kwcommons.kw.ac.kr/em/" + contentID
	}
	return value
}

func readElementText(decoder *xml.Decoder) (string, error) {
	var builder strings.Builder
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch typed := token.(type) {
		case xml.CharData:
			builder.Write([]byte(typed))
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return builder.String(), nil
}

func currentXMLScope(stack []string) string {
	for index := len(stack) - 1; index >= 0; index-- {
		if stack[index] == "desktop" || stack[index] == "mobile" {
			return stack[index]
		}
	}
	return ""
}

func regexMediaCandidates(body string) []mediaCandidate {
	candidates := make([]mediaCandidate, 0)
	for _, match := range desktopMediaPattern.FindAllStringSubmatch(body, -1) {
		candidates = append(candidates, mediaCandidate{URL: strings.TrimSpace(match[1]), Scope: "desktop"})
	}
	for _, match := range mediaURIPattern.FindAllStringSubmatch(body, -1) {
		target := ""
		full := match[0]
		if strings.Contains(strings.ToLower(full), `target="all"`) || strings.Contains(strings.ToLower(full), `target='all'`) {
			target = "all"
		}
		candidates = append(candidates, mediaCandidate{URL: strings.TrimSpace(match[1]), Target: target})
	}
	return candidates
}

func regexMainMedia(body string) string {
	match := mainMediaPattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func resolveKWCommonsURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value, nil
	}
	base, err := url.Parse("https://kwcommons.kw.ac.kr/")
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}

func splitYearHakgi(yearHakgi string) (string, string) {
	parts := strings.FieldsFunc(strings.TrimSpace(yearHakgi), func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if len(parts) == 1 && len(parts[0]) >= 5 {
		return parts[0][:4], parts[0][4:]
	}
	return yearHakgi, ""
}

func rowString(row map[string]any, key string) string {
	value, ok := row[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func (c *Client) hasCookie(name string) bool {
	for _, cookie := range c.jar.Cookies(c.baseURL) {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func (c *Client) cookiesMap() map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range c.jar.Cookies(c.baseURL) {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}
