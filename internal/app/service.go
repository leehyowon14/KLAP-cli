package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kw-klap/klap-cli/internal/account"
	"github.com/kw-klap/klap-cli/internal/cache"
	klapcalendar "github.com/kw-klap/klap-cli/internal/calendar"
	"github.com/kw-klap/klap-cli/internal/category"
	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/reminder"
	"github.com/kw-klap/klap-cli/internal/settings"
	"github.com/kw-klap/klap-cli/internal/transcript"
)

type Service struct {
	store                *account.Store
	sessions             sessionStore
	settingsStore        *settings.Store
	cacheStore           *cache.Store
	newKlasClient        func() (*klas.Client, error)
	login                func(context.Context, *klas.Client, string, string) (klas.Session, error)
	reminderBridgePath   string
	calendarBridgePath   string
	categoryBridgePath   string
	transcriptBridgePath string
}

type sessionStore interface {
	LoadPassword(context.Context, string) (string, error)
	LoadSession(context.Context, string) (klas.Session, error)
	SaveSession(context.Context, string, klas.Session) error
}

type UserOption struct {
	StudentID string
}

type DashboardOptions struct {
	User    UserOption
	Refresh bool
}

type SearchOptions struct {
	User    UserOption
	Query   string
	Type    string
	Refresh bool
}

type DueOptions struct {
	User    UserOption
	Days    int
	Refresh bool
}

type CourseListOptions struct {
	User    UserOption
	Refresh bool
}

type AssignmentListOptions struct {
	User          UserOption
	CourseFilter  string
	Refresh       bool
	SyncDecisions map[string]SyncDecision
}

type NoticeListOptions struct {
	User         UserOption
	CourseFilter string
	Refresh      bool
}

type TimetableOptions struct {
	User          UserOption
	Refresh       bool
	SyncDecisions map[string]SyncDecision
}

type AttendanceListOptions struct {
	User    UserOption
	Refresh bool
}

type AttendanceDetailOptions struct {
	User     UserOption
	Selector string
}

type CdpAttendanceOptions struct {
	User UserOption
}

type GradeOptions struct {
	User      UserOption
	TermValue string
	Refresh   bool
}

type RankOptions struct {
	User      UserOption
	TermValue string
	Refresh   bool
}

type EvaluationListOptions struct {
	User    UserOption
	Refresh bool
}

type EvaluationSubmitOptions struct {
	User               UserOption
	Selector           string
	Confirm            bool
	IncludeEngineering bool
}

type SyllabusOptions struct {
	User      UserOption
	Selector  string
	TermValue string
}

type SubjectSearchOptions struct {
	Name      string
	Professor string
	TermValue string
}

type TermListOptions struct {
	User UserOption
}

type LectureListOptions struct {
	User          UserOption
	CourseFilter  string
	Refresh       bool
	SyncDecisions map[string]SyncDecision
}

type LectureDownloadOptions struct {
	User       UserOption
	Dir        string
	OnProgress func(LectureDownloadProgress)
}

type LectureDownloadAllOptions struct {
	User         UserOption
	CourseFilter string
	Dir          string
	OnProgress   func(LectureDownloadProgress)
	Concurrency  int
	LectureIDs   []string
}

type LectureTranscriptOptions struct {
	Locale     string
	OnProgress func(LectureTranscriptProgress)
}

type LectureAttendOptions struct {
	User       UserOption
	Interval   time.Duration
	OnProgress func(LectureRow, klas.LectureProgress)
}

type LectureAttendAllOptions struct {
	User         UserOption
	CourseFilter string
	Interval     time.Duration
	OnProgress   func(LectureRow, klas.LectureProgress)
}

type UserRow struct {
	User    account.User
	Current bool
}

type AssignmentRow struct {
	ID         string
	LegacyID   string `json:"-"`
	TermValue  string
	CourseName string
	DetailURL  string
	Assignment klas.Assignment
}

type NoticeRow struct {
	ID         string
	TermValue  string
	CourseName string
	DetailURL  string
	Notice     klas.Notice
}

type AssignmentDetailResult struct {
	ID         string
	TermValue  string
	CourseName string
	DetailURL  string
	Detail     klas.AssignmentDetail
}

type NoticeDetailResult struct {
	ID         string
	TermValue  string
	CourseName string
	DetailURL  string
	Detail     klas.NoticeDetail
}

type OpenURLResult struct {
	URL string
}

type SearchResult struct {
	Query       string
	Type        string
	Courses     []SearchCourseResult
	Assignments []AssignmentRow
	Notices     []NoticeRow
	Lectures    []LectureRow
	Academics   []AcademicEvent
	Errors      []DashboardSectionError
}

type DueResult struct {
	From   time.Time
	Until  time.Time
	Items  []DueItem
	Errors []DashboardSectionError
}

type DueItem struct {
	Kind       string
	ID         string
	CourseName string
	Title      string
	DueAt      time.Time
	Status     string
}

type SearchCourseResult struct {
	Index  int
	Term   klas.Term
	Course klas.Course
}

type DashboardResult struct {
	Term           klas.Term
	GeneratedAt    time.Time
	Cached         bool
	CacheCreatedAt time.Time
	Assignments    []AssignmentRow
	Notices        []NoticeRow
	Lectures       []LectureRow
	Courses        []DashboardCourse
	Attendance     DashboardAttendance
	Evaluation     DashboardEvaluation
	SectionErrors  []DashboardSectionError
}

type DashboardCourse struct {
	Index       int
	Name        string
	Assignments []AssignmentRow
	Notices     []NoticeRow
	Lectures    []LectureRow
	Attendance  *DashboardAttendanceRow
	Evaluation  *EvaluationRow
}

type DashboardAttendance struct {
	TotalCourses int
	Rows         []DashboardAttendanceRow
	Completed    int
	Absent       int
	Late         int
	LeaveEarly   int
	Excused      int
	Unknown      int
	DetailErrors int
}

type DashboardAttendanceRow struct {
	Index      int
	Course     klas.AttendanceCourse
	Completed  int
	Absent     int
	Late       int
	LeaveEarly int
	Excused    int
	Unknown    int
	Err        error
}

type DashboardEvaluation struct {
	Term    klas.EvaluationTerm
	Enabled bool
	Done    int
	Pending int
	Rows    []EvaluationRow
}

type DashboardSectionError struct {
	Section string
	Err     error
}

type TimetableResult struct {
	Term    klas.Term
	Entries []klas.TimetableEntry
}

type AttendanceListResult struct {
	Term klas.Term
	Rows []AttendanceRow
}

type AttendanceRow struct {
	Index    int
	Course   klas.AttendanceCourse
	Sessions []klas.AttendanceSession
	Err      error
}

type AttendanceDetailResult struct {
	Term klas.Term
	Row  AttendanceRow
}

type CdpAttendanceResult struct {
	Report klas.CdpAttendanceReport
}

type GradeResult struct {
	Report    klas.GradeReport
	TermValue string
}

type RankResult struct {
	Rows []klas.Rank
}

type EvaluationListResult struct {
	Term klas.EvaluationTerm
	Rows []EvaluationRow
}

type EvaluationRow struct {
	Index  int
	Course klas.EvaluationCourse
}

type EvaluationSubmitResult struct {
	Term      klas.EvaluationTerm
	Items     []EvaluationSubmitItem
	Submitted bool
}

type EvaluationSubmitItem struct {
	Row     EvaluationRow
	Skipped bool
	Reason  string
	Err     error
}

type SyllabusResult struct {
	Term      klas.Term
	SubjectID string
	Course    klas.Course
	Syllabus  klas.Syllabus
}

type SubjectSearchResult struct {
	Term klas.Term
	Rows []SubjectSearchRow
}

type SubjectSearchRow struct {
	CourseCode string
	SubjectID  string
	Name       string
	Professor  string
	Times      []klas.SyllabusTime
	Err        error
}

type TermRow struct {
	Index   int
	Term    klas.Term
	Current bool
}

type LectureRow struct {
	ID         string
	LegacyID   string `json:"-"`
	TermValue  string
	CourseName string
	Lecture    klas.Lecture
}

type LectureDownloadResult struct {
	Path     string
	Bytes    int64
	Lecture  LectureRow
	MediaURL string
}

type DownloadSettings struct {
	Dir         string
	Concurrency int
	Caffeinate  bool
	KeepPartial bool
}

type TranscriptSettings struct {
	Concurrency int
}

type ConfigSettings struct {
	Reminder   ReminderSettings
	Calendar   CalendarSettings
	Download   DownloadSettings
	Transcript TranscriptSettings
	Term       TermSettings
}

type DownloadStatusResult struct {
	Dir          string
	Files        int
	Bytes        int64
	PartialFiles int
	PartialBytes int64
	Items        []DownloadFile
}

type DownloadFile struct {
	Path       string
	Bytes      int64
	ModifiedAt time.Time
}

type LectureDownloadItem struct {
	Path    string
	Bytes   int64
	Lecture LectureRow
	Skipped bool
	Err     error
}

type LectureDownloadAllResult struct {
	Items []LectureDownloadItem
}

type LectureTranscriptItem struct {
	Lecture    LectureRow
	InputPath  string
	OutputPath string
	Text       string
	Err        error
}

type LectureTranscriptResult struct {
	Items []LectureTranscriptItem
}

type LectureDownloadProgress struct {
	Lecture      LectureRow
	Path         string
	Stage        string
	CurrentIndex int
	TotalItems   int
	Bytes        int64
	TotalBytes   int64
	Skipped      bool
	Err          error
}

type LectureTranscriptProgress struct {
	Lecture    LectureRow
	InputPath  string
	OutputPath string
	Stage      string
	Progress   float64
	Err        error
}

type LectureAttendResult struct {
	Lecture  LectureRow
	Progress klas.LectureProgress
}

type LectureAttendItem struct {
	Lecture  LectureRow
	Progress klas.LectureProgress
	Err      error
}

type LectureAttendAllResult struct {
	Items []LectureAttendItem
}

type ReminderSyncResult struct {
	Result        reminder.SyncResult
	EligibleCount int
	Conflicts     []SyncConflict
}

type CalendarSyncResult struct {
	Result        klapcalendar.SyncResult
	EligibleCount int
	Conflicts     []SyncConflict
}

type SyncDecision string

const (
	SyncDecisionApply SyncDecision = "apply"
	SyncDecisionKeep  SyncDecision = "keep"
)

type SyncConflict struct {
	Key          string
	Scope        string
	ID           string
	Kind         string
	Title        string
	PreviousHash string
	CurrentHash  string
	Summary      string
}

type SyncConflictError struct {
	Conflicts []SyncConflict
}

func (e SyncConflictError) Error() string {
	return fmt.Sprintf("KLAS에서 갱신된 항목 %d개에 대한 확인이 필요합니다", len(e.Conflicts))
}

type ReminderSettings struct {
	ListName        string
	UseExistingList bool
	AlarmBeforeMin  int
}

type CalendarSettings struct {
	Name                     string
	UseExistingList          bool
	TimetableName            string
	TimetableUseExistingList bool
}

type CategoryOptions struct {
	Reminders []string
	Calendars []string
}

type CacheStatusResult struct {
	Dir   string
	Files int
	Bytes int64
}

type CacheClearResult struct {
	Removed int
}

type TermSettings struct {
	Value string
	Label string
}

type selectedCourse struct {
	Index  int
	Course klas.Course
}

type CourseRef struct {
	TermValue string
	CourseID  string
}

func NewCourseRef(termValue string, course klas.Course) (CourseRef, error) {
	ref := CourseRef{
		TermValue: strings.TrimSpace(termValue),
		CourseID:  strings.TrimSpace(course.Value),
	}
	if ref.TermValue == "" {
		return CourseRef{}, errors.New("CourseRef에 학기 값이 없습니다")
	}
	if ref.CourseID == "" {
		return CourseRef{}, errors.New("CourseRef에 과목 ID가 없습니다")
	}
	return ref, nil
}

const stableResourceIDVersion = "v1"

func stableCourseResourceID(kind string, ref CourseRef, remoteParts ...string) (string, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" || strings.Contains(kind, ":") {
		return "", errors.New("stable resource kind가 올바르지 않습니다")
	}
	if strings.TrimSpace(ref.TermValue) == "" || strings.TrimSpace(ref.CourseID) == "" {
		return "", errors.New("stable resource ID에 CourseRef가 필요합니다")
	}
	encoded := []string{
		kind,
		stableResourceIDVersion,
		encodeIDPart(ref.TermValue),
		encodeIDPart(ref.CourseID),
	}
	for _, part := range remoteParts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", errors.New("stable resource ID의 remote ID가 비어 있습니다")
		}
		encoded = append(encoded, encodeIDPart(part))
	}
	return strings.Join(encoded, ":"), nil
}

func parseStableCourseResourceID(kind string, id string, remotePartCount int) (CourseRef, []string, bool, error) {
	prefix := strings.TrimSpace(kind) + ":"
	id = strings.TrimSpace(id)
	if !strings.HasPrefix(id, prefix) {
		return CourseRef{}, nil, false, nil
	}
	parts := strings.Split(id, ":")
	if len(parts) != 4+remotePartCount || parts[1] != stableResourceIDVersion {
		return CourseRef{}, nil, true, fmt.Errorf("%s stable ID 형식이 올바르지 않습니다", kind)
	}
	termValue, err := decodeIDPart(parts[2])
	if err != nil {
		return CourseRef{}, nil, true, fmt.Errorf("stable ID 학기 파싱 실패: %w", err)
	}
	courseID, err := decodeIDPart(parts[3])
	if err != nil {
		return CourseRef{}, nil, true, fmt.Errorf("stable ID 과목 파싱 실패: %w", err)
	}
	ref := CourseRef{TermValue: strings.TrimSpace(termValue), CourseID: strings.TrimSpace(courseID)}
	if ref.TermValue == "" || ref.CourseID == "" {
		return CourseRef{}, nil, true, errors.New("stable ID의 CourseRef가 비어 있습니다")
	}
	remoteParts := make([]string, 0, remotePartCount)
	for _, encoded := range parts[4:] {
		value, err := decodeIDPart(encoded)
		if err != nil {
			return CourseRef{}, nil, true, fmt.Errorf("stable ID remote 값 파싱 실패: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return CourseRef{}, nil, true, errors.New("stable ID의 remote ID가 비어 있습니다")
		}
		remoteParts = append(remoteParts, value)
	}
	return ref, remoteParts, true, nil
}

func encodeIDPart(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(value)))
}

func decodeIDPart(value string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

type serviceStoreFactories struct {
	settings func() (*settings.Store, error)
	cache    func() (*cache.Store, error)
}

func NewService(store *account.Store) (*Service, error) {
	return newService(store, serviceStoreFactories{
		settings: settings.NewStore,
		cache:    cache.NewStore,
	})
}

func newService(store *account.Store, factories serviceStoreFactories) (*Service, error) {
	settingsStore, err := factories.settings()
	if err != nil {
		return nil, fmt.Errorf("settings store 초기화 실패: %w", err)
	}
	cacheStore, err := factories.cache()
	if err != nil {
		return nil, fmt.Errorf("cache store 초기화 실패: %w", err)
	}
	return &Service{
		store:         store,
		sessions:      store,
		settingsStore: settingsStore,
		cacheStore:    cacheStore,
		newKlasClient: klas.NewClient,
		login: func(ctx context.Context, client *klas.Client, studentID string, password string) (klas.Session, error) {
			return client.Login(ctx, studentID, password)
		},
		reminderBridgePath:   defaultReminderBridgePath(),
		calendarBridgePath:   defaultCalendarBridgePath(),
		categoryBridgePath:   defaultCategoryBridgePath(),
		transcriptBridgePath: defaultTranscriptBridgePath(),
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, studentID string, password string) error {
	if strings.TrimSpace(studentID) == "" || strings.TrimSpace(password) == "" {
		return errors.New("학번과 비밀번호를 모두 입력해야 합니다")
	}

	client, err := klas.NewClient()
	if err != nil {
		return err
	}

	session, err := client.Login(ctx, studentID, password)
	if err != nil {
		return fmt.Errorf("로그인 검증 실패: %w", err)
	}

	if err := s.store.Save(ctx, studentID, password, session); err != nil {
		return fmt.Errorf("계정 저장 실패: %w", err)
	}
	return nil
}

func (s *Service) Users(ctx context.Context) ([]UserRow, error) {
	users, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	currentStudentID, err := s.store.Current(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]UserRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, UserRow{
			User:    user,
			Current: user.StudentID == currentStudentID,
		})
	}
	return rows, nil
}

func (s *Service) SelectUser(ctx context.Context, studentID string) error {
	return s.store.Select(ctx, studentID)
}

func (s *Service) RemoveUser(ctx context.Context, studentID string) error {
	return s.store.Remove(ctx, studentID)
}

func (s *Service) ReminderSettings() (ReminderSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return ReminderSettings{}, err
	}
	return ReminderSettings{
		ListName:        current.Reminder.ListName,
		UseExistingList: current.Reminder.UseExistingList,
		AlarmBeforeMin:  current.Reminder.AlarmBeforeMin,
	}, nil
}

func (s *Service) SetReminderConfig(name string, useExistingList bool) (ReminderSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ReminderSettings{}, errors.New("리마인더 목록 이름은 비워둘 수 없습니다")
	}

	current, err := s.loadSettings()
	if err != nil {
		return ReminderSettings{}, err
	}
	current.Reminder.ListName = name
	current.Reminder.UseExistingList = useExistingList
	if err := s.saveSettings(current); err != nil {
		return ReminderSettings{}, err
	}
	return ReminderSettings{
		ListName:        current.Reminder.ListName,
		UseExistingList: current.Reminder.UseExistingList,
		AlarmBeforeMin:  current.Reminder.AlarmBeforeMin,
	}, nil
}

func (s *Service) SetCalendarConfig(name string, useExistingList bool) (CalendarSettings, error) {
	return s.SetAcademicCalendarConfig(name, useExistingList)
}

func (s *Service) SetAcademicCalendarConfig(name string, useExistingList bool) (CalendarSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CalendarSettings{}, errors.New("캘린더 이름은 비워둘 수 없습니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return CalendarSettings{}, err
	}
	current.Calendar.AcademicName = name
	current.Calendar.AcademicUseExistingList = useExistingList
	if err := s.saveSettings(current); err != nil {
		return CalendarSettings{}, err
	}
	return calendarSettingsFrom(current), nil
}

func (s *Service) SetTimetableCalendarConfig(name string, useExistingList bool) (CalendarSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CalendarSettings{}, errors.New("시간표 캘린더 이름은 비워둘 수 없습니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return CalendarSettings{}, err
	}
	current.Calendar.TimetableName = name
	current.Calendar.TimetableUseExistingList = useExistingList
	if err := s.saveSettings(current); err != nil {
		return CalendarSettings{}, err
	}
	return calendarSettingsFrom(current), nil
}

func (s *Service) DownloadSettings() (DownloadSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return DownloadSettings{}, err
	}
	return downloadSettingsFrom(current), nil
}

func (s *Service) TranscriptSettings() (TranscriptSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return TranscriptSettings{}, err
	}
	return transcriptSettingsFrom(current), nil
}

func downloadSettingsFrom(current settings.Settings) DownloadSettings {
	return DownloadSettings{
		Dir:         current.Download.Dir,
		Concurrency: current.Download.Concurrency,
		Caffeinate:  settings.DownloadCaffeinateEnabled(current.Download),
		KeepPartial: current.Download.KeepPartial,
	}
}

func transcriptSettingsFrom(current settings.Settings) TranscriptSettings {
	return TranscriptSettings{Concurrency: current.Transcript.Concurrency}
}

