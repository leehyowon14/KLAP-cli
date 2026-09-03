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
	"html"
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
	Raw          assignmentListItem `json:"-"`
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
	Raw        noticeItem `json:"-"`
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
	Raw         attendanceCourseItem
}

type AttendanceSession struct {
	Week  string
	Slots []AttendanceSlot
	Raw   attendanceSessionItem
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
	Raw     cdpAttendanceItem
}

type GradeReport struct {
	Summary GradeSummary
	Terms   []GradeTerm
}

type GradeSummary struct {
	AppliedCredits        int
	MajorAppliedCredits   int
	CultureAppliedCredits int
	EtcAppliedCredits     int
	EarnedCredits         int
	MajorEarnedCredits    int
	CultureEarnedCredits  int
	EtcEarnedCredits      int
	DeletedCredits        int
	GPA                   string
	RetakeGPA             string
	Raw                   gradeSummaryItem
}

type GradeTerm struct {
	Year    string
	Hakgi   string
	Label   string
	Courses []GradeCourse
	Raw     gradeTermItem
}

type GradeCourse struct {
	Name          string
	CourseType    string
	Credits       int
	Grade         string
	CourseCode    string
	Department    string
	Finished      bool
	Retake        bool
	RetakeGrade   string
	GradePublic   bool
	TermCheckOpen bool
	TermFinished  bool
	Raw           gradeCourseItem
}

type Rank struct {
	Year           string
	Hakgi          string
	TermValue      string
	TermLabel      string
	AppliedCredits string
	TotalScore     string
	GPA            string
	Percentile     string
	ClassRank      string
	ClassSize      string
	Warning        string
	Raw            rankItem
}

type EvaluationTerm struct {
	Year        string
	Hakgi       string
	Value       string
	Label       string
	JudgeChasu  string
	JudgeName   string
	FromDate    string
	ToDate      string
	TermEnabled bool
}

type EvaluationCourse struct {
	Name            string
	Professor       string
	CourseType      string
	EngineeringType string
	Credits         string
	Evaluated       bool
	Engineering     bool
	ThisYear        string
	Hakgi           string
	OpenMajorCode   string
	OpenGrade       string
	OpenGwamokNo    string
	BunbanNo        string
	Raw             evaluationCourseItem
}

type EvaluationForm struct {
	Questions       map[string]any
	BunbanOptions   map[string]any
	SmallCourse     string
	EngineeringView string
	Engineering     map[string]any
}

type EvaluationAnswerOptions struct {
	Choice             string
	Text               string
	DiscriminationOpt  string
	IncludeEngineering bool
}

type EvaluationSubmitResult struct {
	Raw string
}

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
	Raw             syllabusDataItem
	RawTimeResponse []syllabusTimeItem
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

type Lecture struct {
	ContentID    string
	PlayURL      string
	LearningSeq  string
	FileID       string
	WeekNo       string
	WeeklySeq    string
	ModuleTitle  string
	Title        string
	Progress     string
	AchievedTime string
	RequiredTime string
	StartAt      *time.Time
	EndAt        *time.Time
	Raw          lectureListItem
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
	Raw            assignmentDetailResponse `json:"-"`
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
	Raw         noticeDetailResponse `json:"-"`
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
	List []noticeItem    `json:"list"`
	Page *noticePageInfo `json:"page"`
}

