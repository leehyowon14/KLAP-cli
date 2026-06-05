package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kw-klap/klap-cli/internal/account"
	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/ui"
)

func Run(ctx context.Context, args []string) error {
	store, err := account.NewStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "auth":
		return runAuth(ctx, store)
	case "user":
		return runUser(ctx, store, args[1:])
	case "course":
		return runCourse(ctx, store, args[1:])
	case "assignment":
		return runAssignment(ctx, store, args[1:])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runAssignment(ctx context.Context, store *account.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap assignment <list|detail|remind>")
	}

	switch args[0] {
	case "list":
		return runAssignmentList(ctx, store, args[1:])
	case "detail":
		if len(args) != 2 {
			return errors.New("usage: klap assignment detail <과제ID>")
		}
		return runAssignmentDetail(ctx, store, args[1])
	case "remind":
		return errors.New("assignment remind는 다음 원자 작업에서 구현합니다")
	default:
		return fmt.Errorf("unknown assignment command: %s", args[0])
	}
}

func runAssignmentList(ctx context.Context, store *account.Store, args []string) error {
	studentID, err := selectedStudentID(ctx, store, args)
	if err != nil {
		return err
	}
	client, term, err := latestTerm(ctx, store, studentID)
	if err != nil {
		return err
	}

	filter, err := courseFilter(args)
	if err != nil {
		return err
	}
	courses, err := selectedCourses(term, filter)
	if err != nil {
		return err
	}

	rows := make([]assignmentRow, 0)
	for _, selectedCourse := range courses {
		assignments, err := client.Assignments(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := refreshedClientAfterSessionError(ctx, store, studentID, err)
			if refreshErr != nil {
				return refreshErr
			}
			if refreshed {
				client = refreshedClient
				assignments, err = client.Assignments(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			rows = append(rows, assignmentRow{
				ID:         assignmentID(selectedCourse.Index, assignment.OrdSeq),
				CourseName: selectedCourse.Course.Name,
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

	printAssignmentRows(rows)
	return nil
}

func runAssignmentDetail(ctx context.Context, store *account.Store, id string) error {
	studentID, err := selectedStudentID(ctx, store, nil)
	if err != nil {
		return err
	}
	client, term, err := latestTerm(ctx, store, studentID)
	if err != nil {
		return err
	}

	courseIndex, ordSeq, err := parseAssignmentID(id)
	if err != nil {
		return err
	}
	if courseIndex < 1 || courseIndex > len(term.Courses) {
		return fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", courseIndex)
	}

	course := term.Courses[courseIndex-1]
	detail, err := client.AssignmentDetail(ctx, term.Value, course, ordSeq)
	if err != nil {
		refreshedClient, refreshed, refreshErr := refreshedClientAfterSessionError(ctx, store, studentID, err)
		if refreshErr != nil {
			return refreshErr
		}
		if refreshed {
			client = refreshedClient
			detail, err = client.AssignmentDetail(ctx, term.Value, course, ordSeq)
		}
	}
	if err != nil {
		return err
	}

	printAssignmentDetail(id, course.Name, detail)
	return nil
}

func runCourse(ctx context.Context, store *account.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap course list [--user <학번>]")
	}

	switch args[0] {
	case "list":
		studentID, err := selectedStudentID(ctx, store, args[1:])
		if err != nil {
			return err
		}

		client, err := authenticatedClient(ctx, store, studentID)
		if err != nil {
			return err
		}

		terms, err := client.Courses(ctx)
		if err != nil {
			refreshedClient, refreshed, refreshErr := refreshedClientAfterSessionError(ctx, store, studentID, err)
			if refreshErr != nil {
				return refreshErr
			}
			if refreshed {
				terms, err = refreshedClient.Courses(ctx)
			}
		}
		if err != nil {
			return err
		}
		printCourseList(terms)
		return nil
	default:
		return fmt.Errorf("unknown course command: %s", args[0])
	}
}

func runAuth(ctx context.Context, store *account.Store) error {
	credentials, err := ui.RunAuthForm()
	if err != nil {
		return err
	}
	if credentials.StudentID == "" || credentials.Password == "" {
		return errors.New("학번과 비밀번호를 모두 입력해야 합니다")
	}

	client, err := klas.NewClient()
	if err != nil {
		return err
	}

	session, err := client.Login(ctx, credentials.StudentID, credentials.Password)
	if err != nil {
		return fmt.Errorf("로그인 검증 실패: %w", err)
	}

	if err := store.Save(ctx, credentials.StudentID, credentials.Password, session); err != nil {
		return fmt.Errorf("계정 저장 실패: %w", err)
	}

	fmt.Printf("저장 완료: %s\n", credentials.StudentID)
	return nil
}

func runUser(ctx context.Context, store *account.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap user <list|rm>")
	}

	switch args[0] {
	case "list":
		users, err := store.List(ctx)
		if err != nil {
			return err
		}
		currentStudentID, err := store.Current(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Println("저장된 유저가 없습니다")
			return nil
		}
		for _, user := range users {
			prefix := " "
			if user.StudentID == currentStudentID {
				prefix = "*"
			}
			fmt.Printf("%s %s\n", prefix, user.StudentID)
		}
		return nil
	case "select":
		if len(args) != 2 {
			return errors.New("usage: klap user select <학번>")
		}
		if err := store.Select(ctx, args[1]); err != nil {
			return err
		}
		fmt.Printf("현재 유저: %s\n", args[1])
		return nil
	case "rm":
		if len(args) != 2 {
			return errors.New("usage: klap user rm <학번>")
		}
		if err := store.Remove(ctx, args[1]); err != nil {
			return err
		}
		fmt.Printf("삭제 완료: %s\n", args[1])
		return nil
	default:
		return fmt.Errorf("unknown user command: %s", args[0])
	}
}

func printHelp() {
	fmt.Println(`KLAP CLI

Usage:
  klap auth              KLAS 로그인 검증 후 계정 저장
  klap user list         저장된 학번 목록 출력
  klap user select <학번> 현재 유저 선택
  klap user rm <학번>    저장된 계정 삭제
  klap course list       최신 학기 수업 목록 출력
  klap assignment list   과제 목록 출력
  klap assignment detail <과제ID> 과제 상세 출력`)
}

type selectedCourse struct {
	Index  int
	Course klas.Course
}

type assignmentRow struct {
	ID         string
	CourseName string
	Assignment klas.Assignment
}

func latestTerm(ctx context.Context, store *account.Store, studentID string) (*klas.Client, klas.Term, error) {
	client, err := authenticatedClient(ctx, store, studentID)
	if err != nil {
		return nil, klas.Term{}, err
	}

	terms, err := client.Courses(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := refreshedClientAfterSessionError(ctx, store, studentID, err)
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

func courseFilter(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--course" {
			continue
		}
		if i+1 >= len(args) {
			return "", errors.New("--course에는 과목명 또는 course list 번호가 필요합니다")
		}
		return args[i+1], nil
	}
	return "", nil
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

func assignmentID(courseIndex int, ordSeq string) string {
	return fmt.Sprintf("%d:%s", courseIndex, strings.TrimSpace(ordSeq))
}

func parseAssignmentID(id string) (int, string, error) {
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

func printAssignmentRows(rows []assignmentRow) {
	if len(rows) == 0 {
		fmt.Println("과제가 없습니다")
		return
	}

	for _, row := range rows {
		status := "미제출"
		if row.Assignment.Submitted {
			status = "제출"
		}
		fmt.Printf("%s | %s | %s | %s | %s\n",
			row.ID,
			formatTime(row.Assignment.DueAt),
			status,
			row.CourseName,
			row.Assignment.Title,
		)
	}
}

func printAssignmentDetail(id string, courseName string, detail klas.AssignmentDetail) {
	status := "미제출"
	if detail.Submitted {
		status = "제출"
	}

	fmt.Printf("ID: %s\n", id)
	fmt.Printf("과목: %s\n", courseName)
	fmt.Printf("제목: %s\n", detail.Title)
	fmt.Printf("마감: %s\n", formatTime(detail.DueAt))
	fmt.Printf("상태: %s\n", status)
	if detail.ReportType != "" {
		fmt.Printf("제출 방식: %s\n", detail.ReportType)
	}
	if detail.SubmitFileType != "" {
		fmt.Printf("파일 형식: %s\n", detail.SubmitFileType)
	}
	if detail.FileLimitMB != "" {
		fmt.Printf("파일 제한: %sMB\n", detail.FileLimitMB)
	}
	if detail.ContentText != "" {
		fmt.Printf("\n%s\n", detail.ContentText)
	}
	if detail.SubmittedText != "" || detail.SubmittedTitle != "" {
		fmt.Println("\n내 제출")
		if detail.SubmittedTitle != "" {
			fmt.Printf("제목: %s\n", detail.SubmittedTitle)
		}
		if detail.SubmittedText != "" {
			fmt.Println(detail.SubmittedText)
		}
	}
	if detail.FinalScore != "" && detail.FinalScore != "<nil>" {
		fmt.Printf("\n점수: %s\n", detail.FinalScore)
	}
	if detail.TutorText != "" {
		fmt.Printf("\n피드백:\n%s\n", detail.TutorText)
	}
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "마감 확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}

func selectedStudentID(ctx context.Context, store *account.Store, args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--user" {
			continue
		}
		if i+1 >= len(args) {
			return "", errors.New("--user에는 학번이 필요합니다")
		}
		return args[i+1], nil
	}

	currentStudentID, err := store.Current(ctx)
	if err != nil {
		return "", err
	}
	if currentStudentID != "" {
		return currentStudentID, nil
	}

	users, err := store.List(ctx)
	if err != nil {
		return "", err
	}
	switch len(users) {
	case 0:
		return "", errors.New("저장된 유저가 없습니다. 먼저 klap auth를 실행하세요")
	case 1:
		return users[0].StudentID, nil
	default:
		return "", errors.New("저장된 유저가 여러 명입니다. --user <학번>을 지정하세요")
	}
}

func authenticatedClient(ctx context.Context, store *account.Store, studentID string) (*klas.Client, error) {
	client, err := klas.NewClient()
	if err != nil {
		return nil, err
	}

	session, err := store.LoadSession(ctx, studentID)
	if err == nil {
		client.SetSession(session)
		return client, nil
	}

	password, err := store.LoadPassword(ctx, studentID)
	if err != nil {
		return nil, err
	}
	session, err = client.Login(ctx, studentID, password)
	if err != nil {
		return nil, fmt.Errorf("재로그인 실패: %w", err)
	}
	if err := store.SaveSession(ctx, studentID, session); err != nil {
		return nil, err
	}
	return client, nil
}

func refreshedClientAfterSessionError(ctx context.Context, store *account.Store, studentID string, err error) (*klas.Client, bool, error) {
	if !errors.Is(err, klas.ErrSessionExpired) {
		return nil, false, err
	}

	client, clientErr := klas.NewClient()
	if clientErr != nil {
		return nil, true, clientErr
	}
	password, passwordErr := store.LoadPassword(ctx, studentID)
	if passwordErr != nil {
		return nil, true, passwordErr
	}
	session, loginErr := client.Login(ctx, studentID, password)
	if loginErr != nil {
		return nil, true, fmt.Errorf("재로그인 실패: %w", loginErr)
	}
	if saveErr := store.SaveSession(ctx, studentID, session); saveErr != nil {
		return nil, true, saveErr
	}
	return client, true, nil
}

func printCourseList(terms []klas.Term) {
	if len(terms) == 0 {
		fmt.Println("수강 학기가 없습니다")
		return
	}

	term := terms[0]
	fmt.Printf("%s (%s)\n", term.Label, term.Value)
	if len(term.Courses) == 0 {
		fmt.Println("수업이 없습니다")
		return
	}

	for index, course := range term.Courses {
		fmt.Printf("%d. %s\n", index+1, strings.TrimSpace(course.Name))
	}
}
