package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kw-klap/klap-cli/internal/account"
	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/reminder"
	"github.com/kw-klap/klap-cli/internal/settings"
)

type Service struct {
	store              *account.Store
	settingsStore      *settings.Store
	reminderBridgePath string
}

type UserOption struct {
	StudentID string
}

type AssignmentListOptions struct {
	User         UserOption
	CourseFilter string
}

type NoticeListOptions struct {
	User         UserOption
	CourseFilter string
}

type TimetableOptions struct {
	User UserOption
}

type AttendanceListOptions struct {
	User UserOption
}

type AttendanceDetailOptions struct {
	User     UserOption
	Selector string
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
	User         UserOption
	CourseFilter string
}

type LectureDownloadOptions struct {
	User UserOption
	Dir  string
}

type LectureDownloadAllOptions struct {
	User         UserOption
	CourseFilter string
	Dir          string
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
}

type ReminderSettings struct {
	ListName        string
	UseExistingList bool
	AlarmBeforeMin  int
}

type TermSettings struct {
	Value string
	Label string
}

type selectedCourse struct {
	Index  int
	Course klas.Course
}

func NewService(store *account.Store) *Service {
	settingsStore, _ := settings.NewStore()
	return &Service{
		store:              store,
		settingsStore:      settingsStore,
		reminderBridgePath: defaultReminderBridgePath(),
	}
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

func (s *Service) CourseList(ctx context.Context, user UserOption) ([]klas.Term, error) {
	studentID, err := s.selectedStudentID(ctx, user)
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
	return []klas.Term{term}, nil
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
			id := AssignmentID(selectedCourse.Index, assignment.OrdSeq)
			rows = append(rows, AssignmentRow{
				ID:         id,
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

	return rows, nil
}

func (s *Service) AssignmentDetail(ctx context.Context, id string, user UserOption) (AssignmentDetailResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return AssignmentDetailResult{}, err
	}

	courseIndex, ordSeq, err := ParseAssignmentID(id)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	if courseIndex < 1 || courseIndex > len(term.Courses) {
		return AssignmentDetailResult{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", courseIndex)
	}

	course := term.Courses[courseIndex-1]
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
		ID:         id,
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
			id := NoticeID(selectedCourse.Index, notice.BoardNo, notice.MasterNo)
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

	return rows, nil
}

func (s *Service) NoticeDetail(ctx context.Context, id string, user UserOption) (NoticeDetailResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return NoticeDetailResult{}, err
	}

	courseIndex, boardNo, masterNo, err := ParseNoticeID(id)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	if courseIndex < 1 || courseIndex > len(term.Courses) {
		return NoticeDetailResult{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", courseIndex)
	}

	course := term.Courses[courseIndex-1]
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
		ID:         id,
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

	return TimetableResult{
		Term:    term,
		Entries: entries,
	}, nil
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
	return AttendanceListResult{
		Term: term,
		Rows: rows,
	}, nil
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
			rows = append(rows, LectureRow{
				ID:         LectureID(selectedCourse.Index, lectureAttendKey(lecture)),
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				Lecture:    lecture,
			})
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

	return rows, nil
}

func (s *Service) LectureOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return OpenURLResult{}, err
	}

	courseIndex, lectureKey, err := ParseLectureID(id)
	if err != nil {
		return OpenURLResult{}, err
	}
	if courseIndex < 1 || courseIndex > len(term.Courses) {
		return OpenURLResult{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", courseIndex)
	}

	course := term.Courses[courseIndex-1]
	lectures, err := client.Lectures(ctx, term.Value, course)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return OpenURLResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			lectures, err = client.Lectures(ctx, term.Value, course)
		}
	}
	if err != nil {
		return OpenURLResult{}, err
	}

	for _, lecture := range lectures {
		if !lectureMatchesKey(lecture, lectureKey) {
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
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return LectureDownloadResult{}, err
	}

	courseIndex, contentID, err := ParseLectureID(id)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	if courseIndex < 1 || courseIndex > len(term.Courses) {
		return LectureDownloadResult{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", courseIndex)
	}

	course := term.Courses[courseIndex-1]
	lectures, err := client.Lectures(ctx, term.Value, course)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return LectureDownloadResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			lectures, err = client.Lectures(ctx, term.Value, course)
		}
	}
	if err != nil {
		return LectureDownloadResult{}, err
	}

	var matched *klas.Lecture
	for index := range lectures {
		if strings.TrimSpace(lectures[index].ContentID) == contentID {
			matched = &lectures[index]
			break
		}
	}
	if matched == nil {
		return LectureDownloadResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
	}

	mediaURL, err := client.ResolveLectureMediaURL(ctx, contentID)
	if err != nil {
		return LectureDownloadResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	if dir == "" {
		dir = defaultLectureDownloadDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return LectureDownloadResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}

	path := filepath.Join(dir, lectureFilename(course.Name, *matched, mediaURL))
	bytesWritten, err := downloadFile(ctx, mediaURL, path)
	if err != nil {
		return LectureDownloadResult{}, err
	}

	row := LectureRow{
		ID:         id,
		TermValue:  term.Value,
		CourseName: course.Name,
		Lecture:    *matched,
	}
	return LectureDownloadResult{
		Path:     path,
		Bytes:    bytesWritten,
		Lecture:  row,
		MediaURL: mediaURL,
	}, nil
}

func (s *Service) DownloadAllLectures(ctx context.Context, opts LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
	if strings.TrimSpace(opts.CourseFilter) == "" {
		return LectureDownloadAllResult{}, errors.New("전체 다운로드에는 과목명 또는 course list 번호가 필요합니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	if dir == "" {
		dir = defaultLectureDownloadDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return LectureDownloadAllResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}

	result := LectureDownloadAllResult{}
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

		for _, lecture := range lectures {
			row := LectureRow{
				ID:         LectureID(selectedCourse.Index, lecture.ContentID),
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				Lecture:    lecture,
			}
			item := LectureDownloadItem{Lecture: row}
			if strings.TrimSpace(lecture.ContentID) == "" {
				item.Skipped = true
				item.Err = errors.New("KWCommons 콘텐츠 ID가 없습니다")
				result.Items = append(result.Items, item)
				continue
			}

			mediaURL, err := client.ResolveLectureMediaURL(ctx, lecture.ContentID)
			if err != nil {
				item.Err = err
				result.Items = append(result.Items, item)
				continue
			}

			item.Path = filepath.Join(dir, lectureFilename(selectedCourse.Course.Name, lecture, mediaURL))
			if _, err := os.Stat(item.Path); err == nil {
				item.Skipped = true
				result.Items = append(result.Items, item)
				continue
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				item.Err = err
				result.Items = append(result.Items, item)
				continue
			}

			bytesWritten, err := downloadFile(ctx, mediaURL, item.Path)
			item.Bytes = bytesWritten
			if err != nil {
				item.Err = err
			}
			result.Items = append(result.Items, item)
		}
	}

	return result, nil
}

func (s *Service) AttendLecture(ctx context.Context, id string, opts LectureAttendOptions) (LectureAttendResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureAttendResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return LectureAttendResult{}, err
	}

	courseIndex, lectureKey, err := ParseLectureID(id)
	if err != nil {
		return LectureAttendResult{}, err
	}
	if courseIndex < 1 || courseIndex > len(term.Courses) {
		return LectureAttendResult{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", courseIndex)
	}

	course := term.Courses[courseIndex-1]
	lectures, err := client.Lectures(ctx, term.Value, course)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return LectureAttendResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			lectures, err = client.Lectures(ctx, term.Value, course)
		}
	}
	if err != nil {
		return LectureAttendResult{}, err
	}

	var matched *klas.Lecture
	for index := range lectures {
		if lectureMatchesKey(lectures[index], lectureKey) {
			matched = &lectures[index]
			break
		}
	}
	if matched == nil {
		return LectureAttendResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
	}

	row := LectureRow{
		ID:         id,
		TermValue:  term.Value,
		CourseName: course.Name,
		Lecture:    *matched,
	}
	progress, err := attendLecture(ctx, client, row, opts.Interval, opts.OnProgress)
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
			row := LectureRow{
				ID:         LectureID(selectedCourse.Index, lectureAttendKey(lecture)),
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				Lecture:    lecture,
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

	currentSettings, err := s.loadSettings()
	if err != nil {
		return ReminderSyncResult{}, err
	}

	assignments := make([]reminder.Assignment, 0, len(rows))
	for _, row := range rows {
		if row.Assignment.DueAt == nil {
			continue
		}
		detail, detailErr := s.AssignmentDetail(ctx, row.ID, opts.User)
		if detailErr != nil {
			return ReminderSyncResult{}, detailErr
		}
		assignments = append(assignments, reminder.Assignment{
			ID:        row.ID,
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

	result, err := reminder.NewMacOSBridge(s.reminderBridgePath).Sync(reminder.SyncRequest{
		ListName:        currentSettings.Reminder.ListName,
		UseExistingList: currentSettings.Reminder.UseExistingList,
		AlarmBeforeMin:  currentSettings.Reminder.AlarmBeforeMin,
		Assignments:     assignments,
	})
	if err != nil {
		return ReminderSyncResult{}, err
	}
	return ReminderSyncResult{
		Result:        result,
		EligibleCount: len(assignments),
	}, nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
	client, err := klas.NewClient()
	if err != nil {
		return nil, err
	}

	session, err := s.store.LoadSession(ctx, studentID)
	if err == nil {
		client.SetSession(session)
		return client, nil
	}

	password, err := s.store.LoadPassword(ctx, studentID)
	if err != nil {
		return nil, err
	}
	session, err = client.Login(ctx, studentID, password)
	if err != nil {
		return nil, fmt.Errorf("재로그인 실패: %w", err)
	}
	_ = s.store.SaveSession(ctx, studentID, session)
	return client, nil
}

func (s *Service) refreshedClientAfterSessionError(ctx context.Context, studentID string, err error) (*klas.Client, bool, error) {
	if !errors.Is(err, klas.ErrSessionExpired) {
		return nil, false, err
	}

	client, clientErr := klas.NewClient()
	if clientErr != nil {
		return nil, true, clientErr
	}
	password, passwordErr := s.store.LoadPassword(ctx, studentID)
	if passwordErr != nil {
		return nil, true, passwordErr
	}
	session, loginErr := client.Login(ctx, studentID, password)
	if loginErr != nil {
		return nil, true, fmt.Errorf("재로그인 실패: %w", loginErr)
	}
	_ = s.store.SaveSession(ctx, studentID, session)
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
	return "downloads"
}

func lectureFilename(courseName string, lecture klas.Lecture, mediaURL string) string {
	parts := []string{
		sanitizePathComponent(courseName),
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

	extension := ".mp4"
	if parsed, err := url.Parse(mediaURL); err == nil {
		if ext := filepath.Ext(parsed.Path); ext != "" {
			extension = ext
		}
	}
	return strings.Join(filtered, "_") + extension
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

func downloadFile(ctx context.Context, sourceURL string, path string) (int64, error) {
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
	defer response.Body.Close()

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

	file, err := os.OpenFile(partPath, flag, 0o644)
	if err != nil {
		return 0, fmt.Errorf("임시 파일 열기 실패: %w", err)
	}
	written, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return offset + written, fmt.Errorf("동영상 다운로드 실패: %w", copyErr)
	}
	if closeErr != nil {
		return offset + written, fmt.Errorf("임시 파일 닫기 실패: %w", closeErr)
	}

	if err := os.Rename(partPath, path); err != nil {
		return offset + written, fmt.Errorf("다운로드 파일 저장 실패: %w", err)
	}
	return offset + written, nil
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

func lectureMatchesKey(lecture klas.Lecture, key string) bool {
	key = strings.TrimSpace(key)
	return key != "" && (strings.TrimSpace(lecture.ContentID) == key || lectureAttendKey(lecture) == key)
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