func calendarSettingsFrom(current settings.Settings) CalendarSettings {
	current.Normalize()
	return CalendarSettings{
		Name:                     current.Calendar.AcademicName,
		UseExistingList:          current.Calendar.AcademicUseExistingList,
		TimetableName:            current.Calendar.TimetableName,
		TimetableUseExistingList: current.Calendar.TimetableUseExistingList,
	}
}

func (s *Service) SetDownloadDir(dir string) (DownloadSettings, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return DownloadSettings{}, errors.New("다운로드 폴더 경로가 필요합니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return DownloadSettings{}, err
	}
	current.Download.Dir = dir
	if err := s.saveSettings(current); err != nil {
		return DownloadSettings{}, err
	}
	return downloadSettingsFrom(current), nil
}

func (s *Service) SetDownloadConfig(dir string, concurrency int, caffeinate *bool, keepPartial *bool) (DownloadSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return DownloadSettings{}, err
	}
	if strings.TrimSpace(dir) != "" {
		current.Download.Dir = strings.TrimSpace(dir)
	}
	if concurrency > 0 {
		current.Download.Concurrency = concurrency
	}
	if caffeinate != nil {
		current.Download.Caffeinate = caffeinate
	}
	if keepPartial != nil {
		current.Download.KeepPartial = *keepPartial
	}
	if err := s.saveSettings(current); err != nil {
		return DownloadSettings{}, err
	}
	return downloadSettingsFrom(current), nil
}

func (s *Service) SetTranscriptConfig(concurrency int) (TranscriptSettings, error) {
	if concurrency < 1 || concurrency > settings.MaxTranscriptConcurrency {
		return TranscriptSettings{}, fmt.Errorf("transcript.concurrency에는 1~%d 사이의 정수가 필요합니다", settings.MaxTranscriptConcurrency)
	}
	current, err := s.loadSettings()
	if err != nil {
		return TranscriptSettings{}, err
	}
	current.Transcript.Concurrency = concurrency
	if err := s.saveSettings(current); err != nil {
		return TranscriptSettings{}, err
	}
	return transcriptSettingsFrom(current), nil
}

func (s *Service) ConfigSettings() (ConfigSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return ConfigSettings{}, err
	}
	return ConfigSettings{
		Reminder: ReminderSettings{
			ListName:        current.Reminder.ListName,
			UseExistingList: current.Reminder.UseExistingList,
			AlarmBeforeMin:  current.Reminder.AlarmBeforeMin,
		},
		Calendar:   calendarSettingsFrom(current),
		Download:   downloadSettingsFrom(current),
		Transcript: transcriptSettingsFrom(current),
		Term:       TermSettings{Value: current.Term.Value, Label: termLabel(current.Term.Value)},
	}, nil
}

func (s *Service) CategoryOptions() (CategoryOptions, error) {
	current, err := s.loadSettings()
	if err != nil {
		return CategoryOptions{}, err
	}
	options, err := category.NewMacOSBridge(s.categoryBridgePath).List()
	if err != nil {
		return CategoryOptions{
			Reminders: uniqueNonEmpty(current.Reminder.ListName, settings.DefaultReminderListName),
			Calendars: uniqueNonEmpty(current.Calendar.AcademicName, current.Calendar.TimetableName, settings.DefaultAcademicCalendarName, settings.DefaultTimetableCalendarName),
		}, nil
	}
	return CategoryOptions{
		Reminders: uniqueNonEmpty(append([]string{current.Reminder.ListName}, options.Reminders...)...),
		Calendars: uniqueNonEmpty(append([]string{current.Calendar.AcademicName, current.Calendar.TimetableName}, options.Calendars...)...),
	}, nil
}

func (s *Service) ResetConfigSettings() (ConfigSettings, error) {
	if err := s.saveSettings(settings.Default()); err != nil {
		return ConfigSettings{}, err
	}
	return s.ConfigSettings()
}

func (s *Service) SetConfigValue(key string, value string) (ConfigSettings, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" {
		return ConfigSettings{}, errors.New("설정 키가 필요합니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return ConfigSettings{}, err
	}
	switch key {
	case "reminder.name", "reminder.list", "reminder.list-name":
		if value == "" {
			return ConfigSettings{}, errors.New("리마인더 목록 이름은 비워둘 수 없습니다")
		}
		current.Reminder.ListName = value
	case "reminder.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Reminder.UseExistingList = parsed
	case "reminder.alarm-before-min", "reminder.alarm-before":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return ConfigSettings{}, errors.New("reminder.alarm-before-min에는 1 이상의 정수가 필요합니다")
		}
		current.Reminder.AlarmBeforeMin = parsed
	case "calendar.name", "calendar.list", "calendar.list-name":
		if value == "" {
			return ConfigSettings{}, errors.New("캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.AcademicName = value
	case "calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Calendar.AcademicUseExistingList = parsed
	case "calendar.academic.name", "academic-calendar.name", "academic-calendar.list":
		if value == "" {
			return ConfigSettings{}, errors.New("학사일정 캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.AcademicName = value
	case "calendar.academic.use-existing-list", "academic-calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Calendar.AcademicUseExistingList = parsed
	case "calendar.timetable.name", "timetable-calendar.name", "timetable-calendar.list":
		if value == "" {
			return ConfigSettings{}, errors.New("시간표 캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.TimetableName = value
	case "calendar.timetable.use-existing-list", "timetable-calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Calendar.TimetableUseExistingList = parsed
	case "download.dir", "download.path":
		if value == "" {
			return ConfigSettings{}, errors.New("다운로드 폴더 경로가 필요합니다")
		}
		current.Download.Dir = value
	case "download.concurrency", "download.workers":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return ConfigSettings{}, errors.New("download.concurrency에는 1 이상의 정수가 필요합니다")
		}
		current.Download.Concurrency = parsed
	case "download.caffeinate", "download.prevent-sleep", "download.keep-awake":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Download.Caffeinate = &parsed
	case "download.keep-partial", "download.resume", "download.partial":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Download.KeepPartial = parsed
	case "transcript.concurrency", "transcript.workers":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > settings.MaxTranscriptConcurrency {
			return ConfigSettings{}, fmt.Errorf("transcript.concurrency에는 1~%d 사이의 정수가 필요합니다", settings.MaxTranscriptConcurrency)
		}
		current.Transcript.Concurrency = parsed
	case "term", "term.value":
		normalized, err := normalizeTermValue(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Term.Value = normalized
	default:
		return ConfigSettings{}, fmt.Errorf("지원하지 않는 설정 키입니다: %s", key)
	}
	if err := s.saveSettings(current); err != nil {
		return ConfigSettings{}, err
	}
	return s.ConfigSettings()
}

func (s *Service) DownloadStatus(dir string) (DownloadStatusResult, error) {
	dir, err := s.effectiveDownloadDir(dir)
	if err != nil {
		return DownloadStatusResult{}, err
	}
	result := DownloadStatusResult{Dir: dir}
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), ".part") {
			result.PartialFiles++
			result.PartialBytes += info.Size()
			return nil
		}
		result.Files++
		result.Bytes += info.Size()
		result.Items = append(result.Items, DownloadFile{
			Path:       path,
			Bytes:      info.Size(),
			ModifiedAt: info.ModTime(),
		})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return DownloadStatusResult{}, fmt.Errorf("다운로드 폴더 조회 실패: %w", err)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		return result.Items[j].ModifiedAt.Before(result.Items[i].ModifiedAt)
	})
	if len(result.Items) > 10 {
		result.Items = result.Items[:10]
	}
	return result, nil
}

func (s *Service) CacheStatus() (CacheStatusResult, error) {
	stats, err := s.cacheStore.Stats()
	if err != nil {
		return CacheStatusResult{}, err
	}
	return CacheStatusResult{
		Dir:   stats.Dir,
		Files: stats.Files,
		Bytes: stats.Bytes,
	}, nil
}

func (s *Service) ClearCache() (CacheClearResult, error) {
	removed, err := s.cacheStore.ClearExceptPrefixes("sync-source:")
	if err != nil {
		return CacheClearResult{}, err
	}
	return CacheClearResult{Removed: removed}, nil
}

func (s *Service) ClearCacheScope(scope string) (CacheClearResult, error) {
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	if strings.HasPrefix(normalizedScope, "sync-source") {
		return CacheClearResult{}, errors.New("동기화 기준 cache는 삭제할 수 없습니다")
	}
	prefix := cacheScopePrefix(normalizedScope)
	if prefix == "" {
		return s.ClearCache()
	}
	removed, err := s.cacheStore.ClearPrefix(prefix)
	if err != nil {
		return CacheClearResult{}, err
	}
	return CacheClearResult{Removed: removed}, nil
}

func (s *Service) TermList(ctx context.Context, opts TermListOptions) ([]TermRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, err
	}

	terms, _, err := s.courses(ctx, studentID, client)
	if err != nil {
		return nil, err
	}
	current, err := s.loadSettings()
	if err != nil {
		return nil, err
	}

	rows := make([]TermRow, 0, len(terms))
	for index, term := range terms {
		rows = append(rows, TermRow{
			Index:   index + 1,
			Term:    term,
			Current: current.Term.Value != "" && term.Value == current.Term.Value,
		})
	}
	if current.Term.Value == "" && len(rows) > 0 {
		rows[0].Current = true
	}
	return rows, nil
}

func (s *Service) SelectTerm(ctx context.Context, selector string, user UserOption) (TermSettings, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return TermSettings{}, errors.New("학기 번호 또는 값을 입력해야 합니다")
	}

	rows, err := s.TermList(ctx, TermListOptions{User: user})
	if err != nil {
		return TermSettings{}, err
	}
	if len(rows) == 0 {
		return TermSettings{}, errors.New("선택할 학기가 없습니다")
	}

	matched, err := selectTermRow(rows, selector)
	if err != nil {
		return TermSettings{}, err
	}

	current, err := s.loadSettings()
	if err != nil {
		return TermSettings{}, err
	}
	current.Term.Value = matched.Term.Value
	if err := s.saveSettings(current); err != nil {
		return TermSettings{}, err
	}
	return TermSettings{Value: matched.Term.Value, Label: matched.Term.Label}, nil
}

func selectTermRow(rows []TermRow, selector string) (TermRow, error) {
	selector = strings.TrimSpace(selector)
	if number, err := strconv.Atoi(selector); err == nil {
		if number < 1 || number > len(rows) {
			return TermRow{}, fmt.Errorf("학기 번호가 범위를 벗어났습니다: %d", number)
		}
		return rows[number-1], nil
	}

	var matched *TermRow
	for index := range rows {
		if rows[index].Term.Value == selector || strings.Contains(rows[index].Term.Label, selector) {
			if matched != nil {
				return TermRow{}, fmt.Errorf("학기명이 여러 개와 일치합니다. term list 번호를 사용하세요: %s", selector)
			}
			matched = &rows[index]
		}
	}
	if matched == nil {
		return TermRow{}, fmt.Errorf("학기를 찾을 수 없습니다: %s", selector)
	}
	return *matched, nil
}

func (s *Service) CourseList(ctx context.Context, opts CourseListOptions) ([]klas.Term, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}

	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, err
	}

	term, _, err := s.selectedTerm(ctx, studentID, client)
	if err != nil {
		return nil, err
	}
	cacheKey := listCacheKey("course", studentID, term.Value, "")
	if !opts.Refresh {
		var cached []klas.Term
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}
	result := []klas.Term{term}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) Search(ctx context.Context, opts SearchOptions) (SearchResult, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return SearchResult{}, errors.New("검색어를 입력해야 합니다")
	}
	searchType := strings.ToLower(strings.TrimSpace(opts.Type))
	if searchType != "" && !validSearchType(searchType) {
		return SearchResult{}, fmt.Errorf("지원하지 않는 검색 타입입니다: %s", opts.Type)
	}

	result := SearchResult{
		Query: query,
		Type:  searchType,
	}
	user := UserOption{StudentID: strings.TrimSpace(opts.User.StudentID)}

	if searchType == "" || searchType == "course" {
		terms, err := s.CourseList(ctx, CourseListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "과목", Err: err})
		} else if len(terms) > 0 {
			for index, course := range terms[0].Courses {
				if textMatches(query, course.Name, course.Value) {
					result.Courses = append(result.Courses, SearchCourseResult{
						Index:  index + 1,
						Term:   terms[0],
						Course: course,
					})
				}
			}
		}
	}

	if searchType == "" || searchType == "assignment" {
		rows, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "과제", Err: err})
		} else {
			for _, row := range rows {
				if textMatches(query, row.ID, row.CourseName, row.Assignment.Title) {
					result.Assignments = append(result.Assignments, row)
				}
			}
		}
	}

	if searchType == "" || searchType == "notice" {
		rows, err := s.NoticeList(ctx, NoticeListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "공지", Err: err})
		} else {
			for _, row := range rows {
				if textMatches(query, row.ID, row.CourseName, row.Notice.Title, row.Notice.Author) {
					result.Notices = append(result.Notices, row)
				}
			}
		}
	}

	if searchType == "" || searchType == "lecture" {
		rows, err := s.LectureList(ctx, LectureListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "온라인 강의", Err: err})
		} else {
			for _, row := range rows {
				if textMatches(query, row.ID, row.CourseName, row.Lecture.ModuleTitle, row.Lecture.Title) {
					result.Lectures = append(result.Lectures, row)
				}
			}
		}
	}

	if searchType == "" || searchType == "academic" {
		academic, err := s.AcademicList(ctx, AcademicListOptions{Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "학사일정", Err: err})
		} else {
			for _, event := range academic.Events {
				if textMatches(query, event.Year, event.Month, event.Date, event.Title, event.Note) {
					result.Academics = append(result.Academics, event)
				}
			}
		}
	}

	return result, nil
}

func (s *Service) Due(ctx context.Context, opts DueOptions) (DueResult, error) {
	days := opts.Days
	if days <= 0 {
		days = 14
	}
	now := time.Now()
	result := DueResult{
		From:  now,
		Until: now.Add(time.Duration(days) * 24 * time.Hour),
	}
	user := UserOption{StudentID: strings.TrimSpace(opts.User.StudentID)}

	assignments, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.Errors = append(result.Errors, DashboardSectionError{Section: "과제", Err: err})
	} else {
		for _, row := range assignments {
			if row.Assignment.Submitted || row.Assignment.DueAt == nil || !inDueWindow(*row.Assignment.DueAt, result.From, result.Until) {
				continue
			}
			result.Items = append(result.Items, DueItem{
				Kind:       "과제",
				ID:         row.ID,
				CourseName: row.CourseName,
				Title:      row.Assignment.Title,
				DueAt:      *row.Assignment.DueAt,
				Status:     "미제출",
			})
		}
	}

	lectures, err := s.LectureList(ctx, LectureListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.Errors = append(result.Errors, DashboardSectionError{Section: "온라인 강의", Err: err})
	} else {
		for _, row := range lectures {
			if !lectureNeedsAttendance(row.Lecture, now) || row.Lecture.EndAt == nil || !inDueWindow(*row.Lecture.EndAt, result.From, result.Until) {
				continue
			}
			result.Items = append(result.Items, DueItem{
				Kind:       "온라인 강의",
				ID:         row.ID,
				CourseName: row.CourseName,
				Title:      firstNonEmpty(row.Lecture.Title, row.Lecture.ModuleTitle),
				DueAt:      *row.Lecture.EndAt,
				Status:     lectureDueStatus(row.Lecture),
			})
		}
	}

	academic, err := s.AcademicList(ctx, AcademicListOptions{Refresh: opts.Refresh})
	if err != nil {
		result.Errors = append(result.Errors, DashboardSectionError{Section: "학사일정", Err: err})
	} else {
		for _, event := range academic.Events {
			dueAt, ok := academicEventDueAt(event)
			if !ok || !inDueWindow(dueAt, result.From, result.Until) {
				continue
			}
			result.Items = append(result.Items, DueItem{
				Kind:   "학사일정",
				Title:  event.Title,
				DueAt:  dueAt,
				Status: event.Note,
			})
		}
	}

	sort.SliceStable(result.Items, func(i, j int) bool {
		if result.Items[i].DueAt.Equal(result.Items[j].DueAt) {
			return result.Items[i].Title < result.Items[j].Title
		}
		return result.Items[i].DueAt.Before(result.Items[j].DueAt)
	})
	return result, nil
}

func (s *Service) Dashboard(ctx context.Context, opts DashboardOptions) (DashboardResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return DashboardResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	_ = client
	if err != nil {
		return DashboardResult{}, err
	}

	cacheKey := dashboardCacheKey(studentID, term.Value)
	if !opts.Refresh {
		var cached DashboardResult
		hit, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached)
		if cacheErr == nil && ok {
			result, migrated, migrationErr := normalizeCachedDashboardResult(cached, term)
			if migrationErr == nil {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, dashboardCacheTTL(), result)
				}
				result.Cached = true
				result.CacheCreatedAt = hit.CreatedAt
				return result, nil
			}
		}
	}

	user := UserOption{StudentID: studentID}
	result := DashboardResult{
		Term:        term,
		GeneratedAt: time.Now(),
	}
	var assignmentRows []AssignmentRow
	var lectureRows []LectureRow
	var noticeRows []NoticeRow

	assignments, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "과제", Err: err})
	} else {
		assignmentRows = assignments
		result.Assignments = dashboardAssignments(assignments, 5)
	}

	lectures, err := s.LectureList(ctx, LectureListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "온라인 강의", Err: err})
	} else {
		lectureRows = lectures
		result.Lectures = dashboardLectures(lectures, time.Now(), 5)
	}

	notices, err := s.NoticeList(ctx, NoticeListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "공지", Err: err})
	} else {
		noticeRows = notices
		result.Notices = dashboardNotices(notices, 5)
	}

	attendance, err := s.AttendanceList(ctx, AttendanceListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "출석", Err: err})
	} else {
		result.Attendance = dashboardAttendance(attendance.Rows)
	}

	evaluation, err := s.EvaluationList(ctx, EvaluationListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "수업평가", Err: err})
	} else {
		result.Evaluation = dashboardEvaluation(evaluation)
	}
	result.Courses = dashboardCourses(term.Courses, assignmentRows, lectureRows, noticeRows, result.Attendance, result.Evaluation)

	if dashboardCacheable(result) {
		cacheValue := result
		cacheValue.Cached = false
		cacheValue.CacheCreatedAt = time.Time{}
		_ = s.cacheStore.Set(cacheKey, dashboardCacheTTL(), cacheValue)
	}
	return result, nil
}

