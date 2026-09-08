package app

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/category"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	store                *account.Store
	sessions             sessionStore
	settingsStore        *settings.Store
	cacheStore           *cache.Store
	syncStateStore       syncStateStore
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

type TermListOptions struct {
	User UserOption
}

type UserRow struct {
	User    account.User
	Current bool
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

type TermRow struct {
	Index   int
	Term    klas.Term
	Current bool
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
	settings  func() (*settings.Store, error)
	cache     func() (*cache.Store, error)
	syncState func() (*syncstate.Store, error)
}

func NewService(store *account.Store) (*Service, error) {
	return newService(store, serviceStoreFactories{
		settings:  settings.NewStore,
		cache:     cache.NewStore,
		syncState: syncstate.NewStore,
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
	syncStateStore, err := factories.syncState()
	if err != nil {
		return nil, fmt.Errorf("sync state store 초기화 실패: %w", err)
	}
	return &Service{
		store:          store,
		sessions:       store,
		settingsStore:  settingsStore,
		cacheStore:     cacheStore,
		syncStateStore: syncStateStore,
		newKlasClient:  klas.NewClient,
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

func (s *Service) latestTerm(ctx context.Context, studentID string) (*klas.Client, klas.Term, error) {
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, klas.Term{}, err
	}
	term, client, err := s.selectedTerm(ctx, studentID, client)
	return client, term, err
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
	return dashboardCacheKeyVersion("v3", studentID, termValue)
}

func dashboardCacheKeyVersion(version string, studentID string, termValue string) string {
	return "dashboard:" + strings.TrimSpace(version) + ":" + strings.TrimSpace(studentID) + ":" + strings.TrimSpace(termValue)
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

type courseResourceLocator struct {
	Ref         CourseRef
	CourseIndex int
	Stable      bool
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

func courseIndexByID(term klas.Term, courseID string) (int, error) {
	for index, course := range term.Courses {
		if strings.TrimSpace(course.Value) == strings.TrimSpace(courseID) {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("학기 %s에서 과목을 찾을 수 없습니다: %s", term.Value, courseID)
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
		candidates = append(candidates, filepath.Join(executableDirectory(executable), "bridges", "macos", "reminder.swift"))
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
		candidates = append(candidates, filepath.Join(executableDirectory(executable), "bridges", "macos", "calendar.swift"))
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
		candidates = append(candidates, filepath.Join(executableDirectory(executable), "bridges", "macos", "categories.swift"))
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
		executableDir := executableDirectory(executable)
		candidates = append(candidates, filepath.Join(executableDir, "TranscriptBridge"))
		candidates = append(candidates, transcriptBridgeCandidates(filepath.Join(executableDir, "bridges", "macos"))...)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func executableDirectory(executable string) string {
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return filepath.Dir(executable)
}

func transcriptBridgeCandidates(root string) []string {
	return []string{
		filepath.Join(root, ".build", "release", "TranscriptBridge"),
		filepath.Join(root, ".build", "debug", "TranscriptBridge"),
		filepath.Join(root, "transcribe.swift"),
	}
}