type noticePageInfo struct {
	CurrentPage   flexibleString `json:"currentPage"`
	PageSize      flexibleString `json:"pageSize"`
	TotalElements flexibleString `json:"totalElements"`
	TotalPages    flexibleString `json:"totalPages"`
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

type SyllabusListItem struct {
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

type gradeSummaryItem struct {
	AppliedCredits        int     `json:"applyHakjum"`
	MajorAppliedCredits   int     `json:"majorApplyHakjum"`
	CultureAppliedCredits int     `json:"cultureApplyHakjum"`
	EtcAppliedCredits     int     `json:"etcApplyHakjum"`
	EarnedCredits         int     `json:"chidukHakjum"`
	MajorEarnedCredits    int     `json:"majorChidukHakjum"`
	CultureEarnedCredits  int     `json:"cultureChidukHakjum"`
	EtcEarnedCredits      int     `json:"etcChidukHakjum"`
	DeletedCredits        int     `json:"delHakjum"`
	GPA                   float64 `json:"hwakinScoresum"`
	RetakeGPA             float64 `json:"jaechulScoresum"`
}

type gradeTermItem struct {
	ThisYear    string            `json:"thisYear"`
	Hakgi       string            `json:"hakgi"`
	HakgiOrder  string            `json:"hakgiOrder"`
	CourseItems []gradeCourseItem `json:"sungjukList"`
}

type gradeCourseItem struct {
	Name          string `json:"gwamokKname"`
	CourseType    string `json:"codeName1"`
	Credits       int    `json:"hakjumNum"`
	Grade         string `json:"getGrade"`
	Department    string `json:"hakgwa"`
	CourseCode    string `json:"hakjungNo"`
	Finished      string `json:"finishOpt"`
	GradeOption   string `json:"sungjukOpt"`
	Retake        string `json:"retakeOpt"`
	RetakeGrade   string `json:"retakeGetGrade"`
	TermCheckOpen string `json:"termCheck"`
	TermFinished  string `json:"termFinish"`
}

type rankItem struct {
	ThisYear       string         `json:"thisYear"`
	Hakgi          string         `json:"hakgi"`
	AppliedCredits flexibleString `json:"applyHakjum"`
	TotalScore     flexibleString `json:"applySum"`
	GPA            flexibleString `json:"applyPoint"`
	ClassRank      flexibleString `json:"classOrder"`
	ClassSize      flexibleString `json:"manNum"`
	Warning        string         `json:"warningOpt"`
	Percentile     flexibleString `json:"pcnt"`
}

type evaluationTermItem struct {
	ThisYear   string `json:"thisYear"`
	Hakgi      string `json:"hakgi"`
	JudgeChasu string `json:"judgeChasu"`
	FromDate   string `json:"fromDate"`
	ToDate     string `json:"toDate"`
}

type evaluationStudentItem struct {
	StudentID string `json:"hakbun"`
	Name      string `json:"kname"`
	Division  string `json:"gubun"`
	Grade     any    `json:"grade"`
	Status    string `json:"hakjukStat"`
	Major     string `json:"codeName1"`
}

type evaluationCourseItem struct {
	ThisYear        string         `json:"thisYear"`
	Hakgi           string         `json:"hakgi"`
	OpenMajorCode   string         `json:"openMajorCode"`
	OpenGrade       string         `json:"openGrade"`
	OpenGwamokNo    string         `json:"openGwamokNo"`
	BunbanNo        string         `json:"bunbanNo"`
	Name            string         `json:"gwamokKname"`
	Professor       string         `json:"memberName"`
	Credits         flexibleString `json:"hakjumNum"`
	CourseType      string         `json:"isuGubun"`
	EngineeringType string         `json:"engIsuGubun"`
	EngOpt          string         `json:"engOpt"`
	JudgeOpt        string         `json:"judgeOpt"`
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

type lectureListItem struct {
	GroupCode    string         `json:"grcode"`
	SubjectID    string         `json:"subj"`
	Year         string         `json:"year"`
	Hakgi        string         `json:"hakgi"`
	Bunban       string         `json:"bunban"`
	Module       flexibleString `json:"module"`
	Lesson       flexibleString `json:"lesson"`
	OID          string         `json:"oid"`
	PTime        flexibleString `json:"ptime"`
	TotalTime    flexibleString `json:"totalTime"`
	LearnTime    flexibleString `json:"learnTime"`
	AchivTime    flexibleString `json:"achivTime"`
	RcognTime    flexibleString `json:"rcognTime"`
	TotRcognTime flexibleString `json:"totRcognTime"`
	TotAchivTime flexibleString `json:"totAchivTime"`
	WeekNo       flexibleString `json:"weekNo"`
	WeeklySeq    flexibleString `json:"weeklyseq"`
	LearningSeq  flexibleString `json:"lrnSn"`
	FileID       flexibleString `json:"fileId"`
	IsPreview    string         `json:"ispreview"`
	Evaluation   string         `json:"evltnSe"`
	Title        string         `json:"sbjt"`
	ModuleTitle  string         `json:"moduletitle"`
	Progress     flexibleString `json:"prog"`
	StartDate    string         `json:"startDate"`
	EndDate      string         `json:"endDate"`
	StartY       string         `json:"sdateY"`
	StartH       string         `json:"sdateH"`
	StartM       string         `json:"sdateM"`
	EndY         string         `json:"edateY"`
	EndH         string         `json:"edateH"`
	EndM         string         `json:"edateM"`
	Starting     string         `json:"starting"`
	MVPLink      string         `json:"mvpLink"`
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
		terms[termIndex].Label = normalizeTermLabel(terms[termIndex].Label, terms[termIndex].Value)
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

	allNotices := make([]Notice, 0)
	seen := make(map[string]struct{})
	for currentPage := 0; ; currentPage++ {
		body, err := c.do(ctx, http.MethodPost, "/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdList.do", map[string]any{
			"selectYearhakgi": yearHakgi,
			"selectSubj":      course.Value,
			"currentPage":     currentPage,
			"searchCondition": "ALL",
			"searchKeyword":   nil,
		})
		if err != nil {
			return nil, err
		}

		notices, totalPages, err := parseNoticeListResponse(body)
		if err != nil {
			return nil, err
		}
		for _, notice := range notices {
			key := notice.BoardNo + ":" + notice.MasterNo
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			allNotices = append(allNotices, notice)
		}
		if currentPage >= totalPages-1 {
			break
		}
	}
	return allNotices, nil
}

func parseNoticeListResponse(body []byte) ([]Notice, int, error) {
	var response noticeListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, 0, fmt.Errorf("공지 목록 응답 파싱 실패: %w", err)
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
	return notices, noticeTotalPages(response.Page), nil
}

func noticeTotalPages(page *noticePageInfo) int {
	if page == nil {
		return 1
	}
	totalPages, err := strconv.Atoi(strings.TrimSpace(page.TotalPages.String()))
	if err != nil || totalPages < 1 {
		return 1
	}
	return totalPages
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
	if err := json.Unmarshal(body, &response); err != nil {
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
	if err := json.Unmarshal(body, &response); err != nil {
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
	if err := json.Unmarshal(body, &response); err != nil {
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

func (c *Client) Grades(ctx context.Context) (GradeReport, error) {
	summaryBody, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/AtnlcScreSungjukTot.do", map[string]any{})
	if err != nil {
		return GradeReport{}, err
	}
	var summaryItem gradeSummaryItem
	if err := json.Unmarshal(summaryBody, &summaryItem); err != nil {
		return GradeReport{}, fmt.Errorf("성적 요약 응답 파싱 실패: %w", err)
	}

	termsBody, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/AtnlcScreSungjukInfo.do", map[string]any{})
	if err != nil {
		return GradeReport{}, err
	}
	var termItems []gradeTermItem
	if err := json.Unmarshal(termsBody, &termItems); err != nil {
		return GradeReport{}, fmt.Errorf("성적 목록 응답 파싱 실패: %w", err)
	}

	terms := make([]GradeTerm, 0, len(termItems))
	for _, item := range termItems {
		term := buildGradeTerm(item)
		if len(term.Courses) == 0 {
			continue
		}
		terms = append(terms, term)
	}

	return GradeReport{
		Summary: buildGradeSummary(summaryItem),
		Terms:   terms,
	}, nil
}

func (c *Client) Ranks(ctx context.Context) ([]Rank, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/StandStdList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var response []rankItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("석차 조회 응답 파싱 실패: %w", err)
	}

	ranks := make([]Rank, 0, len(response))
	for _, item := range response {
		rank := buildRank(item)
		if strings.TrimSpace(rank.TermValue) == "" {
			continue
		}
		ranks = append(ranks, rank)
	}
	return ranks, nil
}

func (c *Client) EvaluationTerm(ctx context.Context) (EvaluationTerm, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/LctreEvlTermCheck.do", nil)
	if err != nil {
		return EvaluationTerm{}, err
	}

	item, enabled, err := parseEvaluationTerm(body)
	if err != nil {
		return EvaluationTerm{}, err
	}
	if !enabled {
		return EvaluationTerm{TermEnabled: false}, nil
	}
	term := buildEvaluationTerm(item)
	term.TermEnabled = true
	return term, nil
}

func (c *Client) EvaluationStudent(ctx context.Context) (evaluationStudentItem, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/LctreEvlGetHakjuk.do", nil)
	if err != nil {
		return evaluationStudentItem{}, err
	}
	var item evaluationStudentItem
	if err := json.Unmarshal(body, &item); err != nil {
		return evaluationStudentItem{}, fmt.Errorf("수업평가 학적 응답 파싱 실패: %w", err)
	}
	return item, nil
}

func (c *Client) EvaluationCourses(ctx context.Context, term EvaluationTerm) ([]EvaluationCourse, error) {
	payload := evaluationTermPayload(term)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/LctreEvlsugangList.do", payload)
	if err != nil {
		return nil, err
	}

	var response []evaluationCourseItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("수업평가 과목 응답 파싱 실패: %w", err)
	}
	courses := make([]EvaluationCourse, 0, len(response))
	for _, item := range response {
		course := buildEvaluationCourse(item)
		if strings.TrimSpace(course.Name) == "" {
			continue
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func (c *Client) EvaluationForm(ctx context.Context, term EvaluationTerm, course EvaluationCourse) (EvaluationForm, error) {
	payload := evaluationCoursePayload(term, course)
	if err := c.checkEvaluationTarget(ctx, payload); err != nil {
		return EvaluationForm{}, err
	}

	questions, err := c.evaluationMap(ctx, "/std/cps/inqire/LctreEvlGetque.do", payload, false)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("수업평가 문항 응답 파싱 실패: %w", err)
	}
	bunban, err := c.evaluationMap(ctx, "/std/cps/inqire/LctreEvlBunbanCheck.do", payload, true)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("수업평가 분반 문항 응답 파싱 실패: %w", err)
	}
	smallCourse, err := c.evaluationString(ctx, "/std/cps/inqire/LctreEvlBunbanSmallCheck.do", payload, true)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("수업평가 소규모 강좌 응답 파싱 실패: %w", err)
	}
	engineering, err := c.evaluationMap(ctx, "/std/cps/inqire/LctreEvlEngQuestion.do", payload, true)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("공학인증 문항 응답 파싱 실패: %w", err)
	}

	engineeringView := "0"
	if course.Engineering && len(engineering) > 0 {
		engineeringView = "1"
	}
	return EvaluationForm{
		Questions:       questions,
		BunbanOptions:   bunban,
		SmallCourse:     smallCourse,
		EngineeringView: engineeringView,
		Engineering:     engineering,
	}, nil
}

func (c *Client) SubmitEvaluation(ctx context.Context, term EvaluationTerm, course EvaluationCourse, form EvaluationForm, opts EvaluationAnswerOptions) (EvaluationSubmitResult, error) {
	payload := BuildEvaluationPayload(term, course, form, opts)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/insertEvl.do", payload)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}
	if shouldSubmitEngineering(term, course, form, opts) {
		_, err = c.do(ctx, http.MethodPost, "/std/cps/inqire/insertEng.do", payload)
		if err != nil {
			return EvaluationSubmitResult{}, err
		}
	}
	return EvaluationSubmitResult{Raw: strings.TrimSpace(string(body))}, nil
}

func BuildEvaluationPayload(term EvaluationTerm, course EvaluationCourse, form EvaluationForm, opts EvaluationAnswerOptions) map[string]any {
	choice := firstNonEmpty(opts.Choice, "5")
	text := firstNonEmpty(opts.Text, "많은 도움 되었습니다. 한학기동안 감사했습니다.")
	discriminationOpt := firstNonEmpty(opts.DiscriminationOpt, "N")

	payload := evaluationCoursePayload(term, course)
	payload["list"] = form.Questions
	payload["info"] = map[string]any{}
	payload["size"] = len(form.Questions)
	payload["bunbanCheckInfo"] = form.BunbanOptions
	payload["smallOpt"] = form.SmallCourse
	payload["eng100Opt"] = mapString(form.BunbanOptions, "eng100Opt")
	payload["engOpt"] = boolYN(course.Engineering)
	payload["engQuestion"] = form.Engineering
	payload["engView"] = form.EngineeringView

	for i := 1; i <= 40; i++ {
		field := fmt.Sprintf("a%02d", i)
		payload[field] = ""
		payload[field+"reason"] = ""
		question := fmt.Sprintf("q%02d", i)
		if hasQuestion(form.Questions, question) {
			payload[field] = choice
		}
	}
	for i := 1; i <= 21; i++ {
		field := fmt.Sprintf("sa%02d", i)
		payload[field] = ""
		if i <= 16 {
			payload[field+"reason"] = ""
		}
	}
	payload["fa01"] = ""
	payload["fa01reason"] = ""
	payload["ta01"] = ""
	payload["ta01reason"] = ""
	payload["ta02"] = ""
	payload["ta02reason"] = ""
	payload["chamgo"] = ""
	payload["chamgo2"] = ""
	payload["chamgo2Opt"] = discriminationOpt
	payload["chamgo3"] = ""

	for _, group := range evaluationSpecialQuestionGroups() {
		if !strings.EqualFold(mapString(form.BunbanOptions, group.option), "Y") {
			continue
		}
		for _, index := range group.indices {
			question := fmt.Sprintf("sq%02d", index)
			if hasQuestion(form.Questions, question) {
				payload[fmt.Sprintf("sa%02d", index)] = choice
			}
		}
	}
	if strings.EqualFold(mapString(form.BunbanOptions, "eng100Opt"), "Y") && hasQuestion(form.Questions, "fq01") {
		payload["fa01"] = choice
	}
	if hasQuestion(form.Questions, "tq01") {
		payload["ta01"] = choice
	}
	if hasQuestion(form.Questions, "tq02") {
		payload["ta02"] = choice
	}
	if hasQuestion(form.Questions, "chamgo") || hasQuestion(form.Questions, "chamgoEval") {
		payload["chamgo"] = text
	}
	if hasQuestion(form.Questions, "chamgo3") || strings.EqualFold(mapString(form.BunbanOptions, "recOpt"), "Y") {
		payload["chamgo3"] = text
	}

	for i := 1; i <= 20; i++ {
		payload[fmt.Sprintf("ea%d", i)] = ""
		payload[fmt.Sprintf("ea2%d", i)] = ""
		payload[fmt.Sprintf("ea3%d", i)] = ""
	}
	payload["ea41"] = ""
	payload["chamgoEng"] = ""
	if shouldSubmitEngineering(term, course, form, opts) {
		for i := 1; i <= 20; i++ {
			if hasQuestion(form.Engineering, fmt.Sprintf("level%d", i)) {
				payload[fmt.Sprintf("ea%d", i)] = choice
			}
			if hasQuestion(form.Engineering, fmt.Sprintf("studyResult%d", i)) {
				payload[fmt.Sprintf("ea2%d", i)] = choice
				payload[fmt.Sprintf("ea3%d", i)] = choice
			}
		}
		payload["ea41"] = choice
		payload["chamgoEng"] = text
	}

	return payload
}

func (c *Client) checkEvaluationTarget(ctx context.Context, payload map[string]any) error {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/lctreEvlCheck.do", payload)
	if err != nil {
		return err
	}
	var response []map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("수업평가 대상자 응답 파싱 실패: %w", err)
	}
	if len(response) == 0 {
		return errors.New("수업평가 대상자가 아닙니다")
	}
	return nil
}

func (c *Client) evaluationMap(ctx context.Context, path string, payload map[string]any, allowEmpty bool) (map[string]any, error) {
	body, err := c.do(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(body)) == "" {
		if allowEmpty {
			return map[string]any{}, nil
		}
		return nil, errors.New("빈 응답입니다")
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) evaluationString(ctx context.Context, path string, payload map[string]any, allowEmpty bool) (string, error) {
	body, err := c.do(ctx, http.MethodPost, path, payload)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", errors.New("빈 응답입니다")
	}
	var text string
	if err := json.Unmarshal(body, &text); err == nil {
		return strings.TrimSpace(text), nil
	}
	return strings.Trim(trimmed, `"`), nil
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

	var response []SyllabusListItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("강의계획서 목록 응답 파싱 실패: %w", err)
	}
	return response, nil
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
	if err := json.Unmarshal(body, &response); err != nil {
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
	if err := json.Unmarshal(body, &response); err != nil {
		return Syllabus{}, fmt.Errorf("강의계획서 상세 응답 파싱 실패: %w", err)
	}
	if len(response) == 0 {
		return Syllabus{}, errors.New("강의계획서 상세 응답이 비어 있습니다")
	}

	timeResponse, err := c.syllabusTimeInfoRaw(ctx, subjectID)
	if err != nil {
		return Syllabus{}, err
	}

	return buildSyllabus(subjectID, response[0], timeResponse), nil
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
		requiredTime := firstNonEmpty(item.PTime.String(), item.RcognTime.String(), item.TotRcognTime.String())
		achievedTime := firstNonEmpty(item.TotalTime.String(), item.AchivTime.String(), item.LearnTime.String(), item.TotAchivTime.String())
		if contentID == "" {
			requiredTime = firstNonEmpty(item.RcognTime.String(), item.TotRcognTime.String(), item.PTime.String())
			achievedTime = firstNonEmpty(item.AchivTime.String(), item.LearnTime.String(), item.TotAchivTime.String(), item.TotalTime.String())
		}
		lectures = append(lectures, Lecture{
			ContentID:    contentID,
			PlayURL:      normalizeKWCommonsPlayURL(firstNonEmpty(item.MVPLink, item.Starting), contentID),
			LearningSeq:  item.LearningSeq.String(),
			FileID:       item.FileID.String(),
			WeekNo:       item.WeekNo.String(),
			WeeklySeq:    item.WeeklySeq.String(),
			ModuleTitle:  strings.TrimSpace(item.ModuleTitle),
			Title:        title,
			Progress:     item.Progress.String(),
			AchievedTime: achievedTime,
			RequiredTime: requiredTime,
			StartAt:      parseLectureDateTime(item.StartDate, item.StartY, item.StartH, item.StartM),
			EndAt:        parseLectureDateTime(item.EndDate, item.EndY, item.EndH, item.EndM),
			Raw:          item,
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
	defer func() {
		_ = response.Body.Close()
	}()

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

func (c *Client) SaveLectureLearningStatus(ctx context.Context, lecture Lecture, lrnStatus string) (LectureProgress, error) {
	payload, err := lectureLearningStatusPayload(lecture, lrnStatus)
	if err != nil {
		return LectureProgress{}, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/SaveLrnStatus.do", payload)
	if err != nil {
		return LectureProgress{}, err
	}
	if strings.Trim(strings.TrimSpace(string(body)), `"`) != "Y" {
		return LectureProgress{}, errors.New("학습활동 수강 상태 저장에 실패했습니다")
	}

	requiredTime := strings.TrimSpace(lecture.RequiredTime)
	return LectureProgress{
		TotalTime: firstNonEmpty(requiredTime, strings.TrimSpace(lecture.Progress)),
		PTime:     requiredTime,
		Progress:  100,
		Completed: true,
	}, nil
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

func buildGradeSummary(item gradeSummaryItem) GradeSummary {
	return GradeSummary{
		AppliedCredits:        item.AppliedCredits,
		MajorAppliedCredits:   item.MajorAppliedCredits,
		CultureAppliedCredits: item.CultureAppliedCredits,
		EtcAppliedCredits:     item.EtcAppliedCredits,
		EarnedCredits:         item.EarnedCredits,
		MajorEarnedCredits:    item.MajorEarnedCredits,
		CultureEarnedCredits:  item.CultureEarnedCredits,
		EtcEarnedCredits:      item.EtcEarnedCredits,
		DeletedCredits:        item.DeletedCredits,
		GPA:                   formatGradeNumber(item.GPA),
		RetakeGPA:             formatGradeNumber(item.RetakeGPA),
		Raw:                   item,
	}
}

func buildGradeTerm(item gradeTermItem) GradeTerm {
	courses := make([]GradeCourse, 0, len(item.CourseItems))
	for _, courseItem := range item.CourseItems {
		name := strings.TrimSpace(courseItem.Name)
		if name == "" {
			continue
		}
		courses = append(courses, GradeCourse{
			Name:          name,
			CourseType:    strings.TrimSpace(courseItem.CourseType),
			Credits:       courseItem.Credits,
			Grade:         strings.TrimSpace(courseItem.Grade),
			CourseCode:    strings.TrimSpace(courseItem.CourseCode),
			Department:    strings.TrimSpace(courseItem.Department),
			Finished:      strings.EqualFold(strings.TrimSpace(courseItem.Finished), "Y"),
			Retake:        strings.EqualFold(strings.TrimSpace(courseItem.Retake), "Y"),
			RetakeGrade:   strings.TrimSpace(courseItem.RetakeGrade),
			GradePublic:   strings.TrimSpace(courseItem.Grade) != "",
			TermCheckOpen: strings.EqualFold(strings.TrimSpace(courseItem.TermCheckOpen), "Y"),
			TermFinished:  strings.EqualFold(strings.TrimSpace(courseItem.TermFinished), "Y"),
			Raw:           courseItem,
		})
	}
	return GradeTerm{
		Year:    strings.TrimSpace(item.ThisYear),
		Hakgi:   strings.TrimSpace(item.Hakgi),
		Label:   gradeTermLabel(item),
		Courses: courses,
		Raw:     item,
	}
}

func gradeTermLabel(item gradeTermItem) string {
	year := strings.TrimSpace(item.ThisYear)
	hakgi := strings.TrimSpace(item.Hakgi)
	order := strings.TrimSpace(item.HakgiOrder)
	switch hakgi {
	case "1", "2":
		if order == "" {
			order = hakgi
		}
		if year == "" {
			return order + "학기"
		}
		return year + "년도 " + order + "학기"
	case "3":
		if year == "" {
			return "여름학기"
		}
		return year + "년도 여름학기"
	case "4":
		if year == "" {
			return "겨울학기"
		}
		return year + "년도 겨울학기"
	default:
		if order != "" {
			if year == "" {
				return order
			}
			return year + "년도 " + order
		}
		return strings.TrimSpace(year + " " + hakgi)
	}
}

func formatGradeNumber(value float64) string {
	if value == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func buildRank(item rankItem) Rank {
	year := strings.TrimSpace(item.ThisYear)
	hakgi := strings.TrimSpace(item.Hakgi)
	termValue := ""
	if year != "" && hakgi != "" {
		termValue = year + "," + hakgi
	}
	return Rank{
		Year:           year,
		Hakgi:          hakgi,
		TermValue:      termValue,
		TermLabel:      rankTermLabel(year, hakgi),
		AppliedCredits: item.AppliedCredits.String(),
		TotalScore:     item.TotalScore.String(),
		GPA:            item.GPA.String(),
		Percentile:     item.Percentile.String(),
		ClassRank:      item.ClassRank.String(),
		ClassSize:      item.ClassSize.String(),
		Warning:        strings.TrimSpace(item.Warning),
		Raw:            item,
	}
}

func parseEvaluationTerm(body []byte) (evaluationTermItem, bool, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == `""` || trimmed == "null" || trimmed == "[]" {
		return evaluationTermItem{}, false, nil
	}

	var item evaluationTermItem
	if err := json.Unmarshal(body, &item); err == nil && strings.TrimSpace(item.ThisYear+item.Hakgi+item.JudgeChasu) != "" {
		return item, true, nil
	}

	var items []evaluationTermItem
	if err := json.Unmarshal(body, &items); err != nil {
		return evaluationTermItem{}, false, fmt.Errorf("수업평가 기간 응답 파싱 실패: %w", err)
	}
	switch len(items) {
	case 0:
		return evaluationTermItem{}, false, nil
	case 1:
		return items[0], true, nil
	default:
		return evaluationTermItem{}, false, errors.New("수업평가 기간이 중복으로 내려왔습니다")
	}
}

func buildEvaluationTerm(item evaluationTermItem) EvaluationTerm {
	year := strings.TrimSpace(item.ThisYear)
	hakgi := strings.TrimSpace(item.Hakgi)
	value := ""
	if year != "" && hakgi != "" {
		value = year + "," + hakgi
	}
	judgeChasu := strings.TrimSpace(item.JudgeChasu)
	return EvaluationTerm{
		Year:       year,
		Hakgi:      hakgi,
		Value:      value,
		Label:      rankTermLabel(year, hakgi),
		JudgeChasu: judgeChasu,
		JudgeName:  evaluationJudgeName(judgeChasu),
		FromDate:   strings.TrimSpace(item.FromDate),
		ToDate:     strings.TrimSpace(item.ToDate),
	}
}

func evaluationJudgeName(judgeChasu string) string {
	switch strings.TrimSpace(judgeChasu) {
	case "middle":
		return "중간평가"
	case "last":
		return "기말평가"
	default:
		return strings.TrimSpace(judgeChasu)
	}
}

func buildEvaluationCourse(item evaluationCourseItem) EvaluationCourse {
	return EvaluationCourse{
		Name:            strings.TrimSpace(item.Name),
		Professor:       strings.TrimSpace(item.Professor),
		CourseType:      strings.TrimSpace(item.CourseType),
		EngineeringType: strings.TrimSpace(item.EngineeringType),
		Credits:         item.Credits.String(),
		Evaluated:       strings.EqualFold(strings.TrimSpace(item.JudgeOpt), "Y"),
		Engineering:     strings.EqualFold(strings.TrimSpace(item.EngOpt), "Y"),
		ThisYear:        strings.TrimSpace(item.ThisYear),
		Hakgi:           strings.TrimSpace(item.Hakgi),
		OpenMajorCode:   strings.TrimSpace(item.OpenMajorCode),
		OpenGrade:       strings.TrimSpace(item.OpenGrade),
		OpenGwamokNo:    strings.TrimSpace(item.OpenGwamokNo),
		BunbanNo:        strings.TrimSpace(item.BunbanNo),
		Raw:             item,
	}
}

func evaluationTermPayload(term EvaluationTerm) map[string]any {
	return map[string]any{
		"thisYear":       term.Year,
		"hakgi":          term.Hakgi,
		"judgeChasu":     term.JudgeChasu,
		"judgeChasuName": term.JudgeName,
		"termYn":         "Y",
	}
}

func evaluationCoursePayload(term EvaluationTerm, course EvaluationCourse) map[string]any {
	payload := evaluationTermPayload(term)
	payload["thisYear"] = firstNonEmpty(course.ThisYear, term.Year)
	payload["hakgi"] = firstNonEmpty(course.Hakgi, term.Hakgi)
	payload["openGrade"] = course.OpenGrade
	payload["openGwamokNo"] = course.OpenGwamokNo
	payload["openMajorCode"] = course.OpenMajorCode
	payload["gwamokKname"] = course.Name
	payload["bunbanNo"] = course.BunbanNo
	payload["engOpt"] = boolYN(course.Engineering)
	return payload
}

type evaluationQuestionGroup struct {
	option  string
	indices []int
}

func evaluationSpecialQuestionGroups() []evaluationQuestionGroup {
	return []evaluationQuestionGroup{
		{option: "experimentOpt", indices: []int{1, 2}},
		{option: "englishOpt", indices: []int{3, 4}},
		{option: "tblOpt", indices: []int{7, 8}},
		{option: "pblOpt", indices: []int{9, 10}},
		{option: "discuss0onOpt", indices: []int{11, 12}},
		{option: "practiceOpt", indices: []int{13, 14}},
		{option: "etcOpt", indices: []int{15, 16}},
		{option: "recOpt", indices: []int{17, 18, 19, 20, 21}},
	}
}

func shouldSubmitEngineering(term EvaluationTerm, course EvaluationCourse, form EvaluationForm, opts EvaluationAnswerOptions) bool {
	return opts.IncludeEngineering &&
		course.Engineering &&
		strings.EqualFold(strings.TrimSpace(term.JudgeChasu), "last") &&
		strings.EqualFold(strings.TrimSpace(form.EngineeringView), "1")
}

func hasQuestion(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(value)) != "" && strings.TrimSpace(fmt.Sprint(value)) != "<nil>"
}

func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(values[key]))
}

func boolYN(value bool) string {
	if value {
		return "Y"
	}
	return "N"
}

func rankTermLabel(year string, hakgi string) string {
	if year == "" {
		return strings.TrimSpace(hakgi)
	}
	switch strings.TrimSpace(hakgi) {
	case "1":
		return year + "년도 1학기"
	case "2":
		return year + "년도 2학기"
	case "3":
		return year + "년도 여름학기"
	case "4":
		return year + "년도 겨울학기"
	default:
		return strings.TrimSpace(year + " " + hakgi)
	}
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

func ExtractKWCommonsContentID(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if id := extractKWCommonsContentIDFromURL(value); id != "" {
			return id
		}
	}
	return ""
}

func extractKWCommonsContentIDFromURL(value string) string {
	value = strings.TrimSpace(html.UnescapeString(value))
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		if id := extractKWCommonsContentIDFromPath(parsed.EscapedPath()); id != "" {
			return id
		}
		for _, key := range []string{"content_id", "contentId", "contentID", "contents"} {
			if id := normalizeKWCommonsContentID(parsed.Query().Get(key)); id != "" {
				return id
			}
		}
	}
	if id := extractKWCommonsContentIDAfterMarker(value, "em/"); id != "" {
		return id
	}
	for _, marker := range []string{"content_id=", "contentId=", "contentID=", "contents="} {
		if id := extractKWCommonsContentIDAfterMarker(value, marker); id != "" {
			return id
		}
	}
	return ""
}

func extractKWCommonsContentIDFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	for index, part := range parts {
		if part == "em" && index+1 < len(parts) {
			if id := normalizeKWCommonsContentID(parts[index+1]); id != "" {
				return id
			}
		}
	}
	return ""
}

func extractKWCommonsContentIDAfterMarker(value string, marker string) string {
	index := strings.Index(value, marker)
	if index < 0 {
		return ""
	}
	return normalizeKWCommonsContentID(value[index+len(marker):])
}

func normalizeKWCommonsContentID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, delimiter := range []string{"&", "?", "#", "/", "'", "\"", ")", " "} {
		if delimiterIndex := strings.Index(value, delimiter); delimiterIndex >= 0 {
			value = value[:delimiterIndex]
		}
	}
	if decoded, err := url.QueryUnescape(value); err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
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
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if mainMedia == "" {
		mainMedia = regexMainMedia(string(body))
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

func lectureLearningStatusPayload(lecture Lecture, lrnStatus string) (map[string]any, error) {
	item := lecture.Raw
	required := map[string]string{
		"grcode": item.GroupCode,
		"subj":   item.SubjectID,
		"year":   item.Year,
		"hakgi":  item.Hakgi,
		"bunban": item.Bunban,
		"lrnSn":  item.LearningSeq.String(),
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("학습활동 수강 필드가 없습니다: %s", key)
		}
	}
	if strings.TrimSpace(lrnStatus) == "" {
		return nil, errors.New("학습활동 수강 상태가 없습니다")
	}

	return map[string]any{
		"grcode":    item.GroupCode,
		"subj":      item.SubjectID,
		"year":      item.Year,
		"hakgi":     item.Hakgi,
		"bunban":    item.Bunban,
		"lrnSn":     item.LearningSeq.String(),
		"lrnStatus": lrnStatus,
	}, nil
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
	defer func() {
		_ = response.Body.Close()
	}()

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
	defer func() {
		_ = response.Body.Close()
	}()

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

func normalizeTermLabel(label string, value string) string {
	label = strings.TrimSpace(label)
	year, hakgi := splitYearHakgi(value)
	if year == "" || hakgi == "" {
		return label
	}

	seasonLabel := semesterLabel(hakgi)
	if seasonLabel == "" {
		if label != "" {
			return label
		}
		return fmt.Sprintf("%s년도 %s학기", year, hakgi)
	}
	return fmt.Sprintf("%s년도 %s", year, seasonLabel)
}

func semesterLabel(hakgi string) string {
	switch strings.TrimSpace(hakgi) {
	case "3":
		return "여름학기"
	case "4":
		return "겨울학기"
	default:
		return ""
	}
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