func (s *Service) AssignmentList(ctx context.Context, opts AssignmentListOptions) ([]AssignmentRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return nil, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return nil, err
	}

	cacheKey := courseResourceListCacheKeyVersion("assignment", "v2", studentID, term.Value, courses)
	if !opts.Refresh {
		var cached []AssignmentRow
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			rows, migrated, migrationErr := normalizeCachedAssignmentRows(cached, term)
			if migrationErr == nil && resourceIDsMatchSelected(assignmentRowIDs(rows), "assignment", 1, courses, true) {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
				}
				return rows, nil
			}
		}
	}

	rows := make([]AssignmentRow, 0)
	for _, selectedCourse := range courses {
		assignments, err := client.Assignments(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
			if refreshErr != nil {
				return nil, refreshErr
			}
			if refreshed {
				client = refreshedClient
				assignments, err = client.Assignments(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return nil, err
		}
		for _, assignment := range assignments {
			ref, err := NewCourseRef(term.Value, selectedCourse.Course)
			if err != nil {
				return nil, err
			}
			id, err := StableAssignmentID(ref, assignment.OrdSeq)
			if err != nil {
				return nil, err
			}
			rows = append(rows, AssignmentRow{
				ID:         id,
				LegacyID:   AssignmentID(selectedCourse.Index, assignment.OrdSeq),
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				DetailURL:  assignmentDetailURL(term.Value, selectedCourse.Course, assignment.OrdSeq),
				Assignment: assignment,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		left := rows[i].Assignment.DueAt
		right := rows[j].Assignment.DueAt
		if left == nil && right == nil {
			return rows[i].ID < rows[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return left.Before(*right)
	})

	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
	return rows, nil
}

func (s *Service) AssignmentDetail(ctx context.Context, id string, user UserOption) (AssignmentDetailResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return AssignmentDetailResult{}, err
	}

	locator, ordSeq, err := parseAssignmentResourceID(id)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	var term klas.Term
	if locator.Stable {
		term, client, err = s.termForSyllabus(ctx, studentID, client, locator.Ref.TermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	course, err := resolveResourceCourse(term, locator)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	ref, err := NewCourseRef(term.Value, course)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	canonicalID, err := StableAssignmentID(ref, ordSeq)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	detail, err := client.AssignmentDetail(ctx, term.Value, course, ordSeq)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return AssignmentDetailResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			detail, err = client.AssignmentDetail(ctx, term.Value, course, ordSeq)
		}
	}
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	if detail.DueAt == nil || detail.StartAt == nil {
		assignments, listErr := client.Assignments(ctx, term.Value, course)
		if listErr == nil {
			for _, assignment := range assignments {
				if strings.TrimSpace(assignment.OrdSeq) != ordSeq {
					continue
				}
				if detail.DueAt == nil {
					detail.DueAt = assignment.DueAt
				}
				if detail.StartAt == nil {
					detail.StartAt = assignment.StartAt
				}
				break
			}
		}
	}

	return AssignmentDetailResult{
		ID:         canonicalID,
		TermValue:  term.Value,
		CourseName: course.Name,
		DetailURL:  assignmentDetailURL(term.Value, course, ordSeq),
		Detail:     detail,
	}, nil
}

func (s *Service) AssignmentOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	detail, err := s.AssignmentDetail(ctx, id, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	return OpenURLResult{URL: detail.DetailURL}, nil
}

func (s *Service) NoticeList(ctx context.Context, opts NoticeListOptions) ([]NoticeRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return nil, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return nil, err
	}

	cacheKey := courseResourceListCacheKeyVersion("notice", "v2", studentID, term.Value, courses)
	if !opts.Refresh {
		var cached []NoticeRow
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			rows, migrated, migrationErr := normalizeCachedNoticeRows(cached, term)
			if migrationErr == nil && resourceIDsMatchSelected(noticeRowIDs(rows), "notice", 2, courses, true) {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
				}
				return rows, nil
			}
		}
	}

	rows := make([]NoticeRow, 0)
	for _, selectedCourse := range courses {
		notices, err := client.Notices(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
			if refreshErr != nil {
				return nil, refreshErr
			}
			if refreshed {
				client = refreshedClient
				notices, err = client.Notices(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return nil, err
		}
		for _, notice := range notices {
			ref, err := NewCourseRef(term.Value, selectedCourse.Course)
			if err != nil {
				return nil, err
			}
			id, err := StableNoticeID(ref, notice.BoardNo, notice.MasterNo)
			if err != nil {
				return nil, err
			}
			rows = append(rows, NoticeRow{
				ID:         id,
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				DetailURL:  noticeDetailURL(term.Value, selectedCourse.Course, notice.BoardNo, notice.MasterNo),
				Notice:     notice,
			})
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Notice.Top != rows[j].Notice.Top {
			return rows[i].Notice.Top
		}
		left := rows[i].Notice.Registered
		right := rows[j].Notice.Registered
		if left == nil && right == nil {
			return rows[i].ID < rows[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return right.Before(*left)
	})

	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
	return rows, nil
}

func (s *Service) NoticeDetail(ctx context.Context, id string, user UserOption) (NoticeDetailResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return NoticeDetailResult{}, err
	}

	locator, boardNo, masterNo, err := parseNoticeResourceID(id)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	var term klas.Term
	if locator.Stable {
		term, client, err = s.termForSyllabus(ctx, studentID, client, locator.Ref.TermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return NoticeDetailResult{}, err
	}
	course, err := resolveResourceCourse(term, locator)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	ref, err := NewCourseRef(term.Value, course)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	canonicalID, err := StableNoticeID(ref, boardNo, masterNo)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	detail, err := client.NoticeDetail(ctx, term.Value, course, boardNo, masterNo)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return NoticeDetailResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			detail, err = client.NoticeDetail(ctx, term.Value, course, boardNo, masterNo)
		}
	}
	if err != nil {
		return NoticeDetailResult{}, err
	}

	return NoticeDetailResult{
		ID:         canonicalID,
		TermValue:  term.Value,
		CourseName: course.Name,
		DetailURL:  noticeDetailURL(term.Value, course, boardNo, masterNo),
		Detail:     detail,
	}, nil
}

func (s *Service) NoticeOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	detail, err := s.NoticeDetail(ctx, id, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	return OpenURLResult{URL: detail.DetailURL}, nil
}

func (s *Service) Timetable(ctx context.Context, opts TimetableOptions) (TimetableResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return TimetableResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return TimetableResult{}, err
	}

	cacheKey := listCacheKeyVersion("timetable", "v2", studentID, term.Value, "")
	if !opts.Refresh {
		var cached TimetableResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	entries, err := client.Timetable(ctx, term.Value)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return TimetableResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			entries, err = client.Timetable(ctx, term.Value)
		}
	}
	if err != nil {
		return TimetableResult{}, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Online != entries[j].Online {
			return !entries[i].Online
		}
		if entries[i].Weekday != entries[j].Weekday {
			return entries[i].Weekday < entries[j].Weekday
		}
		if entries[i].Period != entries[j].Period {
			return entries[i].Period < entries[j].Period
		}
		return entries[i].SubjectName < entries[j].SubjectName
	})

	result := TimetableResult{
		Term:    term,
		Entries: entries,
	}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) AttendanceList(ctx context.Context, opts AttendanceListOptions) (AttendanceListResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return AttendanceListResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return AttendanceListResult{}, err
	}

	cacheKey := listCacheKeyVersion("attendance", "v2", studentID, term.Value, "")
	if !opts.Refresh {
		var cached AttendanceListResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	courses, err := client.AttendanceCourses(ctx, term.Value)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return AttendanceListResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			courses, err = client.AttendanceCourses(ctx, term.Value)
		}
	}
	if err != nil {
		return AttendanceListResult{}, err
	}

	rows := make([]AttendanceRow, 0, len(courses))
	for index, course := range courses {
		sessions, detailErr := client.AttendanceSessions(ctx, term.Value, course)
		if detailErr != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, detailErr)
			if refreshErr != nil {
				detailErr = refreshErr
			} else if refreshed {
				client = refreshedClient
				sessions, detailErr = client.AttendanceSessions(ctx, term.Value, course)
			}
		}
		rows = append(rows, AttendanceRow{
			Index:    index + 1,
			Course:   course,
			Sessions: sessions,
			Err:      detailErr,
		})
	}
	result := AttendanceListResult{
		Term: term,
		Rows: rows,
	}
	if attendanceCacheable(result) {
		_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	}
	return result, nil
}

func (s *Service) AttendanceDetail(ctx context.Context, opts AttendanceDetailOptions) (AttendanceDetailResult, error) {
	selector := strings.TrimSpace(opts.Selector)
	if selector == "" {
		return AttendanceDetailResult{}, errors.New("출석 상세 조회 대상이 없습니다")
	}
	result, err := s.AttendanceList(ctx, AttendanceListOptions{User: opts.User})
	if err != nil {
		return AttendanceDetailResult{}, err
	}
	rows := selectAttendanceRows(result.Rows, selector)
	if len(rows) == 0 {
		return AttendanceDetailResult{}, fmt.Errorf("출석 현황 과목을 찾을 수 없습니다: %s", selector)
	}
	if len(rows) > 1 {
		return AttendanceDetailResult{}, fmt.Errorf("출석 현황 과목이 여러 개와 일치합니다. 번호 또는 학정번호를 사용하세요: %s", selector)
	}
	return AttendanceDetailResult{
		Term: result.Term,
		Row:  rows[0],
	}, nil
}

func (s *Service) CdpAttendance(ctx context.Context, opts CdpAttendanceOptions) (CdpAttendanceResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return CdpAttendanceResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return CdpAttendanceResult{}, err
	}

	report, err := client.CdpAttendance(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return CdpAttendanceResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			report, err = client.CdpAttendance(ctx)
		}
	}
	if err != nil {
		return CdpAttendanceResult{}, err
	}
	return CdpAttendanceResult{Report: report}, nil
}

func (s *Service) Grade(ctx context.Context, opts GradeOptions) (GradeResult, error) {
	termValue, err := normalizeTermValue(opts.TermValue)
	if err != nil {
		return GradeResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return GradeResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return GradeResult{}, err
	}

	cacheKey := listCacheKeyVersion("grade", "v2", studentID, termValue, "")
	if !opts.Refresh {
		var cached GradeResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	report, err := client.Grades(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return GradeResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			report, err = client.Grades(ctx)
		}
	}
	if err != nil {
		return GradeResult{}, err
	}
	if termValue != "" {
		filtered := report.Terms[:0]
		for _, term := range report.Terms {
			if term.Year+","+term.Hakgi == termValue {
				filtered = append(filtered, term)
			}
		}
		report.Terms = filtered
	}
	result := GradeResult{Report: report, TermValue: termValue}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) Rank(ctx context.Context, opts RankOptions) (RankResult, error) {
	termValue, err := normalizeTermValue(opts.TermValue)
	if err != nil {
		return RankResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return RankResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return RankResult{}, err
	}

	cacheKey := listCacheKeyVersion("rank", "v2", studentID, termValue, "")
	if !opts.Refresh {
		var cached RankResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	rows, err := client.Ranks(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return RankResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			rows, err = client.Ranks(ctx)
		}
	}
	if err != nil {
		return RankResult{}, err
	}
	if termValue != "" {
		filtered := rows[:0]
		for _, row := range rows {
			if row.TermValue == termValue {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	result := RankResult{Rows: rows}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) EvaluationList(ctx context.Context, opts EvaluationListOptions) (EvaluationListResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return EvaluationListResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return EvaluationListResult{}, err
	}

	cacheKey := listCacheKeyVersion("evaluation", "v2", studentID, "", "")
	if !opts.Refresh {
		var cached EvaluationListResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	term, courses, _, err := s.evaluationCourses(ctx, studentID, client)
	if err != nil {
		return EvaluationListResult{}, err
	}
	rows := makeEvaluationRows(courses)
	result := EvaluationListResult{Term: term, Rows: rows}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) EvaluationSubmit(ctx context.Context, opts EvaluationSubmitOptions) (EvaluationSubmitResult, error) {
	selector := strings.TrimSpace(opts.Selector)
	if selector == "" {
		return EvaluationSubmitResult{}, errors.New("수업평가 제출 대상이 없습니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}

	term, courses, client, err := s.evaluationCourses(ctx, studentID, client)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}
	targets, err := selectEvaluationRows(makeEvaluationRows(courses), selector)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}

	result := EvaluationSubmitResult{
		Term:      term,
		Submitted: opts.Confirm,
		Items:     make([]EvaluationSubmitItem, 0, len(targets)),
	}
	answerOpts := klas.EvaluationAnswerOptions{
		Choice:             "5",
		Text:               "많은 도움 되었습니다. 한학기동안 감사했습니다.",
		DiscriminationOpt:  "N",
		IncludeEngineering: opts.IncludeEngineering,
	}
	for _, target := range targets {
		item := EvaluationSubmitItem{Row: target}
		if target.Course.Evaluated {
			item.Skipped = true
			item.Reason = "이미 평가 완료"
			result.Items = append(result.Items, item)
			continue
		}

		form, formErr := client.EvaluationForm(ctx, term, target.Course)
		if formErr != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, formErr)
			if refreshErr != nil {
				formErr = refreshErr
			} else if refreshed {
				client = refreshedClient
				form, formErr = client.EvaluationForm(ctx, term, target.Course)
			}
		}
		if formErr != nil {
			item.Err = formErr
			result.Items = append(result.Items, item)
			continue
		}
		if !opts.Confirm {
			result.Items = append(result.Items, item)
			continue
		}
		if _, submitErr := client.SubmitEvaluation(ctx, term, target.Course, form, answerOpts); submitErr != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, submitErr)
			if refreshErr != nil {
				submitErr = refreshErr
			} else if refreshed {
				client = refreshedClient
				_, submitErr = client.SubmitEvaluation(ctx, term, target.Course, form, answerOpts)
			}
			item.Err = submitErr
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (s *Service) evaluationCourses(ctx context.Context, studentID string, client *klas.Client) (klas.EvaluationTerm, []klas.EvaluationCourse, *klas.Client, error) {
	term, err := client.EvaluationTerm(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return klas.EvaluationTerm{}, nil, client, refreshErr
		}
		if refreshed {
			client = refreshedClient
			term, err = client.EvaluationTerm(ctx)
		}
	}
	if err != nil {
		return klas.EvaluationTerm{}, nil, client, err
	}
	if !term.TermEnabled {
		return term, nil, client, nil
	}

	if _, err := client.EvaluationStudent(ctx); err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return klas.EvaluationTerm{}, nil, client, refreshErr
		}
		if refreshed {
			client = refreshedClient
			_, err = client.EvaluationStudent(ctx)
		}
		if err != nil {
			return klas.EvaluationTerm{}, nil, client, err
		}
	}

	courses, err := client.EvaluationCourses(ctx, term)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return klas.EvaluationTerm{}, nil, client, refreshErr
		}
		if refreshed {
			client = refreshedClient
			courses, err = client.EvaluationCourses(ctx, term)
		}
	}
	if err != nil {
		return klas.EvaluationTerm{}, nil, client, err
	}
	return term, courses, client, nil
}

func (s *Service) Syllabus(ctx context.Context, opts SyllabusOptions) (SyllabusResult, error) {
	selector := strings.TrimSpace(opts.Selector)
	if selector == "" {
		return SyllabusResult{}, errors.New("강의계획서 조회 대상이 없습니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return SyllabusResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return SyllabusResult{}, err
	}

	term, client, err := s.termForSyllabus(ctx, studentID, client, opts.TermValue)
	if err != nil {
		return SyllabusResult{}, err
	}

	course := klas.Course{Name: selector}
	subjectID := ""
	if looksLikeSyllabusCourseCode(selector) {
		subjectID, err = klas.SyllabusSubjectIDFromCourseCode(term.Value, selector)
		if err != nil {
			return SyllabusResult{}, err
		}
	} else {
		courses, err := selectedCourses(term, selector)
		if err != nil {
			return SyllabusResult{}, err
		}
		if len(courses) != 1 {
			return SyllabusResult{}, fmt.Errorf("강의계획서 조회 대상이 여러 개입니다: %s", selector)
		}
		course = courses[0].Course
		subjectID = course.Value
	}

	syllabus, err := client.SyllabusBySubjectID(ctx, subjectID)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return SyllabusResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			syllabus, err = client.SyllabusBySubjectID(ctx, subjectID)
		}
	}
	if err != nil {
		return SyllabusResult{}, err
	}
	if strings.TrimSpace(course.Name) == "" || course.Name == selector {
		course.Name = firstNonEmpty(syllabus.KoreanName, syllabus.FullName, selector)
	}
	if strings.TrimSpace(course.Value) == "" {
		course.Value = subjectID
	}

	return SyllabusResult{
		Term:      term,
		SubjectID: subjectID,
		Course:    course,
		Syllabus:  syllabus,
	}, nil
}

func (s *Service) SubjectSearch(ctx context.Context, opts SubjectSearchOptions) (SubjectSearchResult, error) {
	name := strings.TrimSpace(opts.Name)
	professor := strings.TrimSpace(opts.Professor)
	if name == "" && professor == "" {
		return SubjectSearchResult{}, errors.New("과목명 또는 교수명을 입력해야 합니다")
	}

	studentID, err := s.selectedStudentID(ctx, UserOption{})
	if err != nil {
		return SubjectSearchResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return SubjectSearchResult{}, err
	}
	term, err := s.termForSubjectSearch(opts.TermValue)
	if err != nil {
		return SubjectSearchResult{}, err
	}

	items, err := client.SyllabusList(ctx, term.Value, name, professor)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return SubjectSearchResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			items, err = client.SyllabusList(ctx, term.Value, name, professor)
		}
	}
	if err != nil {
		return SubjectSearchResult{}, err
	}

	rows := make([]SubjectSearchRow, 0, len(items))
	for _, item := range items {
		subjectID, idErr := item.SubjectID()
		row := SubjectSearchRow{
			CourseCode: item.CourseCode(),
			SubjectID:  subjectID,
			Name:       strings.TrimSpace(item.KoreanName),
			Professor:  strings.TrimSpace(item.Professor),
			Err:        idErr,
		}
		if idErr == nil {
			syllabus, detailErr := client.SyllabusBySubjectID(ctx, subjectID)
			if detailErr != nil {
				refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, detailErr)
				if refreshErr != nil {
					row.Err = refreshErr
				} else if refreshed {
					client = refreshedClient
					syllabus, detailErr = client.SyllabusBySubjectID(ctx, subjectID)
					row.Err = detailErr
				} else {
					row.Err = detailErr
				}
			}
			if row.Err == nil {
				row.Times = syllabus.Times
				if strings.TrimSpace(row.Name) == "" {
					row.Name = firstNonEmpty(syllabus.KoreanName, syllabus.FullName)
				}
				if strings.TrimSpace(row.Professor) == "" {
					row.Professor = syllabus.Professor
				}
			}
		}
		rows = append(rows, row)
	}

	return SubjectSearchResult{
		Term: term,
		Rows: rows,
	}, nil
}

func (s *Service) LectureList(ctx context.Context, opts LectureListOptions) ([]LectureRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return nil, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return nil, err
	}

	cacheKey := courseResourceListCacheKey("lecture", studentID, term.Value, courses)
	legacyCacheKey := listCacheKey("lecture", studentID, term.Value, opts.CourseFilter)
	if !opts.Refresh {
		var cached []LectureRow
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			rows, migrated, migrationErr := normalizeCachedLectureRows(cached, term)
			if migrationErr == nil && resourceIDsMatchSelected(lectureRowIDs(rows), "lecture", 1, courses, true) {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
				}
				return rows, nil
			}
		}
		if legacyCacheKey != cacheKey {
			cached = nil
			if _, ok, cacheErr := s.cacheStore.Get(legacyCacheKey, &cached); cacheErr == nil && ok {
				rows, _, migrationErr := normalizeCachedLectureRows(cached, term)
				allowEmpty := strings.TrimSpace(opts.CourseFilter) == ""
				if migrationErr == nil && resourceIDsMatchSelected(lectureRowIDs(rows), "lecture", 1, courses, allowEmpty) {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
					return rows, nil
				}
			}
		}
	}

	rows := make([]LectureRow, 0)
	for _, selectedCourse := range courses {
		lectures, err := client.Lectures(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
			if refreshErr != nil {
				return nil, refreshErr
			}
			if refreshed {
				client = refreshedClient
				lectures, err = client.Lectures(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return nil, err
		}
		for _, lecture := range lectures {
			row, err := newLectureRow(term.Value, selectedCourse, lecture)
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i].Lecture.StartAt
		right := rows[j].Lecture.StartAt
		if left == nil && right == nil {
			return rows[i].ID < rows[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return rows[i].Lecture.Title < rows[j].Lecture.Title
		}
		return left.Before(*right)
	})

	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
	return rows, nil
}

func (s *Service) LectureOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	resource, err := s.resolveLectureResource(ctx, studentID, id)
	if err != nil {
		return OpenURLResult{}, err
	}
	lectures, err := resource.Client.Lectures(ctx, resource.Term.Value, resource.Course)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return OpenURLResult{}, refreshErr
		}
		if refreshed {
			resource.Client = refreshedClient
			lectures, err = resource.Client.Lectures(ctx, resource.Term.Value, resource.Course)
		}
	}
	if err != nil {
		return OpenURLResult{}, err
	}

	for _, lecture := range lectures {
		if !lectureMatchesKey(lecture, resource.Key) {
			continue
		}
		if strings.TrimSpace(lecture.PlayURL) == "" {
			return OpenURLResult{}, fmt.Errorf("열 수 있는 강의 URL이 없습니다: %s", id)
		}
		return OpenURLResult{URL: lecture.PlayURL}, nil
	}
	return OpenURLResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
}

