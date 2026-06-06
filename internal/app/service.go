package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

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

type AssignmentDetailResult struct {
	ID         string
	TermValue  string
	CourseName string
	Detail     klas.AssignmentDetail
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

func (s *Service) CourseList(ctx context.Context, user UserOption) ([]klas.Term, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return nil, err
	}

	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, err
	}

	terms, err := client.Courses(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return nil, refreshErr
		}
		if refreshed {
			terms, err = refreshedClient.Courses(ctx)
		}
	}
	if err != nil {
		return nil, err
	}
	return terms, nil
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
		Detail:     detail,
	}, nil
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

func buildReminderNotes(result AssignmentDetailResult) string {
	detail := result.Detail
	var builder strings.Builder

	if strings.TrimSpace(detail.ContentText) != "" {
		builder.WriteString(strings.TrimSpace(detail.ContentText))
		builder.WriteString("\n\n")
	}

	builder.WriteString("=========================================\n\n")
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

	terms, err := client.Courses(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return nil, klas.Term{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			terms, err = client.Courses(ctx)
		}
	}
	if err != nil {
		return nil, klas.Term{}, err
	}
	if len(terms) == 0 {
		return nil, klas.Term{}, errors.New("수강 학기가 없습니다")
	}
	return client, terms[0], nil
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
	if err := s.store.SaveSession(ctx, studentID, session); err != nil {
		return nil, err
	}
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
	if saveErr := s.store.SaveSession(ctx, studentID, session); saveErr != nil {
		return nil, true, saveErr
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