func (s *Service) DownloadLecture(ctx context.Context, id string, opts LectureDownloadOptions) (LectureDownloadResult, error) {
	defer s.startCaffeinate(ctx)()
	currentSettings, err := s.loadSettings()
	if err != nil {
		return LectureDownloadResult{}, err
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	resource, err := s.resolveLectureResource(ctx, studentID, id)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	lectures, err := resource.Client.Lectures(ctx, resource.Term.Value, resource.Course)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return LectureDownloadResult{}, refreshErr
		}
		if refreshed {
			resource.Client = refreshedClient
			lectures, err = resource.Client.Lectures(ctx, resource.Term.Value, resource.Course)
		}
	}
	if err != nil {
		return LectureDownloadResult{}, err
	}

	var matched *klas.Lecture
	for index := range lectures {
		if strings.TrimSpace(lectures[index].ContentID) == resource.Key {
			matched = &lectures[index]
			break
		}
	}
	if matched == nil {
		return LectureDownloadResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
	}
	row := LectureRow{
		ID:         resource.ID,
		LegacyID:   resource.LegacyID,
		TermValue:  resource.Term.Value,
		CourseName: resource.Course.Name,
		Lecture:    *matched,
	}
	weekOrder := lectureWeekOrder(lectures, *matched)
	emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
		Lecture:      row,
		Stage:        "resolve",
		CurrentIndex: 1,
		TotalItems:   1,
	})

	mediaURL, err := resource.Client.ResolveLectureMediaURL(ctx, resource.Key)
	if err != nil {
		return LectureDownloadResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	dir, err = s.effectiveDownloadDir(dir)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return LectureDownloadResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}

	path := lectureVideoPath(dir, resource.Course.Name, *matched, mediaURL, weekOrder)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return LectureDownloadResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}
	emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         path,
		Stage:        "download",
		CurrentIndex: 1,
		TotalItems:   1,
	})
	bytesWritten, err := downloadFile(ctx, mediaURL, path, currentSettings.Download.KeepPartial, func(bytesWritten int64, totalBytes int64) {
		emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         path,
			Stage:        "download",
			CurrentIndex: 1,
			TotalItems:   1,
			Bytes:        bytesWritten,
			TotalBytes:   totalBytes,
		})
	})
	if err != nil {
		emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         path,
			Stage:        "error",
			CurrentIndex: 1,
			TotalItems:   1,
			Bytes:        bytesWritten,
			Err:          err,
		})
		return LectureDownloadResult{}, err
	}
	emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         path,
		Stage:        "done",
		CurrentIndex: 1,
		TotalItems:   1,
		Bytes:        bytesWritten,
		TotalBytes:   bytesWritten,
	})
	return LectureDownloadResult{
		Path:     path,
		Bytes:    bytesWritten,
		Lecture:  row,
		MediaURL: mediaURL,
	}, nil
}

func (s *Service) DownloadAllLectures(ctx context.Context, opts LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
	defer s.startCaffeinate(ctx)()
	currentSettings, err := s.loadSettings()
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	if strings.TrimSpace(opts.CourseFilter) == "" && len(opts.LectureIDs) == 0 {
		return LectureDownloadAllResult{}, errors.New("전체 다운로드에는 과목명 또는 course list 번호가 필요합니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	stableTermValue, err := stableLectureTermValue(opts.LectureIDs)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	var term klas.Term
	if stableTermValue != "" {
		term, client, err = s.termForSyllabus(ctx, studentID, client, stableTermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	dir, err = s.effectiveDownloadDir(dir)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return LectureDownloadAllResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}

	concurrency, err := s.effectiveDownloadConcurrency(opts.Concurrency)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	selectedIDs := make(map[string]struct{}, len(opts.LectureIDs))
	for _, id := range opts.LectureIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			selectedIDs[id] = struct{}{}
		}
	}

	type downloadTask struct {
		Index     int
		Total     int
		Course    klas.Course
		Row       LectureRow
		WeekOrder int
	}
	tasks := make([]downloadTask, 0)
	for _, selectedCourse := range courses {
		lectures, err := client.Lectures(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
			if refreshErr != nil {
				return LectureDownloadAllResult{}, refreshErr
			}
			if refreshed {
				client = refreshedClient
				lectures, err = client.Lectures(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return LectureDownloadAllResult{}, err
		}

		rows := make([]LectureRow, 0, len(lectures))
		for _, lecture := range lectures {
			row, err := newLectureRow(term.Value, selectedCourse, lecture)
			if err != nil {
				return LectureDownloadAllResult{}, err
			}
			rows = append(rows, row)
		}
		weekOrders := lectureRowWeekOrders(rows)
		for _, row := range rows {
			if len(selectedIDs) > 0 {
				_, stableSelected := selectedIDs[row.ID]
				_, legacySelected := selectedIDs[row.LegacyID]
				if !stableSelected && !legacySelected {
					continue
				}
			}
			tasks = append(tasks, downloadTask{
				Course:    selectedCourse.Course,
				Row:       row,
				WeekOrder: weekOrders[row.ID],
			})
		}
	}
	for index := range tasks {
		tasks[index].Index = index + 1
		tasks[index].Total = len(tasks)
	}
	if len(tasks) == 0 {
		return LectureDownloadAllResult{}, nil
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}

	type downloadTaskResult struct {
		Index int
		Item  LectureDownloadItem
	}
	jobs := make(chan downloadTask)
	results := make(chan downloadTaskResult, len(tasks))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for task := range jobs {
			results <- downloadTaskResult{Index: task.Index - 1, Item: downloadLectureTask(ctx, client, dir, currentSettings.Download.KeepPartial, task.Index, task.Total, task.Course, task.Row, task.WeekOrder, opts.OnProgress)}
		}
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	go func() {
		defer close(jobs)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case jobs <- task:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	result := LectureDownloadAllResult{Items: make([]LectureDownloadItem, len(tasks))}
	for item := range results {
		result.Items[item.Index] = item.Item
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func downloadLectureTask(ctx context.Context, client *klas.Client, dir string, keepPartial bool, index int, total int, course klas.Course, row LectureRow, weekOrder int, onProgress func(LectureDownloadProgress)) LectureDownloadItem {
	item := LectureDownloadItem{Lecture: row}
	if strings.TrimSpace(row.Lecture.ContentID) == "" {
		item.Skipped = true
		item.Err = errors.New("KWCommons 콘텐츠 ID가 없습니다")
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Stage:        "skip",
			CurrentIndex: index,
			TotalItems:   total,
			Skipped:      true,
			Err:          item.Err,
		})
		return item
	}

	emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
		Lecture:      row,
		Stage:        "resolve",
		CurrentIndex: index,
		TotalItems:   total,
	})
	mediaURL, err := client.ResolveLectureMediaURL(ctx, row.Lecture.ContentID)
	if err != nil {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Stage:        "error",
			CurrentIndex: index,
			TotalItems:   total,
			Err:          err,
		})
		return item
	}

	item.Path = lectureVideoPath(dir, course.Name, row.Lecture, mediaURL, weekOrder)
	if err := os.MkdirAll(filepath.Dir(item.Path), 0o755); err != nil {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        "error",
			CurrentIndex: index,
			TotalItems:   total,
			Err:          err,
		})
		return item
	}
	if _, err := os.Stat(item.Path); err == nil {
		item.Skipped = true
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        "skip",
			CurrentIndex: index,
			TotalItems:   total,
			Skipped:      true,
		})
		return item
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        "error",
			CurrentIndex: index,
			TotalItems:   total,
			Err:          err,
		})
		return item
	}

	emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         item.Path,
		Stage:        "download",
		CurrentIndex: index,
		TotalItems:   total,
	})
	bytesWritten, err := downloadFile(ctx, mediaURL, item.Path, keepPartial, func(bytesWritten int64, totalBytes int64) {
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        "download",
			CurrentIndex: index,
			TotalItems:   total,
			Bytes:        bytesWritten,
			TotalBytes:   totalBytes,
		})
	})
	item.Bytes = bytesWritten
	if err != nil {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        "error",
			CurrentIndex: index,
			TotalItems:   total,
			Bytes:        bytesWritten,
			Err:          err,
		})
		return item
	}
	emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         item.Path,
		Stage:        "done",
		CurrentIndex: index,
		TotalItems:   total,
		Bytes:        bytesWritten,
		TotalBytes:   bytesWritten,
	})
	return item
}

func (s *Service) TranscribeDownloadedLectures(ctx context.Context, items []LectureDownloadItem, opts LectureTranscriptOptions) LectureTranscriptResult {
	defer s.startCaffeinate(ctx)()

	jobs := make([]transcript.Job, 0, len(items))
	rows := make([]LectureRow, 0, len(items))
	failedItems := make([]LectureTranscriptItem, 0)
	for _, item := range items {
		if !LectureDownloadItemNeedsTranscript(item) {
			continue
		}
		outputPath := transcriptPath(item.Path)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			failedItems = append(failedItems, LectureTranscriptItem{
				Lecture:    item.Lecture,
				InputPath:  item.Path,
				OutputPath: outputPath,
				Err:        err,
			})
			emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
				Lecture:    item.Lecture,
				InputPath:  item.Path,
				OutputPath: outputPath,
				Stage:      "transcript-error",
				Err:        err,
			})
			continue
		}
		rows = append(rows, item.Lecture)
		jobs = append(jobs, transcript.Job{
			InputPath:         item.Path,
			OutputPath:        outputPath,
			Locale:            opts.Locale,
			ContextualStrings: lectureTranscriptContext(item.Lecture),
		})
		emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
			Lecture:    item.Lecture,
			InputPath:  item.Path,
			OutputPath: outputPath,
			Stage:      "transcribe",
		})
	}
	if len(jobs) == 0 {
		return LectureTranscriptResult{Items: failedItems}
	}

	select {
	case <-ctx.Done():
		result := LectureTranscriptResult{Items: append([]LectureTranscriptItem{}, failedItems...)}
		for index, job := range jobs {
			result.Items = append(result.Items, LectureTranscriptItem{
				Lecture:    rows[index],
				InputPath:  job.InputPath,
				OutputPath: job.OutputPath,
				Err:        ctx.Err(),
			})
		}
		return result
	default:
	}

	response, err := transcript.NewMacOSBridge(s.transcriptBridgePath).TranscribeWithProgress(ctx, transcript.Request{Jobs: jobs}, func(progress transcript.Progress) {
		index := transcriptJobIndex(jobs, progress.InputPath, progress.OutputPath)
		row := LectureRow{}
		if index >= 0 && index < len(rows) {
			row = rows[index]
		}
		emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
			Lecture:    row,
			InputPath:  progress.InputPath,
			OutputPath: progress.OutputPath,
			Stage:      "transcribe",
			Progress:   progress.Progress,
		})
	})
	if err != nil {
		result := LectureTranscriptResult{Items: append([]LectureTranscriptItem{}, failedItems...)}
		for index, job := range jobs {
			item := LectureTranscriptItem{
				Lecture:    rows[index],
				InputPath:  job.InputPath,
				OutputPath: job.OutputPath,
				Err:        err,
			}
			result.Items = append(result.Items, item)
			emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
				Lecture:    item.Lecture,
				InputPath:  item.InputPath,
				OutputPath: item.OutputPath,
				Stage:      "transcript-error",
				Err:        err,
			})
		}
		return result
	}

	result := LectureTranscriptResult{Items: append([]LectureTranscriptItem{}, failedItems...)}
	for index, bridgeResult := range response.Results {
		row := LectureRow{}
		if index < len(rows) {
			row = rows[index]
		}
		item := LectureTranscriptItem{
			Lecture:    row,
			InputPath:  bridgeResult.InputPath,
			OutputPath: bridgeResult.OutputPath,
			Text:       bridgeResult.Text,
		}
		stage := "transcribed"
		if strings.TrimSpace(bridgeResult.Err) != "" {
			item.Err = errors.New(bridgeResult.Err)
			stage = "transcript-error"
		}
		result.Items = append(result.Items, item)
		emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
			Lecture:    item.Lecture,
			InputPath:  item.InputPath,
			OutputPath: item.OutputPath,
			Stage:      stage,
			Err:        item.Err,
		})
	}
	return result
}

func transcriptJobIndex(jobs []transcript.Job, inputPath string, outputPath string) int {
	for index, job := range jobs {
		if strings.TrimSpace(inputPath) != "" && job.InputPath == inputPath {
			return index
		}
		if strings.TrimSpace(outputPath) != "" && job.OutputPath == outputPath {
			return index
		}
	}
	return -1
}

func transcriptPath(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(dir) == "video" {
		dir = filepath.Join(filepath.Dir(dir), "transcription")
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return filepath.Join(dir, filepath.Base(path)+".txt")
	}
	return filepath.Join(dir, strings.TrimSuffix(filepath.Base(path), ext)+".txt")
}

func TranscriptPathForDownload(path string) string {
	return transcriptPath(path)
}

func LectureDownloadItemNeedsTranscript(item LectureDownloadItem) bool {
	if item.Err != nil || strings.TrimSpace(item.Path) == "" {
		return false
	}
	if !item.Skipped {
		return true
	}
	_, err := os.Stat(transcriptPath(item.Path))
	return err != nil
}

func lectureTranscriptContext(row LectureRow) []string {
	values := []string{
		"광운대학교",
		"Kwangwoon University",
		"KLAS",
		"이 강의는 한국어와 영어 등 language switching code switching이 포함될 수 있습니다",
		"language switching",
		"code switching",
		row.CourseName,
		row.Lecture.ModuleTitle,
		row.Lecture.Title,
	}
	seen := make(map[string]struct{}, len(values))
	context := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		context = append(context, value)
	}
	return context
}

func emitLectureTranscriptProgress(onProgress func(LectureTranscriptProgress), progress LectureTranscriptProgress) {
	if onProgress != nil {
		onProgress(progress)
	}
}

func (s *Service) AttendLecture(ctx context.Context, id string, opts LectureAttendOptions) (LectureAttendResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureAttendResult{}, err
	}
	resource, err := s.resolveLectureResource(ctx, studentID, id)
	if err != nil {
		return LectureAttendResult{}, err
	}
	lectures, err := resource.Client.Lectures(ctx, resource.Term.Value, resource.Course)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return LectureAttendResult{}, refreshErr
		}
		if refreshed {
			resource.Client = refreshedClient
			lectures, err = resource.Client.Lectures(ctx, resource.Term.Value, resource.Course)
		}
	}
	if err != nil {
		return LectureAttendResult{}, err
	}

	var matched *klas.Lecture
	for index := range lectures {
		if lectureMatchesKey(lectures[index], resource.Key) {
			matched = &lectures[index]
			break
		}
	}
	if matched == nil {
		return LectureAttendResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
	}

	row := LectureRow{
		ID:         resource.ID,
		LegacyID:   resource.LegacyID,
		TermValue:  resource.Term.Value,
		CourseName: resource.Course.Name,
		Lecture:    *matched,
	}
	progress, err := attendLecture(ctx, resource.Client, row, opts.Interval, opts.OnProgress)
	if err != nil {
		return LectureAttendResult{}, err
	}
	return LectureAttendResult{Lecture: row, Progress: progress}, nil
}

func (s *Service) AttendAllLectures(ctx context.Context, opts LectureAttendAllOptions) (LectureAttendAllResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureAttendAllResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return LectureAttendAllResult{}, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return LectureAttendAllResult{}, err
	}

	result := LectureAttendAllResult{}
	now := time.Now()
	for _, selectedCourse := range courses {
		lectures, err := client.Lectures(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
			if refreshErr != nil {
				return LectureAttendAllResult{}, refreshErr
			}
			if refreshed {
				client = refreshedClient
				lectures, err = client.Lectures(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return LectureAttendAllResult{}, err
		}

		for _, lecture := range lectures {
			if !lectureNeedsAttendance(lecture, now) {
				continue
			}
			row, rowErr := newLectureRow(term.Value, selectedCourse, lecture)
			if rowErr != nil {
				return LectureAttendAllResult{}, rowErr
			}
			progress, err := attendLecture(ctx, client, row, opts.Interval, opts.OnProgress)
			result.Items = append(result.Items, LectureAttendItem{
				Lecture:  row,
				Progress: progress,
				Err:      err,
			})
		}
	}
	return result, nil
}

func (s *Service) SyncAssignmentReminders(ctx context.Context, opts AssignmentListOptions) (ReminderSyncResult, error) {
	rows, err := s.AssignmentList(ctx, opts)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return ReminderSyncResult{}, err
	}

	currentSettings, err := s.loadSettings()
	if err != nil {
		return ReminderSyncResult{}, err
	}

	now := time.Now()
	assignments := make([]reminder.Assignment, 0, len(rows))
	for _, row := range rows {
		if !futureTime(row.Assignment.DueAt, now) {
			continue
		}
		detail, detailErr := s.AssignmentDetail(ctx, row.ID, opts.User)
		if detailErr != nil {
			return ReminderSyncResult{}, detailErr
		}
		assignments = append(assignments, reminder.Assignment{
			ID:        detail.ID,
			LegacyIDs: compactNonEmpty([]string{row.LegacyID}),
			TermValue: detail.TermValue,
			Title:     detail.Detail.Title,
			Course:    detail.CourseName,
			DueAt:     detail.Detail.DueAt,
			Submitted: detail.Detail.Submitted,
			DetailURL: row.DetailURL,
			Notes:     buildReminderNotes(detail),
		})
	}

	if len(assignments) == 0 {
		return ReminderSyncResult{}, nil
	}
	prepared, err := s.prepareReminderSync("assignment", studentID, assignments, opts.SyncDecisions)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return ReminderSyncResult{EligibleCount: len(assignments), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Assignments) == 0 {
		if err := s.saveSyncSourceState("assignment", studentID, prepared.State); err != nil {
			return ReminderSyncResult{}, err
		}
		return ReminderSyncResult{Result: reminder.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(assignments)}, nil
	}

	result, err := reminder.NewMacOSBridge(s.reminderBridgePath).Sync(reminder.SyncRequest{
		ListName:        currentSettings.Reminder.ListName,
		UseExistingList: currentSettings.Reminder.UseExistingList,
		AlarmBeforeMin:  currentSettings.Reminder.AlarmBeforeMin,
		Assignments:     prepared.Assignments,
	})
	if err != nil {
		return ReminderSyncResult{}, err
	}
	result.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, result.SyncedIDs)
	if err := s.saveSyncSourceState("assignment", studentID, prepared.State); err != nil {
		return ReminderSyncResult{}, err
	}
	return ReminderSyncResult{
		Result:        result,
		EligibleCount: len(assignments),
	}, nil
}

func (s *Service) SyncLectureReminders(ctx context.Context, opts LectureListOptions) (ReminderSyncResult, error) {
	rows, err := s.LectureList(ctx, opts)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return ReminderSyncResult{}, err
	}

	currentSettings, err := s.loadSettings()
	if err != nil {
		return ReminderSyncResult{}, err
	}

	now := time.Now()
	assignments := make([]reminder.Assignment, 0, len(rows))
	for _, row := range rows {
		if !futureTime(row.Lecture.EndAt, now) {
			continue
		}
		assignments = append(assignments, reminder.Assignment{
			ID:        lectureReminderID(row.ID),
			LegacyIDs: compactNonEmpty([]string{lectureReminderID(row.LegacyID)}),
			TermValue: row.TermValue,
			Title:     firstNonEmpty(row.Lecture.Title, row.Lecture.ModuleTitle, "온라인 강의"),
			Course:    row.CourseName,
			DueAt:     row.Lecture.EndAt,
			Submitted: !lectureNeedsAttendance(row.Lecture, now),
			DetailURL: row.Lecture.PlayURL,
			Notes:     buildLectureReminderNotes(row),
		})
	}

	if len(assignments) == 0 {
		return ReminderSyncResult{}, nil
	}
	prepared, err := s.prepareReminderSync("lecture", studentID, assignments, opts.SyncDecisions)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return ReminderSyncResult{EligibleCount: len(assignments), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Assignments) == 0 {
		if err := s.saveSyncSourceState("lecture", studentID, prepared.State); err != nil {
			return ReminderSyncResult{}, err
		}
		return ReminderSyncResult{Result: reminder.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(assignments)}, nil
	}

	result, err := reminder.NewMacOSBridge(s.reminderBridgePath).Sync(reminder.SyncRequest{
		ListName:        currentSettings.Reminder.ListName,
		UseExistingList: currentSettings.Reminder.UseExistingList,
		AlarmBeforeMin:  currentSettings.Reminder.AlarmBeforeMin,
		Assignments:     prepared.Assignments,
	})
	if err != nil {
		return ReminderSyncResult{}, err
	}
	result.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, result.SyncedIDs)
	if err := s.saveSyncSourceState("lecture", studentID, prepared.State); err != nil {
		return ReminderSyncResult{}, err
	}
	return ReminderSyncResult{
		Result:        result,
		EligibleCount: len(assignments),
	}, nil
}

func (s *Service) SyncAcademicCalendar(ctx context.Context, opts AcademicListOptions) (CalendarSyncResult, error) {
	result, err := s.AcademicList(ctx, opts)
	if err != nil {
		return CalendarSyncResult{}, err
	}

	currentSettings, err := s.loadSettings()
	if err != nil {
		return CalendarSyncResult{}, err
	}

	events := make([]klapcalendar.Event, 0, len(result.Events))
	now := time.Now()
	for _, academicEvent := range result.Events {
		startAt, endAt, ok := academicEventRange(academicEvent)
		if !ok || startAt.Before(now) {
			continue
		}
		events = append(events, klapcalendar.Event{
			ID:      academicEventID(academicEvent),
			Title:   academicEvent.Title,
			StartAt: startAt,
			EndAt:   endAt,
			AllDay:  true,
			Notes:   buildAcademicCalendarNotes(academicEvent),
			URL:     result.SourceURL,
		})
	}
	if len(events) == 0 {
		return CalendarSyncResult{}, nil
	}
	prepared, err := s.prepareCalendarSync("academic", "global", events, opts.SyncDecisions)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return CalendarSyncResult{EligibleCount: len(events), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Events) == 0 {
		if err := s.saveSyncSourceState("academic", "global", prepared.State); err != nil {
			return CalendarSyncResult{}, err
		}
		return CalendarSyncResult{Result: klapcalendar.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(events)}, nil
	}

	syncResult, err := klapcalendar.NewMacOSBridge(s.calendarBridgePath).Sync(klapcalendar.SyncRequest{
		CalendarName:    currentSettings.Calendar.Name,
		UseExistingList: currentSettings.Calendar.UseExistingList,
		Events:          prepared.Events,
	})
	if err != nil {
		return CalendarSyncResult{}, err
	}
	syncResult.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, syncResult.SyncedIDs)
	if err := s.saveSyncSourceState("academic", "global", prepared.State); err != nil {
		return CalendarSyncResult{}, err
	}
	return CalendarSyncResult{Result: syncResult, EligibleCount: len(events)}, nil
}

func (s *Service) SyncTimetableCalendar(ctx context.Context, opts TimetableOptions) (CalendarSyncResult, error) {
	currentSettings, err := s.loadSettings()
	if err != nil {
		return CalendarSyncResult{}, err
	}
	result, err := s.Timetable(ctx, opts)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	startAt, endAt := timetableTermRange(result.Term.Value)
	if academic, academicErr := s.AcademicList(ctx, AcademicListOptions{Year: timetableTermYear(result.Term.Value), Refresh: opts.Refresh}); academicErr == nil {
		startAt, endAt = timetableTermRangeFromAcademic(result.Term.Value, academic.Events, startAt, endAt)
	}

	events := make([]klapcalendar.Event, 0, len(result.Entries))
	for _, entry := range result.Entries {
		if entry.Online || strings.TrimSpace(entry.Room) == "" {
			continue
		}
		startClock, endClock, ok := timetablePeriodRange(entry.Period, entry.Span)
		if !ok {
			continue
		}
		firstDay := firstWeekdayOnOrAfter(startAt, entry.Weekday)
		eventStart := time.Date(firstDay.Year(), firstDay.Month(), firstDay.Day(), startClock.Hour(), startClock.Minute(), 0, 0, time.Local)
		eventEnd := time.Date(firstDay.Year(), firstDay.Month(), firstDay.Day(), endClock.Hour(), endClock.Minute(), 0, 0, time.Local)
		if !eventEnd.After(eventStart) {
			continue
		}
		events = append(events, klapcalendar.Event{
			ID:            timetableCalendarEventID(result.Term.Value, entry),
			Title:         entry.SubjectName,
			StartAt:       eventStart,
			EndAt:         eventEnd,
			AllDay:        false,
			Notes:         buildTimetableCalendarNotes(result.Term.Value, entry),
			Recurrence:    "weekly",
			RecurrenceEnd: &endAt,
		})
	}
	if len(events) == 0 {
		return CalendarSyncResult{}, nil
	}
	prepared, err := s.prepareCalendarSync("timetable", studentID, events, opts.SyncDecisions)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return CalendarSyncResult{EligibleCount: len(events), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Events) == 0 {
		if err := s.saveSyncSourceState("timetable", studentID, prepared.State); err != nil {
			return CalendarSyncResult{}, err
		}
		return CalendarSyncResult{Result: klapcalendar.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(events)}, nil
	}

	syncResult, err := klapcalendar.NewMacOSBridge(s.calendarBridgePath).Sync(klapcalendar.SyncRequest{
		CalendarName:    currentSettings.Calendar.TimetableName,
		UseExistingList: currentSettings.Calendar.TimetableUseExistingList,
		Events:          prepared.Events,
	})
	if err != nil {
		return CalendarSyncResult{}, err
	}
	syncResult.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, syncResult.SyncedIDs)
	if err := s.saveSyncSourceState("timetable", studentID, prepared.State); err != nil {
		return CalendarSyncResult{}, err
	}
	return CalendarSyncResult{Result: syncResult, EligibleCount: len(events)}, nil
}

const syncSourceCacheTTL = 24 * 365 * 10 * time.Hour

type syncSourceState struct {
	Items map[string]syncSourceItem `json:"items"`
}

type syncSourceItem struct {
	Hash        string `json:"hash"`
	IgnoredHash string `json:"ignoredHash,omitempty"`
}

type preparedReminderSync struct {
	Assignments []reminder.Assignment
	State       syncSourceState
	Pending     map[string]string
	Skipped     int
	Conflicts   []SyncConflict
}

type preparedCalendarSync struct {
	Events    []klapcalendar.Event
	State     syncSourceState
	Pending   map[string]string
	Skipped   int
	Conflicts []SyncConflict
}

func (s *Service) prepareReminderSync(scope string, owner string, assignments []reminder.Assignment, decisions map[string]SyncDecision) (preparedReminderSync, error) {
	state, err := s.loadSyncSourceState(scope, owner)
	if err != nil {
		return preparedReminderSync{}, err
	}
	result := preparedReminderSync{
		State:   state,
		Pending: map[string]string{},
	}
	for _, assignment := range assignments {
		hash := reminderSourceHash(assignment)
		item, exists := state.Items[assignment.ID]
		if !exists {
			for _, legacyID := range assignment.LegacyIDs {
				legacyItem, legacyExists := state.Items[legacyID]
				if !legacyExists {
					continue
				}
				if legacyItem.Hash == legacyReminderSourceHash(assignment, legacyID) {
					legacyItem.Hash = hash
					legacyItem.IgnoredHash = ""
					delete(state.Items, legacyID)
				}
				state.Items[assignment.ID] = legacyItem
				item = legacyItem
				break
			}
		}
		assignment.KnownSourceHash = item.Hash
		key := syncConflictKey(scope, assignment.ID)
		if item.Hash != "" && item.Hash != hash {
			switch decisions[key] {
			case SyncDecisionApply:
				assignment.ForceUpdate = true
			case SyncDecisionKeep:
				item.IgnoredHash = hash
				state.Items[assignment.ID] = item
				result.Skipped++
				continue
			default:
				if item.IgnoredHash == hash {
					result.Skipped++
					continue
				}
				result.Conflicts = append(result.Conflicts, SyncConflict{
					Key:          key,
					Scope:        scope,
					ID:           assignment.ID,
					Kind:         "reminder",
					Title:        assignment.Title,
					PreviousHash: item.Hash,
					CurrentHash:  hash,
					Summary:      reminderConflictSummary(assignment),
				})
				continue
			}
		}
		result.Assignments = append(result.Assignments, assignment)
		result.Pending[assignment.ID] = hash
	}
	result.State = state
	return result, nil
}

func (s *Service) prepareCalendarSync(scope string, owner string, events []klapcalendar.Event, decisions map[string]SyncDecision) (preparedCalendarSync, error) {
	state, err := s.loadSyncSourceState(scope, owner)
	if err != nil {
		return preparedCalendarSync{}, err
	}
	result := preparedCalendarSync{
		State:   state,
		Pending: map[string]string{},
	}
	for _, event := range events {
		hash := calendarSourceHash(event)
		item := state.Items[event.ID]
		event.KnownSourceHash = item.Hash
		key := syncConflictKey(scope, event.ID)
		if item.Hash != "" && item.Hash != hash {
			switch decisions[key] {
			case SyncDecisionApply:
				event.ForceUpdate = true
			case SyncDecisionKeep:
				item.IgnoredHash = hash
				state.Items[event.ID] = item
				result.Skipped++
				continue
			default:
				if item.IgnoredHash == hash {
					result.Skipped++
					continue
				}
				result.Conflicts = append(result.Conflicts, SyncConflict{
					Key:          key,
					Scope:        scope,
					ID:           event.ID,
					Kind:         "calendar",
					Title:        event.Title,
					PreviousHash: item.Hash,
					CurrentHash:  hash,
					Summary:      calendarConflictSummary(event),
				})
				continue
			}
		}
		result.Events = append(result.Events, event)
		result.Pending[event.ID] = hash
	}
	result.State = state
	return result, nil
}

func (s *Service) loadSyncSourceState(scope string, owner string) (syncSourceState, error) {
	state := syncSourceState{Items: map[string]syncSourceItem{}}
	if s.cacheStore == nil {
		return state, nil
	}
	_, ok, err := s.cacheStore.Get(syncSourceCacheKey(scope, owner), &state)
	if err != nil {
		return syncSourceState{}, err
	}
	if !ok || state.Items == nil {
		state.Items = map[string]syncSourceItem{}
	}
	return state, nil
}

func (s *Service) saveSyncSourceState(scope string, owner string, state syncSourceState) error {
	if s.cacheStore == nil {
		return nil
	}
	if state.Items == nil {
		state.Items = map[string]syncSourceItem{}
	}
	return s.cacheStore.Set(syncSourceCacheKey(scope, owner), syncSourceCacheTTL, state)
}

func syncSourceCacheKey(scope string, owner string) string {
	return "sync-source:" + strings.TrimSpace(owner) + ":" + strings.TrimSpace(scope)
}

func syncConflictKey(scope string, id string) string {
	return strings.TrimSpace(scope) + ":" + strings.TrimSpace(id)
}

func commitSyncSourceHashes(state *syncSourceState, pending map[string]string, syncedIDs []string) {
	if state.Items == nil {
		state.Items = map[string]syncSourceItem{}
	}
	for _, id := range syncedIDs {
		if hash, ok := pending[id]; ok {
			state.Items[id] = syncSourceItem{Hash: hash}
		}
	}
}

func reminderSourceHash(assignment reminder.Assignment) string {
	dueAt := ""
	if assignment.DueAt != nil {
		dueAt = assignment.DueAt.UTC().Format(time.RFC3339Nano)
	}
	return hashSyncParts(
		assignment.ID,
		assignment.Title,
		assignment.Course,
		dueAt,
		strconv.FormatBool(assignment.Submitted),
		assignment.DetailURL,
		assignment.Notes,
	)
}

func legacyReminderSourceHash(assignment reminder.Assignment, legacyID string) string {
	assignment.ID = strings.TrimSpace(legacyID)
	assignment.Notes = replaceReminderNoteID(assignment.Notes, assignment.ID)
	return reminderSourceHash(assignment)
}

func replaceReminderNoteID(notes string, id string) string {
	lines := strings.Split(notes, "\n")
	metadataStart, metadataEnd, ok := reminderMetadataBounds(lines)
	if !ok {
		return notes
	}
	for index := metadataStart; index < metadataEnd; index++ {
		line := lines[index]
		if strings.HasPrefix(line, "ID: ") {
			lines[index] = "ID: " + strings.TrimSpace(id)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func reminderMetadataBounds(lines []string) (int, int, bool) {
	footer := -1
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.TrimSpace(lines[index]) == "[This reminder is created by KLAP.]" {
			footer = index
			break
		}
	}
	if footer < 0 {
		return 0, 0, false
	}
	for index := footer - 1; index >= 0; index-- {
		marker := strings.TrimSpace(lines[index])
		if marker == "--- KLAP ---" || marker == "=========================================" {
			return index + 1, footer, true
		}
	}
	return 0, 0, false
}

func calendarSourceHash(event klapcalendar.Event) string {
	recurrenceEnd := ""
	if event.RecurrenceEnd != nil {
		recurrenceEnd = event.RecurrenceEnd.UTC().Format(time.RFC3339Nano)
	}
	return hashSyncParts(
		event.ID,
		event.Title,
		event.StartAt.UTC().Format(time.RFC3339Nano),
		event.EndAt.UTC().Format(time.RFC3339Nano),
		strconv.FormatBool(event.AllDay),
		event.Notes,
		event.URL,
		event.Recurrence,
		recurrenceEnd,
	)
}

func hashSyncParts(parts ...string) string {
	hasher := sha256.New()
	for _, part := range parts {
		hasher.Write([]byte(part))
		hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func reminderConflictSummary(assignment reminder.Assignment) string {
	if assignment.DueAt == nil {
		return assignment.Title
	}
	return fmt.Sprintf("%s · %s", assignment.Title, assignment.DueAt.Format("2006-01-02 15:04"))
}

func calendarConflictSummary(event klapcalendar.Event) string {
	if event.AllDay {
		return fmt.Sprintf("%s · %s ~ %s", event.Title, event.StartAt.Format("2006-01-02"), event.EndAt.Format("2006-01-02"))
	}
	return fmt.Sprintf("%s · %s ~ %s", event.Title, event.StartAt.Format("2006-01-02 15:04"), event.EndAt.Format("15:04"))
}

func (s *Service) loadSettings() (settings.Settings, error) {
	if s.settingsStore == nil {
		return settings.Default(), nil
	}
	return s.settingsStore.Load()
}

func (s *Service) saveSettings(value settings.Settings) error {
	if s.settingsStore == nil {
		return errors.New("settings store가 초기화되지 않았습니다")
	}
	return s.settingsStore.Save(value)
}

func (s *Service) startCaffeinate(ctx context.Context) func() {
	if runtime.GOOS != "darwin" {
		return func() {}
	}
	current, err := s.loadSettings()
	if err != nil || !settings.DownloadCaffeinateEnabled(current.Download) {
		return func() {}
	}

	cmd := exec.CommandContext(ctx, "caffeinate", "-dims")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return func() {}
	}
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
}

func assignmentDetailURL(yearHakgi string, course klas.Course, ordSeq string) string {
	values := url.Values{}
	values.Set("selectYearhakgi", yearHakgi)
	values.Set("selectSubj", course.Value)
	values.Set("ordseq", strings.TrimSpace(ordSeq))
	return "https://klas.kw.ac.kr/std/lis/evltn/TaskViewStdPage.do?" + values.Encode()
}

func noticeDetailURL(yearHakgi string, course klas.Course, boardNo string, masterNo string) string {
	values := url.Values{}
	values.Set("selectYearhakgi", yearHakgi)
	values.Set("selectSubj", course.Value)
	values.Set("boardNo", strings.TrimSpace(boardNo))
	values.Set("masterNo", strings.TrimSpace(masterNo))
	return "https://klas.kw.ac.kr/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardViewStdPage.do?" + values.Encode()
}

func buildReminderNotes(result AssignmentDetailResult) string {
	detail := result.Detail
	var builder strings.Builder

	if strings.TrimSpace(detail.ContentText) != "" {
		builder.WriteString(strings.TrimSpace(detail.ContentText))
		builder.WriteString("\n\n")
	}

	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(result.ID)
	builder.WriteString("\n")
	builder.WriteString("과목: ")
	builder.WriteString(result.CourseName)
	builder.WriteString("\n")
	builder.WriteString("제목: ")
	builder.WriteString(detail.Title)
	builder.WriteString("\n")
	builder.WriteString("마감: ")
	if detail.DueAt == nil {
		builder.WriteString("마감 확인 필요")
	} else {
		builder.WriteString(detail.DueAt.Format("2006-01-02 15:04"))
	}
	builder.WriteString("\n")
	builder.WriteString("상태: ")
	if detail.Submitted {
		builder.WriteString("제출")
	} else {
		builder.WriteString("미제출")
	}
	builder.WriteString("\n")
	if detail.ReportType != "" {
		builder.WriteString("제출 방식: ")
		builder.WriteString(detail.ReportType)
		builder.WriteString("\n")
	}
	if detail.SubmitFileType != "" {
		builder.WriteString("파일 형식: ")
		builder.WriteString(detail.SubmitFileType)
		builder.WriteString("\n")
	}
	if detail.FileLimitMB != "" {
		builder.WriteString("파일 제한: ")
		builder.WriteString(detail.FileLimitMB)
		builder.WriteString("MB\n")
	}
	if hashtags := reminderHashtags(result.TermValue, result.CourseName); hashtags != "" {
		builder.WriteString(hashtags)
		builder.WriteString("\n")
	}
	builder.WriteString("[This reminder is created by KLAP.]")

	return builder.String()
}

func buildLectureReminderNotes(row LectureRow) string {
	var builder strings.Builder
	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(lectureReminderID(row.ID))
	builder.WriteString("\n")
	builder.WriteString("과목: ")
	builder.WriteString(row.CourseName)
	builder.WriteString("\n")
	builder.WriteString("제목: ")
	builder.WriteString(firstNonEmpty(row.Lecture.Title, row.Lecture.ModuleTitle, "온라인 강의"))
	builder.WriteString("\n")
	builder.WriteString("마감: ")
	if row.Lecture.EndAt == nil {
		builder.WriteString("마감 확인 필요")
	} else {
		builder.WriteString(row.Lecture.EndAt.Format("2006-01-02 15:04"))
	}
	builder.WriteString("\n")
	builder.WriteString("상태: ")
	builder.WriteString(lectureDueStatus(row.Lecture))
	builder.WriteString("\n")
	if row.Lecture.ModuleTitle != "" {
		builder.WriteString("주차/모듈: ")
		builder.WriteString(row.Lecture.ModuleTitle)
		builder.WriteString("\n")
	}
	if hashtags := reminderHashtags(row.TermValue, row.CourseName); hashtags != "" {
		builder.WriteString(hashtags)
		builder.WriteString("\n")
	}
	builder.WriteString("[This reminder is created by KLAP.]")
	return builder.String()
}

func buildAcademicCalendarNotes(event AcademicEvent) string {
	var builder strings.Builder
	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(academicEventID(event))
	builder.WriteString("\n")
	builder.WriteString("학년도: ")
	builder.WriteString(event.Year)
	builder.WriteString("\n")
	builder.WriteString("날짜: ")
	builder.WriteString(event.Month)
	builder.WriteString(" ")
	builder.WriteString(event.Date)
	builder.WriteString("\n")
	if strings.TrimSpace(event.Note) != "" {
		builder.WriteString("비고: ")
		builder.WriteString(event.Note)
		builder.WriteString("\n")
	}
	builder.WriteString("[This calendar event is created by KLAP.]")
	return builder.String()
}

func academicEventID(event AcademicEvent) string {
	parts := []string{event.Year, event.Title, event.Note}
	for index, part := range parts {
		parts[index] = strings.TrimSpace(part)
	}
	return "academic:" + strings.Join(compactNonEmpty(parts), ":")
}

func buildTimetableCalendarNotes(termValue string, entry klas.TimetableEntry) string {
	var builder strings.Builder
	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(timetableCalendarEventID(termValue, entry))
	builder.WriteString("\n")
	builder.WriteString("학기: ")
	builder.WriteString(termLabel(termValue))
	builder.WriteString("\n")
	builder.WriteString("과목: ")
	builder.WriteString(entry.SubjectName)
	builder.WriteString("\n")
	builder.WriteString("교시: ")
	builder.WriteString(fmt.Sprintf("%s %d교시", RoomWeekdayLabel(entry.Weekday), entry.Period))
	if entry.Span > 1 {
		builder.WriteString(fmt.Sprintf(" (%d교시 연강)", entry.Span))
	}
	builder.WriteString("\n")
	if strings.TrimSpace(entry.Room) != "" {
		builder.WriteString("강의실: ")
		builder.WriteString(entry.Room)
		builder.WriteString("\n")
	}
	if strings.TrimSpace(entry.Professor) != "" {
		builder.WriteString("교수: ")
		builder.WriteString(entry.Professor)
		builder.WriteString("\n")
	}
	builder.WriteString("[This calendar event is created by KLAP.]")
	return builder.String()
}

func timetableCalendarEventID(termValue string, entry klas.TimetableEntry) string {
	parts := []string{
		"timetable",
		strings.TrimSpace(termValue),
		strings.TrimSpace(entry.SubjectID),
		strconv.Itoa(entry.Weekday),
		strconv.Itoa(entry.Period),
		strconv.Itoa(entry.Span),
		strings.TrimSpace(entry.Room),
	}
	return strings.Join(compactNonEmpty(parts), ":")
}

func reminderHashtags(termValue string, courseName string) string {
	tags := make([]string, 0, 2)
	if tag := termHashtag(termValue); tag != "" {
		tags = append(tags, tag)
	}
	if tag := courseHashtag(courseName); tag != "" {
		tags = append(tags, tag)
	}
	return strings.Join(tags, " ")
}

func termHashtag(termValue string) string {
	termValue = strings.TrimSpace(termValue)
	if termValue == "" {
		return ""
	}
	parts := strings.FieldsFunc(termValue, func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(parts) >= 2 {
		year := strings.TrimSpace(parts[0])
		switch strings.TrimSpace(parts[1]) {
		case "3":
			return "#" + year + "-여름학기"
		case "4":
			return "#" + year + "-겨울학기"
		}
	}
	return "#" + strings.ReplaceAll(termValue, ",", "-")
}

func courseHashtag(courseName string) string {
	courseName = strings.TrimSpace(courseName)
	if courseName == "" {
		return ""
	}
	return "#" + strings.Join(strings.Fields(courseName), "")
}

func (s *Service) latestTerm(ctx context.Context, studentID string) (*klas.Client, klas.Term, error) {
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, klas.Term{}, err
	}
	term, client, err := s.selectedTerm(ctx, studentID, client)
	return client, term, err
}

func (s *Service) termForSyllabus(ctx context.Context, studentID string, client *klas.Client, termValue string) (klas.Term, *klas.Client, error) {
	termValue, err := normalizeTermValue(termValue)
	if err != nil {
		return klas.Term{}, client, err
	}
	if termValue == "" {
		return s.selectedTerm(ctx, studentID, client)
	}

	terms, client, err := s.courses(ctx, studentID, client)
	if err != nil {
		return klas.Term{}, client, err
	}
	for _, term := range terms {
		if term.Value == termValue {
			return term, client, nil
		}
	}
	return klas.Term{}, client, fmt.Errorf("학기를 찾을 수 없습니다: %s", termValue)
}

func (s *Service) termForSubjectSearch(termValue string) (klas.Term, error) {
	termValue, err := normalizeTermValue(termValue)
	if err != nil {
		return klas.Term{}, err
	}
	if termValue == "" {
		current, err := s.loadSettings()
		if err != nil {
			return klas.Term{}, err
		}
		termValue = strings.TrimSpace(current.Term.Value)
	}
	if termValue == "" {
		termValue = currentAcademicTermValue(time.Now())
	}
	return klas.Term{
		Value: termValue,
		Label: termLabel(termValue),
	}, nil
}

func (s *Service) selectedTerm(ctx context.Context, studentID string, client *klas.Client) (klas.Term, *klas.Client, error) {
	terms, client, err := s.courses(ctx, studentID, client)
	if err != nil {
		return klas.Term{}, client, err
	}
	if len(terms) == 0 {
		return klas.Term{}, client, errors.New("수강 학기가 없습니다")
	}

	current, err := s.loadSettings()
	if err != nil {
		return klas.Term{}, client, err
	}
	if strings.TrimSpace(current.Term.Value) == "" {
		return terms[0], client, nil
	}
	for _, term := range terms {
		if term.Value == current.Term.Value {
			return term, client, nil
		}
	}
	return klas.Term{}, client, fmt.Errorf("선택된 학기를 현재 유저에서 찾을 수 없습니다: %s", current.Term.Value)
}

func (s *Service) courses(ctx context.Context, studentID string, client *klas.Client) ([]klas.Term, *klas.Client, error) {
	terms, err := client.Courses(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return nil, client, refreshErr
		}
		if refreshed {
			client = refreshedClient
			terms, err = client.Courses(ctx)
		}
	}
	if err != nil {
		return nil, client, err
	}
	return terms, client, nil
}

func normalizeTermValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(parts) != 2 {
		return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
	}
	year := strings.TrimSpace(parts[0])
	semester := strings.TrimSpace(parts[1])
	if len(year) != 4 {
		return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
	}
	for _, r := range year + semester {
		if r < '0' || r > '9' {
			return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
		}
	}
	if semester < "1" || semester > "4" {
		return "", errors.New("학기는 1, 2, 3, 4 중 하나여야 합니다")
	}
	return year + "," + semester, nil
}

func currentAcademicTermValue(now time.Time) string {
	year := now.Year()
	switch now.Month() {
	case time.January, time.February:
		return fmt.Sprintf("%d,4", year-1)
	case time.March, time.April, time.May, time.June:
		return fmt.Sprintf("%d,1", year)
	case time.July, time.August:
		return fmt.Sprintf("%d,3", year)
	case time.December:
		return fmt.Sprintf("%d,4", year)
	default:
		return fmt.Sprintf("%d,2", year)
	}
}

func termLabel(termValue string) string {
	parts := strings.Split(strings.TrimSpace(termValue), ",")
	if len(parts) != 2 {
		return strings.TrimSpace(termValue)
	}
	switch parts[1] {
	case "1":
		return fmt.Sprintf("%s년도 1학기", parts[0])
	case "2":
		return fmt.Sprintf("%s년도 2학기", parts[0])
	case "3":
		return fmt.Sprintf("%s년도 여름학기", parts[0])
	case "4":
		return fmt.Sprintf("%s년도 겨울학기", parts[0])
	default:
		return strings.TrimSpace(termValue)
	}
}

func looksLikeSyllabusCourseCode(value string) bool {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}
	return true
}

func makeEvaluationRows(courses []klas.EvaluationCourse) []EvaluationRow {
	rows := make([]EvaluationRow, 0, len(courses))
	for index, course := range courses {
		rows = append(rows, EvaluationRow{
			Index:  index + 1,
			Course: course,
		})
	}
	return rows
}

func selectEvaluationRows(rows []EvaluationRow, selector string) ([]EvaluationRow, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, errors.New("수업평가 제출 대상이 없습니다")
	}
	if strings.EqualFold(selector, "all") {
		return rows, nil
	}
	if number, err := strconv.Atoi(selector); err == nil {
		if number < 1 || number > len(rows) {
			return nil, fmt.Errorf("수업평가 과목 번호가 범위를 벗어났습니다: %d", number)
		}
		return []EvaluationRow{rows[number-1]}, nil
	}

	normalizedSelector := strings.ToLower(selector)
	var matches []EvaluationRow
	for _, row := range rows {
		name := strings.ToLower(strings.TrimSpace(row.Course.Name))
		professor := strings.ToLower(strings.TrimSpace(row.Course.Professor))
		if name == normalizedSelector {
			return []EvaluationRow{row}, nil
		}
		if strings.Contains(name, normalizedSelector) || strings.Contains(professor, normalizedSelector) {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("수업평가 과목이 여러 개와 일치합니다. evaluation list 번호를 사용하세요: %s", selector)
	}
	return nil, fmt.Errorf("수업평가 과목을 찾을 수 없습니다: %s", selector)
}

func dashboardAssignments(rows []AssignmentRow, limit int) []AssignmentRow {
	filtered := make([]AssignmentRow, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		if row.Assignment.Submitted {
			continue
		}
		if row.Assignment.DueAt != nil && row.Assignment.DueAt.Before(now) {
			continue
		}
		filtered = append(filtered, row)
	}
	return limitAssignments(filtered, limit)
}

func dashboardLectures(rows []LectureRow, now time.Time, limit int) []LectureRow {
	filtered := make([]LectureRow, 0, len(rows))
	for _, row := range rows {
		if !lectureNeedsAttendance(row.Lecture, now) {
			continue
		}
		filtered = append(filtered, row)
	}
	return limitLectures(filtered, limit)
}

func dashboardNotices(rows []NoticeRow, limit int) []NoticeRow {
	sorted := append([]NoticeRow(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i].Notice.Registered
		right := sorted[j].Notice.Registered
		if left == nil && right == nil {
			return sorted[i].ID < sorted[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return sorted[i].ID < sorted[j].ID
		}
		return right.Before(*left)
	})
	return limitNotices(sorted, limit)
}

func dashboardAttendance(rows []AttendanceRow) DashboardAttendance {
	summary := DashboardAttendance{TotalCourses: len(rows)}
	for _, row := range rows {
		item := DashboardAttendanceRow{
			Index:  row.Index,
			Course: row.Course,
			Err:    row.Err,
		}
		if row.Err != nil {
			summary.DetailErrors++
			summary.Rows = append(summary.Rows, item)
			continue
		}
		for _, session := range row.Sessions {
			for _, slot := range session.Slots {
				switch strings.ToUpper(strings.TrimSpace(slot.Mark)) {
				case "O":
					item.Completed++
				case "X":
					item.Absent++
				case "L":
					item.Late++
				case "R":
					item.LeaveEarly++
				case "A":
					item.Excused++
				default:
					if strings.TrimSpace(slot.Mark+slot.Status) != "" {
						item.Unknown++
					}
				}
			}
		}
		summary.Completed += item.Completed
		summary.Absent += item.Absent
		summary.Late += item.Late
		summary.LeaveEarly += item.LeaveEarly
		summary.Excused += item.Excused
		summary.Unknown += item.Unknown
		summary.Rows = append(summary.Rows, item)
	}
	return summary
}

func dashboardEvaluation(result EvaluationListResult) DashboardEvaluation {
	evaluation := DashboardEvaluation{
		Term:    result.Term,
		Enabled: result.Term.TermEnabled,
	}
	for _, row := range result.Rows {
		if row.Course.Evaluated {
			evaluation.Done++
			continue
		}
		evaluation.Pending++
		evaluation.Rows = append(evaluation.Rows, row)
	}
	return evaluation
}

func dashboardCourses(courses []klas.Course, assignments []AssignmentRow, lectures []LectureRow, notices []NoticeRow, attendance DashboardAttendance, evaluation DashboardEvaluation) []DashboardCourse {
	result := make([]DashboardCourse, 0, len(courses))
	now := time.Now()
	for index, course := range courses {
		name := strings.TrimSpace(course.Name)
		if name == "" {
			name = "과목 확인 필요"
		}
		item := DashboardCourse{
			Index:       index + 1,
			Name:        name,
			Assignments: dashboardAssignments(filterRowsByCourse(assignments, name, func(row AssignmentRow) string { return row.CourseName }), 5),
			Lectures:    dashboardLectures(filterRowsByCourse(lectures, name, func(row LectureRow) string { return row.CourseName }), now, 5),
			Notices:     dashboardNotices(filterRowsByCourse(notices, name, func(row NoticeRow) string { return row.CourseName }), 5),
		}
		if row, ok := dashboardAttendanceForCourse(attendance.Rows, name); ok {
			item.Attendance = &row
		}
		if row, ok := dashboardEvaluationForCourse(evaluation.Rows, name); ok {
			item.Evaluation = &row
		}
		result = append(result, item)
	}
	return result
}

func filterRowsByCourse[T any](rows []T, courseName string, course func(T) string) []T {
	filtered := make([]T, 0)
	for _, row := range rows {
		if strings.TrimSpace(course(row)) == courseName {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func dashboardAttendanceForCourse(rows []DashboardAttendanceRow, courseName string) (DashboardAttendanceRow, bool) {
	for _, row := range rows {
		if strings.TrimSpace(row.Course.Name) == courseName {
			return row, true
		}
	}
	return DashboardAttendanceRow{}, false
}

func dashboardEvaluationForCourse(rows []EvaluationRow, courseName string) (EvaluationRow, bool) {
	for _, row := range rows {
		if strings.TrimSpace(row.Course.Name) == courseName {
			return row, true
		}
	}
	return EvaluationRow{}, false
}

func dashboardCacheKey(studentID string, termValue string) string {
	return "dashboard:v2:" + strings.TrimSpace(studentID) + ":" + strings.TrimSpace(termValue)
}

func dashboardCacheTTL() time.Duration {
	return 5 * time.Minute
}

func dashboardCacheable(result DashboardResult) bool {
	return len(result.SectionErrors) == 0 && result.Attendance.DetailErrors == 0
}

func normalizeCachedDashboardResult(result DashboardResult, term klas.Term) (DashboardResult, bool, error) {
	migrated := false
	assignments, changed, err := normalizeCachedAssignmentRows(result.Assignments, term)
	if err != nil {
		return DashboardResult{}, false, err
	}
	result.Assignments = assignments
	migrated = migrated || changed
	notices, changed, err := normalizeCachedNoticeRows(result.Notices, term)
	if err != nil {
		return DashboardResult{}, false, err
	}
	result.Notices = notices
	migrated = migrated || changed
	lectures, changed, err := normalizeCachedLectureRows(result.Lectures, term)
	if err != nil {
		return DashboardResult{}, false, err
	}
	result.Lectures = lectures
	migrated = migrated || changed
	for index := range result.Courses {
		assignments, changed, err = normalizeCachedAssignmentRows(result.Courses[index].Assignments, term)
		if err != nil {
			return DashboardResult{}, false, err
		}
		result.Courses[index].Assignments = assignments
		migrated = migrated || changed
		notices, changed, err = normalizeCachedNoticeRows(result.Courses[index].Notices, term)
		if err != nil {
			return DashboardResult{}, false, err
		}
		result.Courses[index].Notices = notices
		migrated = migrated || changed
		lectures, changed, err = normalizeCachedLectureRows(result.Courses[index].Lectures, term)
		if err != nil {
			return DashboardResult{}, false, err
		}
		result.Courses[index].Lectures = lectures
		migrated = migrated || changed
	}
	return result, migrated, nil
}

func listCacheKey(scope string, studentID string, termValue string, selector string) string {
	return listCacheKeyVersion(scope, "v1", studentID, termValue, selector)
}

func listCacheKeyVersion(scope string, version string, studentID string, termValue string, selector string) string {
	parts := []string{
		strings.TrimSpace(scope),
		strings.TrimSpace(version),
		strings.TrimSpace(studentID),
		strings.TrimSpace(termValue),
		strings.TrimSpace(selector),
	}
	return strings.Join(parts, ":")
}

func courseResourceListCacheKey(scope string, studentID string, termValue string, courses []selectedCourse) string {
	return courseResourceListCacheKeyVersion(scope, "v1", studentID, termValue, courses)
}

func courseResourceListCacheKeyVersion(scope string, version string, studentID string, termValue string, courses []selectedCourse) string {
	courseIDs := make([]string, 0, len(courses))
	for _, course := range courses {
		courseIDs = append(courseIDs, strings.TrimSpace(course.Course.Value))
	}
	sort.Strings(courseIDs)
	parts := []string{
		strings.TrimSpace(scope),
		strings.TrimSpace(version),
		strings.TrimSpace(studentID),
		strings.TrimSpace(termValue),
		"courses-" + hashSyncParts(courseIDs...)[:24],
	}
	return strings.Join(parts, ":")
}

func assignmentRowIDs(rows []AssignmentRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func noticeRowIDs(rows []NoticeRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func lectureRowIDs(rows []LectureRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func resourceIDsMatchSelected(ids []string, kind string, remotePartCount int, courses []selectedCourse, allowEmpty bool) bool {
	if len(ids) == 0 {
		return allowEmpty
	}
	selected := make(map[string]struct{}, len(courses))
	for _, course := range courses {
		selected[strings.TrimSpace(course.Course.Value)] = struct{}{}
	}
	for _, id := range ids {
		ref, _, stable, err := parseStableCourseResourceID(kind, id, remotePartCount)
		if err != nil || !stable {
			return false
		}
		if _, ok := selected[ref.CourseID]; !ok {
			return false
		}
	}
	return true
}

func listCacheTTL() time.Duration {
	return 5 * time.Minute
}

func attendanceCacheable(result AttendanceListResult) bool {
	for _, row := range result.Rows {
		if row.Err != nil {
			return false
		}
	}
	return true
}

func cacheScopePrefix(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "all":
		return ""
	case "dashboard":
		return "dashboard:"
	case "course", "courses":
		return "course:"
	case "assignment", "assignments":
		return "assignment:"
	case "notice", "notices":
		return "notice:"
	case "lecture", "lectures":
		return "lecture:"
	case "timetable":
		return "timetable:"
	case "attendance":
		return "attendance:"
	case "academic":
		return "academic:"
	case "grade", "grades":
		return "grade:"
	case "rank", "ranks":
		return "rank:"
	case "evaluation", "evaluations":
		return "evaluation:"
	default:
		return strings.TrimSpace(scope) + ":"
	}
}

func validSearchType(value string) bool {
	switch value {
	case "course", "assignment", "notice", "lecture", "academic":
		return true
	default:
		return false
	}
}

func textMatches(query string, values ...string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return false
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), query) {
			return true
		}
	}
	return false
}

func inDueWindow(value time.Time, from time.Time, until time.Time) bool {
	return !value.Before(from) && !value.After(until)
}

func futureTime(value *time.Time, now time.Time) bool {
	return value != nil && !value.Before(now)
}

func lectureDueStatus(lecture klas.Lecture) string {
	if lectureIsLearningActivity(lecture) {
		return formatProgressValue(lecture.AchievedTime, lecture.RequiredTime)
	}
	return emptyStatusFallback(lecture.Progress, "진도 확인 필요")
}

func formatProgressValue(achieved string, required string) string {
	return emptyStatusFallback(achieved, "0") + "/" + emptyStatusFallback(required, "?") + "분"
}

func emptyStatusFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

var academicDatePattern = regexp.MustCompile(`(?:(\d{1,2})\s*[./]\s*)?(\d{1,2})\s*(?:일|\([^)]*\))?`)

func academicEventDueAt(event AcademicEvent) (time.Time, bool) {
	startAt, _, ok := academicEventRange(event)
	return startAt, ok
}

func AcademicEventRange(event AcademicEvent) (time.Time, time.Time, bool) {
	return academicEventRange(event)
}

func academicEventRange(event AcademicEvent) (time.Time, time.Time, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(event.Year))
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	monthText := strings.TrimSuffix(strings.TrimSpace(event.Month), "월")
	month, err := strconv.Atoi(monthText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	dateText := strings.TrimSpace(event.Date)
	matches := academicDatePattern.FindAllStringSubmatch(dateText, -1)
	if len(matches) == 0 {
		return time.Time{}, time.Time{}, false
	}
	dates := make([]time.Time, 0, len(matches))
	for _, match := range matches {
		eventMonth := month
		if strings.TrimSpace(match[1]) != "" {
			parsedMonth, monthErr := strconv.Atoi(match[1])
			if monthErr != nil {
				return time.Time{}, time.Time{}, false
			}
			eventMonth = parsedMonth
		}
		day, dayErr := strconv.Atoi(match[2])
		if dayErr != nil {
			return time.Time{}, time.Time{}, false
		}
		dates = append(dates, time.Date(year, time.Month(eventMonth), day, 0, 0, 0, 0, time.Local))
	}
	startAt := dates[0]
	endAt := dates[len(dates)-1]
	if endAt.Before(startAt) {
		endAt = endAt.AddDate(1, 0, 0)
	}
	return startAt, endAt.AddDate(0, 0, 1), true
}

type dayClock struct {
	hour   int
	minute int
}

func (c dayClock) Hour() int {
	return c.hour
}

func (c dayClock) Minute() int {
	return c.minute
}

var timetableSinglePeriodTimes = map[int][2]dayClock{
	0:  {dayClock{8, 0}, dayClock{10, 45}},
	1:  {dayClock{9, 0}, dayClock{10, 15}},
	2:  {dayClock{10, 30}, dayClock{11, 45}},
	3:  {dayClock{12, 0}, dayClock{13, 15}},
	4:  {dayClock{13, 30}, dayClock{14, 45}},
	5:  {dayClock{15, 0}, dayClock{16, 15}},
	6:  {dayClock{16, 30}, dayClock{17, 45}},
	7:  {dayClock{18, 0}, dayClock{18, 45}},
	8:  {dayClock{18, 50}, dayClock{19, 35}},
	9:  {dayClock{19, 40}, dayClock{20, 25}},
	10: {dayClock{20, 30}, dayClock{21, 15}},
	11: {dayClock{21, 20}, dayClock{22, 5}},
}

var timetableConsecutiveTimes = map[int]map[int][2]dayClock{
	2: {
		0: {dayClock{8, 0}, dayClock{9, 50}},
		1: {dayClock{9, 0}, dayClock{10, 50}},
		3: {dayClock{12, 0}, dayClock{13, 50}},
		5: {dayClock{15, 0}, dayClock{16, 50}},
	},
	3: {
		0: {dayClock{8, 0}, dayClock{10, 45}},
		6: {dayClock{16, 30}, dayClock{19, 15}},
	},
	4: {
		0: {dayClock{8, 0}, dayClock{11, 50}},
		5: {dayClock{15, 0}, dayClock{18, 50}},
	},
}

func timetablePeriodRange(period int, span int) (dayClock, dayClock, bool) {
	if span <= 1 {
		times, ok := timetableSinglePeriodTimes[period]
		return times[0], times[1], ok
	}
	if byPeriod, ok := timetableConsecutiveTimes[span]; ok {
		if times, ok := byPeriod[period]; ok {
			return times[0], times[1], true
		}
	}
	start, ok := timetableSinglePeriodTimes[period]
	if !ok {
		return dayClock{}, dayClock{}, false
	}
	end, ok := timetableSinglePeriodTimes[period+span-1]
	if !ok {
		return dayClock{}, dayClock{}, false
	}
	return start[0], end[1], true
}

func timetableTermYear(termValue string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(termValue), func(r rune) bool { return r == ',' || r == '-' })
	if len(parts) == 0 {
		return time.Now().Format("2006")
	}
	return parts[0]
}

func timetableTermRange(termValue string) (time.Time, time.Time) {
	year, _ := strconv.Atoi(timetableTermYear(termValue))
	if year <= 0 {
		year = time.Now().Year()
	}
	semester := "1"
	parts := strings.FieldsFunc(strings.TrimSpace(termValue), func(r rune) bool { return r == ',' || r == '-' })
	if len(parts) > 1 {
		semester = strings.TrimSpace(parts[1])
	}
	switch semester {
	case "2":
		return time.Date(year, time.September, 1, 0, 0, 0, 0, time.Local), time.Date(year, time.December, 31, 23, 59, 59, 0, time.Local)
	case "3":
		return time.Date(year, time.June, 22, 0, 0, 0, 0, time.Local), time.Date(year, time.August, 31, 23, 59, 59, 0, time.Local)
	case "4":
		return time.Date(year, time.December, 22, 0, 0, 0, 0, time.Local), time.Date(year+1, time.February, 28, 23, 59, 59, 0, time.Local)
	default:
		return time.Date(year, time.March, 1, 0, 0, 0, 0, time.Local), time.Date(year, time.June, 30, 23, 59, 59, 0, time.Local)
	}
}

func timetableTermRangeFromAcademic(termValue string, events []AcademicEvent, fallbackStart time.Time, fallbackEnd time.Time) (time.Time, time.Time) {
	start := fallbackStart
	end := fallbackEnd
	startMonth, endMonth := timetableTermMonths(termValue)
	for _, event := range events {
		eventStart, eventEnd, ok := academicEventRange(event)
		if !ok {
			continue
		}
		title := strings.TrimSpace(event.Title)
		if strings.Contains(title, "개강") && int(eventStart.Month()) == startMonth {
			start = eventStart
		}
		if strings.Contains(title, "종강") && int(eventStart.Month()) == endMonth {
			end = eventEnd.Add(-time.Second)
		}
	}
	if end.Before(start) {
		return fallbackStart, fallbackEnd
	}
	return start, end
}

func timetableTermMonths(termValue string) (int, int) {
	parts := strings.FieldsFunc(strings.TrimSpace(termValue), func(r rune) bool { return r == ',' || r == '-' })
	if len(parts) > 1 {
		switch strings.TrimSpace(parts[1]) {
		case "2":
			return 9, 12
		case "3":
			return 6, 8
		case "4":
			return 12, 2
		}
	}
	return 3, 6
}

func firstWeekdayOnOrAfter(startAt time.Time, weekday int) time.Time {
	target := time.Weekday(weekday % 7)
	for day := startAt; ; day = day.AddDate(0, 0, 1) {
		if day.Weekday() == target {
			return day
		}
	}
}

func parseConfigBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "t", "1", "yes", "y", "on":
		return true, nil
	case "false", "f", "0", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("boolean 값이 필요합니다: %s", value)
	}
}

func limitAssignments(rows []AssignmentRow, limit int) []AssignmentRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func limitLectures(rows []LectureRow, limit int) []LectureRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func limitNotices(rows []NoticeRow, limit int) []NoticeRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func compactNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (s *Service) selectedStudentID(ctx context.Context, user UserOption) (string, error) {
	if strings.TrimSpace(user.StudentID) != "" {
		return strings.TrimSpace(user.StudentID), nil
	}

	currentStudentID, err := s.store.Current(ctx)
	if err != nil {
		return "", err
	}
	if currentStudentID != "" {
		return currentStudentID, nil
	}

	users, err := s.store.List(ctx)
	if err != nil {
		return "", err
	}
	switch len(users) {
	case 0:
		return "", errors.New("저장된 유저가 없습니다. 먼저 klap auth를 실행하세요")
	case 1:
		return users[0].StudentID, nil
	default:
		return "", errors.New("저장된 유저가 여러 명입니다. klap user select <학번> 또는 --user <학번>을 지정하세요")
	}
}

func (s *Service) authenticatedClient(ctx context.Context, studentID string) (*klas.Client, error) {
	client, err := s.newKlasClient()
	if err != nil {
		return nil, err
	}

	session, err := s.sessions.LoadSession(ctx, studentID)
	if err == nil {
		client.SetSession(session)
		return client, nil
	}

	password, err := s.sessions.LoadPassword(ctx, studentID)
	if err != nil {
		return nil, err
	}
	session, err = s.login(ctx, client, studentID, password)
	if err != nil {
		return nil, fmt.Errorf("재로그인 실패: %w", err)
	}
	if err := s.sessions.SaveSession(ctx, studentID, session); err != nil {
		return nil, fmt.Errorf("갱신 세션 저장 실패: %w", err)
	}
	return client, nil
}

func (s *Service) refreshedClientAfterSessionError(ctx context.Context, studentID string, err error) (*klas.Client, bool, error) {
	if !errors.Is(err, klas.ErrSessionExpired) {
		return nil, false, err
	}

	client, clientErr := s.newKlasClient()
	if clientErr != nil {
		return nil, true, clientErr
	}
	password, passwordErr := s.sessions.LoadPassword(ctx, studentID)
	if passwordErr != nil {
		return nil, true, passwordErr
	}
	session, loginErr := s.login(ctx, client, studentID, password)
	if loginErr != nil {
		return nil, true, fmt.Errorf("재로그인 실패: %w", loginErr)
	}
	if saveErr := s.sessions.SaveSession(ctx, studentID, session); saveErr != nil {
		return nil, true, fmt.Errorf("갱신 세션 저장 실패: %w", saveErr)
	}
	return client, true, nil
}

func selectedCourses(term klas.Term, filter string) ([]selectedCourse, error) {
	if strings.TrimSpace(filter) == "" {
		selected := make([]selectedCourse, 0, len(term.Courses))
		for index, course := range term.Courses {
			selected = append(selected, selectedCourse{Index: index + 1, Course: course})
		}
		return selected, nil
	}

	if number, err := strconv.Atoi(filter); err == nil {
		if number < 1 || number > len(term.Courses) {
			return nil, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", number)
		}
		return []selectedCourse{{Index: number, Course: term.Courses[number-1]}}, nil
	}

	normalizedFilter := strings.ToLower(strings.TrimSpace(filter))
	var containsMatches []selectedCourse
	for index, course := range term.Courses {
		normalizedName := strings.ToLower(strings.TrimSpace(course.Name))
		if normalizedName == normalizedFilter {
			return []selectedCourse{{Index: index + 1, Course: course}}, nil
		}
		if strings.Contains(normalizedName, normalizedFilter) {
			containsMatches = append(containsMatches, selectedCourse{Index: index + 1, Course: course})
		}
	}
	if len(containsMatches) == 1 {
		return containsMatches, nil
	}
	if len(containsMatches) > 1 {
		return nil, fmt.Errorf("과목명이 여러 개와 일치합니다. course list 번호를 사용하세요: %s", filter)
	}
	return nil, fmt.Errorf("과목을 찾을 수 없습니다: %s", filter)
}

func selectAttendanceRows(rows []AttendanceRow, selector string) []AttendanceRow {
	selector = strings.TrimSpace(selector)
	if number, err := strconv.Atoi(selector); err == nil {
		if number >= 1 && number <= len(rows) {
			return []AttendanceRow{rows[number-1]}
		}
		return nil
	}

	matches := make([]AttendanceRow, 0)
	normalizedSelector := strings.ToLower(selector)
	for _, row := range rows {
		course := row.Course
		if strings.EqualFold(course.CourseCode, selector) ||
			strings.Contains(strings.ToLower(course.Name), normalizedSelector) ||
			strings.Contains(strings.ToLower(course.Professor), normalizedSelector) {
			matches = append(matches, row)
		}
	}
	return matches
}

func AssignmentID(courseIndex int, ordSeq string) string {
	return fmt.Sprintf("%d:%s", courseIndex, strings.TrimSpace(ordSeq))
}

func StableAssignmentID(ref CourseRef, ordSeq string) (string, error) {
	return stableCourseResourceID("assignment", ref, ordSeq)
}

type courseResourceLocator struct {
	Ref         CourseRef
	CourseIndex int
	Stable      bool
}

func parseAssignmentResourceID(id string) (courseResourceLocator, string, error) {
	ref, remoteParts, stable, err := parseStableCourseResourceID("assignment", id, 1)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	if stable {
		return courseResourceLocator{Ref: ref, Stable: true}, remoteParts[0], nil
	}
	courseIndex, ordSeq, err := ParseAssignmentID(id)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	return courseResourceLocator{CourseIndex: courseIndex}, ordSeq, nil
}

func resolveResourceCourse(term klas.Term, locator courseResourceLocator) (klas.Course, error) {
	if locator.Stable {
		if strings.TrimSpace(term.Value) != strings.TrimSpace(locator.Ref.TermValue) {
			return klas.Course{}, fmt.Errorf("stable ID 학기와 조회 학기가 다릅니다: %s != %s", locator.Ref.TermValue, term.Value)
		}
		for _, course := range term.Courses {
			if strings.TrimSpace(course.Value) == locator.Ref.CourseID {
				return course, nil
			}
		}
		return klas.Course{}, fmt.Errorf("학기 %s에서 과목을 찾을 수 없습니다: %s", term.Value, locator.Ref.CourseID)
	}
	if locator.CourseIndex < 1 || locator.CourseIndex > len(term.Courses) {
		return klas.Course{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", locator.CourseIndex)
	}
	return term.Courses[locator.CourseIndex-1], nil
}

func normalizeCachedAssignmentRows(rows []AssignmentRow, term klas.Term) ([]AssignmentRow, bool, error) {
	migrated := false
	for index := range rows {
		locator, ordSeq, err := parseAssignmentResourceID(rows[index].ID)
		if err != nil {
			return nil, false, err
		}
		if locator.Stable {
			course, err := resolveResourceCourse(term, locator)
			if err != nil {
				return nil, false, err
			}
			for courseIndex, candidate := range term.Courses {
				if strings.TrimSpace(candidate.Value) == strings.TrimSpace(course.Value) {
					rows[index].LegacyID = AssignmentID(courseIndex+1, ordSeq)
					break
				}
			}
			continue
		}
		course, err := resolveLegacyCachedCourse(term, locator.CourseIndex, rows[index].CourseName)
		if err != nil {
			return nil, false, err
		}
		ref, err := NewCourseRef(term.Value, course)
		if err != nil {
			return nil, false, err
		}
		stableID, err := StableAssignmentID(ref, ordSeq)
		if err != nil {
			return nil, false, err
		}
		rows[index].LegacyID = rows[index].ID
		rows[index].ID = stableID
		rows[index].TermValue = term.Value
		migrated = true
	}
	return rows, migrated, nil
}

func resolveLegacyCachedCourse(term klas.Term, courseIndex int, courseName string) (klas.Course, error) {
	normalizedName := strings.TrimSpace(courseName)
	if normalizedName == "" {
		return klas.Course{}, fmt.Errorf("legacy cache 과목명 없이 순번을 안전하게 이전할 수 없습니다: %d", courseIndex)
	}
	var match *klas.Course
	for index := range term.Courses {
		if !strings.EqualFold(strings.TrimSpace(term.Courses[index].Name), normalizedName) {
			continue
		}
		if match != nil {
			return klas.Course{}, fmt.Errorf("legacy cache 과목명이 여러 과목과 일치합니다: %s", courseName)
		}
		match = &term.Courses[index]
	}
	if match != nil {
		return *match, nil
	}
	return klas.Course{}, fmt.Errorf("legacy cache 과목을 찾을 수 없습니다: %s", courseName)
}

func ParseAssignmentID(id string) (int, string, error) {
	parts := strings.SplitN(strings.TrimSpace(id), ":", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("과제ID는 course list 번호와 ordseq를 조합한 <과목번호>:<ordseq> 형식이어야 합니다")
	}

	courseIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("과목 번호 파싱 실패: %w", err)
	}
	ordSeq := strings.TrimSpace(parts[1])
	if ordSeq == "" {
		return 0, "", errors.New("과제ID에 ordseq가 없습니다")
	}
	return courseIndex, ordSeq, nil
}

func NoticeID(courseIndex int, boardNo string, masterNo string) string {
	return fmt.Sprintf("%d:%s:%s", courseIndex, strings.TrimSpace(boardNo), strings.TrimSpace(masterNo))
}

func StableNoticeID(ref CourseRef, boardNo string, masterNo string) (string, error) {
	return stableCourseResourceID("notice", ref, boardNo, masterNo)
}

func parseNoticeResourceID(id string) (courseResourceLocator, string, string, error) {
	ref, remoteParts, stable, err := parseStableCourseResourceID("notice", id, 2)
	if err != nil {
		return courseResourceLocator{}, "", "", err
	}
	if stable {
		return courseResourceLocator{Ref: ref, Stable: true}, remoteParts[0], remoteParts[1], nil
	}
	courseIndex, boardNo, masterNo, err := ParseNoticeID(id)
	if err != nil {
		return courseResourceLocator{}, "", "", err
	}
	return courseResourceLocator{CourseIndex: courseIndex}, boardNo, masterNo, nil
}

func normalizeCachedNoticeRows(rows []NoticeRow, term klas.Term) ([]NoticeRow, bool, error) {
	migrated := false
	for index := range rows {
		locator, boardNo, masterNo, err := parseNoticeResourceID(rows[index].ID)
		if err != nil {
			return nil, false, err
		}
		if locator.Stable {
			if _, err := resolveResourceCourse(term, locator); err != nil {
				return nil, false, err
			}
			continue
		}
		course, err := resolveLegacyCachedCourse(term, locator.CourseIndex, rows[index].CourseName)
		if err != nil {
			return nil, false, err
		}
		ref, err := NewCourseRef(term.Value, course)
		if err != nil {
			return nil, false, err
		}
		stableID, err := StableNoticeID(ref, boardNo, masterNo)
		if err != nil {
			return nil, false, err
		}
		rows[index].ID = stableID
		rows[index].TermValue = term.Value
		migrated = true
	}
	return rows, migrated, nil
}

func ParseNoticeID(id string) (int, string, string, error) {
	parts := strings.Split(strings.TrimSpace(id), ":")
	if len(parts) != 3 {
		return 0, "", "", errors.New("공지ID는 course list 번호, boardNo, masterNo를 조합한 <과목번호>:<boardNo>:<masterNo> 형식이어야 합니다")
	}

	courseIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("과목 번호 파싱 실패: %w", err)
	}
	boardNo := strings.TrimSpace(parts[1])
	if boardNo == "" {
		return 0, "", "", errors.New("공지ID에 boardNo가 없습니다")
	}
	masterNo := strings.TrimSpace(parts[2])
	if masterNo == "" {
		return 0, "", "", errors.New("공지ID에 masterNo가 없습니다")
	}
	return courseIndex, boardNo, masterNo, nil
}

func LectureID(courseIndex int, lectureKey string) string {
	lectureKey = strings.TrimSpace(lectureKey)
	if lectureKey == "" {
		return "-"
	}
	return fmt.Sprintf("%d:%s", courseIndex, lectureKey)
}

func StableLectureID(ref CourseRef, lectureKey string) (string, error) {
	return stableCourseResourceID("lecture", ref, lectureKey)
}

func parseLectureResourceID(id string) (courseResourceLocator, string, error) {
	ref, remoteParts, stable, err := parseStableCourseResourceID("lecture", id, 1)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	if stable {
		return courseResourceLocator{Ref: ref, Stable: true}, remoteParts[0], nil
	}
	courseIndex, lectureKey, err := ParseLectureID(id)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	return courseResourceLocator{CourseIndex: courseIndex}, lectureKey, nil
}

func stableLectureTermValue(ids []string) (string, error) {
	termValue := ""
	for _, id := range ids {
		ref, _, stable, err := parseStableCourseResourceID("lecture", id, 1)
		if err != nil {
			return "", err
		}
		if !stable {
			continue
		}
		if termValue != "" && termValue != ref.TermValue {
			return "", errors.New("서로 다른 학기의 강의를 한 번에 처리할 수 없습니다")
		}
		termValue = ref.TermValue
	}
	return termValue, nil
}

type resolvedLectureResource struct {
	Client   *klas.Client
	Term     klas.Term
	Course   klas.Course
	Key      string
	ID       string
	LegacyID string
}

func (s *Service) resolveLectureResource(ctx context.Context, studentID string, id string) (resolvedLectureResource, error) {
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	locator, lectureKey, err := parseLectureResourceID(id)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	var term klas.Term
	if locator.Stable {
		term, client, err = s.termForSyllabus(ctx, studentID, client, locator.Ref.TermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return resolvedLectureResource{}, err
	}
	course, err := resolveResourceCourse(term, locator)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	ref, err := NewCourseRef(term.Value, course)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	stableID, err := StableLectureID(ref, lectureKey)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	courseIndex, err := courseIndexByID(term, course.Value)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	return resolvedLectureResource{
		Client:   client,
		Term:     term,
		Course:   course,
		Key:      lectureKey,
		ID:       stableID,
		LegacyID: LectureID(courseIndex, lectureKey),
	}, nil
}

func newLectureRow(termValue string, selected selectedCourse, lecture klas.Lecture) (LectureRow, error) {
	ref, err := NewCourseRef(termValue, selected.Course)
	if err != nil {
		return LectureRow{}, err
	}
	lectureKey := lectureResourceKey(lecture)
	id, err := StableLectureID(ref, lectureKey)
	if err != nil {
		return LectureRow{}, err
	}
	legacyID := lectureRowID(selected.Index, lecture)
	if legacyID == "-" {
		legacyID = ""
	}
	return LectureRow{
		ID:         id,
		LegacyID:   legacyID,
		TermValue:  strings.TrimSpace(termValue),
		CourseName: selected.Course.Name,
		Lecture:    lecture,
	}, nil
}

func normalizeCachedLectureRows(rows []LectureRow, term klas.Term) ([]LectureRow, bool, error) {
	migrated := false
	for index := range rows {
		locator, lectureKey, parseErr := parseLectureResourceID(rows[index].ID)
		legacyID := ""
		if parseErr != nil && strings.TrimSpace(rows[index].ID) != "-" {
			return nil, false, parseErr
		}
		var course klas.Course
		var err error
		if parseErr == nil && locator.Stable {
			course, err = resolveResourceCourse(term, locator)
			if err != nil {
				return nil, false, err
			}
		} else {
			course, err = resolveLegacyCachedCourse(term, locator.CourseIndex, rows[index].CourseName)
			if err != nil {
				return nil, false, err
			}
			legacyID = strings.TrimSpace(rows[index].ID)
			if legacyID == "-" {
				legacyID = ""
			}
			lectureKey = lectureResourceKey(rows[index].Lecture)
			ref, err := NewCourseRef(term.Value, course)
			if err != nil {
				return nil, false, err
			}
			rows[index].ID, err = StableLectureID(ref, lectureKey)
			if err != nil {
				return nil, false, err
			}
			rows[index].TermValue = term.Value
			migrated = true
		}
		if legacyID == "" {
			courseIndex, err := courseIndexByID(term, course.Value)
			if err != nil {
				return nil, false, err
			}
			legacyID = LectureID(courseIndex, lectureKey)
		}
		rows[index].LegacyID = legacyID
	}
	return rows, migrated, nil
}

func courseIndexByID(term klas.Term, courseID string) (int, error) {
	for index, course := range term.Courses {
		if strings.TrimSpace(course.Value) == strings.TrimSpace(courseID) {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("학기 %s에서 과목을 찾을 수 없습니다: %s", term.Value, courseID)
}

func lectureReminderID(rowID string) string {
	rowID = strings.TrimSpace(rowID)
	if rowID == "" {
		return ""
	}
	if strings.HasPrefix(rowID, "lecture:") {
		return rowID
	}
	return "lecture:" + rowID
}

func ParseLectureID(id string) (int, string, error) {
	parts := strings.SplitN(strings.TrimSpace(id), ":", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("강의ID는 course list 번호와 강의 키를 조합한 <과목번호>:<contentID|lrn-lrnSn> 형식이어야 합니다")
	}

	courseIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("과목 번호 파싱 실패: %w", err)
	}
	lectureKey := strings.TrimSpace(parts[1])
	if lectureKey == "" {
		return 0, "", errors.New("강의ID에 강의 키가 없습니다")
	}
	return courseIndex, lectureKey, nil
}

func defaultLectureDownloadDir() string {
	return settings.DefaultDownloadDir()
}

func (s *Service) effectiveDownloadDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir != "" {
		return dir, nil
	}
	current, err := s.loadSettings()
	if err != nil {
		return "", err
	}
	return current.Download.Dir, nil
}

func (s *Service) effectiveDownloadConcurrency(concurrency int) (int, error) {
	if concurrency > 0 {
		return concurrency, nil
	}
	current, err := s.loadSettings()
	if err != nil {
		return 0, err
	}
	if current.Download.Concurrency <= 0 {
		return settings.DefaultDownloadConcurrency, nil
	}
	return current.Download.Concurrency, nil
}

var lectureWeekPattern = regexp.MustCompile(`([0-9]{1,2})\s*주차`)

func lectureVideoPath(root string, courseName string, lecture klas.Lecture, mediaURL string, weekOrder int) string {
	return filepath.Join(root, lectureCourseDirName(courseName), "video", lectureFilename(lecture, mediaURL, weekOrder))
}

func lectureCourseDirName(courseName string) string {
	courseDir := sanitizePathComponent(courseName)
	if courseDir == "" {
		return "course"
	}
	return courseDir
}

func lectureFilename(lecture klas.Lecture, mediaURL string, weekOrder int) string {
	extension := ".mp4"
	if parsed, err := url.Parse(mediaURL); err == nil {
		if ext := filepath.Ext(parsed.Path); ext != "" {
			extension = ext
		}
	}

	if week := lectureWeekNumber(lecture); week > 0 && weekOrder > 0 {
		title := sanitizePathComponent(firstNonEmpty(lecture.Title, lecture.ModuleTitle, lecture.ContentID, lecture.LearningSeq, "lecture"))
		return fmt.Sprintf("%d-%d. %s%s", week, weekOrder, title, extension)
	}

	parts := []string{
		sanitizePathComponent(lecture.ModuleTitle),
		sanitizePathComponent(lecture.Title),
	}

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		filtered = append(filtered, "lecture")
	}
	return strings.Join(filtered, "_") + extension
}

func lectureWeekNumber(lecture klas.Lecture) int {
	if week := parsePositiveInt(lecture.WeekNo); week > 0 {
		return week
	}
	match := lectureWeekPattern.FindStringSubmatch(lecture.ModuleTitle)
	if len(match) >= 2 {
		return parsePositiveInt(match[1])
	}
	return 0
}

func lectureWeekOrder(lectures []klas.Lecture, target klas.Lecture) int {
	if order := parsePositiveInt(target.WeeklySeq); order > 0 {
		return order
	}

	week := lectureWeekNumber(target)
	if week <= 0 {
		return 0
	}

	order := 0
	for _, lecture := range lectures {
		if lectureWeekNumber(lecture) != week {
			continue
		}
		order++
		if sameLectureForFilename(lecture, target) {
			return order
		}
	}
	return 0
}

func lectureRowWeekOrders(rows []LectureRow) map[string]int {
	counts := make(map[int]int)
	orders := make(map[string]int, len(rows))
	for _, row := range rows {
		if order := parsePositiveInt(row.Lecture.WeeklySeq); order > 0 {
			orders[row.ID] = order
			week := lectureWeekNumber(row.Lecture)
			if week > 0 && counts[week] < order {
				counts[week] = order
			}
			continue
		}

		week := lectureWeekNumber(row.Lecture)
		if week <= 0 {
			continue
		}
		counts[week]++
		orders[row.ID] = counts[week]
	}
	return orders
}

func sameLectureForFilename(left klas.Lecture, right klas.Lecture) bool {
	if strings.TrimSpace(left.ContentID) != "" && strings.TrimSpace(left.ContentID) == strings.TrimSpace(right.ContentID) {
		return true
	}
	if strings.TrimSpace(left.LearningSeq) != "" && strings.TrimSpace(left.LearningSeq) == strings.TrimSpace(right.LearningSeq) {
		return true
	}
	return strings.TrimSpace(left.ModuleTitle) == strings.TrimSpace(right.ModuleTitle) &&
		strings.TrimSpace(left.Title) == strings.TrimSpace(right.Title)
}

func parsePositiveInt(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return 0
	}
	return number
}

func sanitizePathComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	value = replacer.Replace(value)
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 120 {
		runes := []rune(value)
		value = string(runes[:120])
	}
	return strings.Trim(value, ". ")
}

func emitLectureDownloadProgress(onProgress func(LectureDownloadProgress), progress LectureDownloadProgress) {
	if onProgress != nil {
		onProgress(progress)
	}
}

func downloadFile(ctx context.Context, sourceURL string, path string, keepPartial bool, onProgress func(bytesWritten int64, totalBytes int64)) (int64, error) {
	if _, err := os.Stat(path); err == nil {
		return 0, fmt.Errorf("이미 파일이 있습니다: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}

	partPath := path + ".part"
	var offset int64
	if stat, err := os.Stat(partPath); err == nil {
		offset = stat.Size()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("동영상 다운로드 요청 실패: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	flag := os.O_CREATE | os.O_WRONLY
	if offset > 0 && response.StatusCode == http.StatusPartialContent {
		flag |= os.O_APPEND
	} else {
		offset = 0
		flag |= os.O_TRUNC
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, fmt.Errorf("동영상 다운로드 HTTP 오류: %s", response.Status)
	}
	totalBytes := response.ContentLength
	if totalBytes > 0 {
		totalBytes += offset
	}
	if onProgress != nil {
		onProgress(offset, totalBytes)
	}

	file, err := os.OpenFile(partPath, flag, 0o644)
	if err != nil {
		return 0, fmt.Errorf("임시 파일 열기 실패: %w", err)
	}
	written, copyErr := copyWithProgress(file, response.Body, offset, totalBytes, onProgress)
	closeErr := file.Close()
	if copyErr != nil {
		cleanupPartialDownload(partPath, keepPartial)
		return offset + written, fmt.Errorf("동영상 다운로드 실패: %w", copyErr)
	}
	if closeErr != nil {
		cleanupPartialDownload(partPath, keepPartial)
		return offset + written, fmt.Errorf("임시 파일 닫기 실패: %w", closeErr)
	}

	if err := os.Rename(partPath, path); err != nil {
		return offset + written, fmt.Errorf("다운로드 파일 저장 실패: %w", err)
	}
	return offset + written, nil
}

func cleanupPartialDownload(partPath string, keepPartial bool) {
	if keepPartial {
		return
	}
	_ = os.Remove(partPath)
}

func copyWithProgress(dst io.Writer, src io.Reader, offset int64, totalBytes int64, onProgress func(bytesWritten int64, totalBytes int64)) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64
	for {
		nr, readErr := src.Read(buffer)
		if nr > 0 {
			nw, writeErr := dst.Write(buffer[:nr])
			if nw > 0 {
				written += int64(nw)
				if onProgress != nil {
					onProgress(offset+written, totalBytes)
				}
			}
			if writeErr != nil {
				return written, writeErr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			return written, readErr
		}
	}
}

func attendLecture(ctx context.Context, client *klas.Client, row LectureRow, interval time.Duration, onProgress func(LectureRow, klas.LectureProgress)) (klas.LectureProgress, error) {
	if lectureIsLearningActivity(row.Lecture) {
		status := lectureLearningStatus(row.Lecture, time.Now())
		if status != "Y" {
			return klas.LectureProgress{}, errors.New("학습기간이 아니어서 학습활동 시간이 반영되지 않습니다")
		}
		progress, err := client.SaveLectureLearningStatus(ctx, row.Lecture, status)
		if err != nil {
			return progress, err
		}
		if onProgress != nil {
			onProgress(row, progress)
		}
		return progress, nil
	}
	return attendLectureLoop(ctx, client, row, interval, onProgress)
}

func attendLectureLoop(ctx context.Context, client *klas.Client, row LectureRow, interval time.Duration, onProgress func(LectureRow, klas.LectureProgress)) (klas.LectureProgress, error) {
	if interval <= 0 {
		interval = 60 * time.Second
	}

	lecKey, err := client.LectureKey(ctx, row.Lecture)
	if err != nil {
		return klas.LectureProgress{}, err
	}

	var lastProgress klas.LectureProgress
	for {
		if err := client.CheckLectureView(ctx, row.Lecture, lecKey); err != nil {
			return lastProgress, err
		}
		progress, err := client.UpdateLectureProgress(ctx, row.Lecture, lecKey)
		if err != nil {
			return lastProgress, err
		}
		lastProgress = progress
		if onProgress != nil {
			onProgress(row, progress)
		}
		if progress.Completed {
			return progress, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastProgress, ctx.Err()
		case <-timer.C:
		}
	}
}

func lectureNeedsAttendance(lecture klas.Lecture, now time.Time) bool {
	if lectureIsLearningActivity(lecture) {
		if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
			return false
		}
		if lecture.EndAt != nil && now.After(*lecture.EndAt) {
			return false
		}
		progress, progressErr := strconv.ParseFloat(strings.TrimSpace(lecture.AchievedTime), 64)
		required, requiredErr := strconv.ParseFloat(strings.TrimSpace(lecture.RequiredTime), 64)
		if progressErr == nil && requiredErr == nil && required > 0 && progress >= required {
			return false
		}
		return strings.TrimSpace(lecture.LearningSeq) != ""
	}

	progress, err := strconv.ParseFloat(strings.TrimSpace(lecture.Progress), 64)
	if err == nil && progress >= 100 {
		return false
	}
	if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
		return false
	}
	if lecture.EndAt != nil && now.After(*lecture.EndAt) {
		return false
	}
	return strings.TrimSpace(lecture.ContentID) != ""
}

func lectureIsLearningActivity(lecture klas.Lecture) bool {
	return strings.TrimSpace(lecture.ContentID) == "" && strings.TrimSpace(lecture.LearningSeq) != ""
}

func lectureLearningStatus(lecture klas.Lecture, now time.Time) string {
	if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
		return "N"
	}
	if lecture.EndAt != nil && now.After(*lecture.EndAt) {
		return "N"
	}
	return "Y"
}

func lectureAttendKey(lecture klas.Lecture) string {
	if contentID := strings.TrimSpace(lecture.ContentID); contentID != "" {
		return contentID
	}
	if learningSeq := strings.TrimSpace(lecture.LearningSeq); learningSeq != "" {
		return "lrn-" + learningSeq
	}
	return ""
}

func lectureResourceKey(lecture klas.Lecture) string {
	if key := lectureAttendKey(lecture); key != "" {
		return key
	}
	if fileID := strings.TrimSpace(lecture.FileID); fileID != "" {
		return "file-" + fileID
	}
	if weekNo := strings.TrimSpace(lecture.WeekNo); weekNo != "" || strings.TrimSpace(lecture.WeeklySeq) != "" {
		return "week-" + weekNo + "-" + strings.TrimSpace(lecture.WeeklySeq)
	}
	startAt := ""
	if lecture.StartAt != nil {
		startAt = lecture.StartAt.UTC().Format(time.RFC3339Nano)
	}
	endAt := ""
	if lecture.EndAt != nil {
		endAt = lecture.EndAt.UTC().Format(time.RFC3339Nano)
	}
	return "meta-" + hashSyncParts(lecture.ModuleTitle, lecture.Title, startAt, endAt)[:24]
}

func lectureRowID(courseIndex int, lecture klas.Lecture) string {
	return LectureID(courseIndex, lectureAttendKey(lecture))
}

func lectureMatchesKey(lecture klas.Lecture, key string) bool {
	key = strings.TrimSpace(key)
	return key != "" && lectureResourceKey(lecture) == key
}

func defaultReminderBridgePath() string {
	if override := os.Getenv("KLAP_REMINDER_BRIDGE"); override != "" {
		return override
	}

	candidates := []string{
		filepath.Join("bridges", "macos", "reminder.swift"),
	}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, filepath.Join(repoRoot, "bridges", "macos", "reminder.swift"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "bridges", "macos", "reminder.swift"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func defaultCalendarBridgePath() string {
	if override := os.Getenv("KLAP_CALENDAR_BRIDGE"); override != "" {
		return override
	}

	candidates := []string{
		filepath.Join("bridges", "macos", "calendar.swift"),
	}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, filepath.Join(repoRoot, "bridges", "macos", "calendar.swift"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "bridges", "macos", "calendar.swift"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func defaultCategoryBridgePath() string {
	if override := os.Getenv("KLAP_CATEGORY_BRIDGE"); override != "" {
		return override
	}

	candidates := []string{
		filepath.Join("bridges", "macos", "categories.swift"),
	}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, filepath.Join(repoRoot, "bridges", "macos", "categories.swift"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "bridges", "macos", "categories.swift"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func defaultTranscriptBridgePath() string {
	if override := os.Getenv("KLAP_TRANSCRIPT_BRIDGE"); override != "" {
		return override
	}

	candidates := transcriptBridgeCandidates(filepath.Join("bridges", "macos"))
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, transcriptBridgeCandidates(filepath.Join(repoRoot, "bridges", "macos"))...)
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "TranscriptBridge"))
		candidates = append(candidates, transcriptBridgeCandidates(filepath.Join(filepath.Dir(executable), "bridges", "macos"))...)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func transcriptBridgeCandidates(root string) []string {
	return []string{
		filepath.Join(root, ".build", "release", "TranscriptBridge"),
		filepath.Join(root, ".build", "debug", "TranscriptBridge"),
		filepath.Join(root, "transcribe.swift"),
	}
}
