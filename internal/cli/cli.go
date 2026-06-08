package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	bubblesprogress "github.com/charmbracelet/bubbles/progress"
	"github.com/kw-klap/klap-cli/internal/account"
	"github.com/kw-klap/klap-cli/internal/app"
	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/ui"
)

func Run(ctx context.Context, args []string) error {
	store, err := account.NewStore()
	if err != nil {
		return err
	}
	service := app.NewService(store)

	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "auth":
		return runAuth(ctx, service)
	case "user":
		return runUser(ctx, service, args[1:])
	case "course":
		return runCourse(ctx, service, args[1:])
	case "subject":
		return runSubject(ctx, service, args[1:])
	case "term":
		return runTerm(ctx, service, args[1:])
	case "assignment":
		return runAssignment(ctx, service, args[1:])
	case "notice":
		return runNotice(ctx, service, args[1:])
	case "timetable":
		return runTimetable(ctx, service, args[1:])
	case "attendance":
		return runAttendance(ctx, service, args[1:])
	case "grade":
		return runGrade(ctx, service, args[1:])
	case "syllabus":
		return runSyllabus(ctx, service, args[1:])
	case "lecture":
		return runLecture(ctx, service, args[1:])
	case "attend":
		return runAttend(ctx, service, args[1:])
	case "academic":
		return runAcademic(ctx, service, args[1:])
	case "config":
		return runConfig(ctx, service, args[1:])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runConfig(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap config <reminder>")
	}

	switch args[0] {
	case "reminder":
		return runConfigReminder(ctx, service, args[1:])
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

func runConfigReminder(ctx context.Context, service *app.Service, args []string) error {
	_ = ctx

	if len(args) == 0 {
		settings, err := service.ReminderSettings()
		if err != nil {
			return err
		}
		printReminderSettings(settings)
		return nil
	}

	name, useExistingList, ok, err := parseReminderConfigArgs(args)
	if err != nil {
		return err
	}
	if ok {
		settings, err := service.SetReminderConfig(name, useExistingList)
		if err != nil {
			return err
		}
		printReminderSettings(settings)
		return nil
	}

	return errors.New(`usage: klap config reminder [--name "Kwangwoon Univ." [--use-existing-list]]`)
}

func parseReminderConfigArgs(args []string) (name string, useExistingList bool, ok bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return "", false, false, errors.New("--name에는 리마인더 목록 이름이 필요합니다")
			}
			name = args[i+1]
			i++
		case "--use-existing-list":
			useExistingList = true
		default:
			return "", false, false, fmt.Errorf("unknown reminder config option: %s", args[i])
		}
	}
	if name == "" {
		return "", false, false, nil
	}
	return name, useExistingList, true, nil
}

func runAuth(ctx context.Context, service *app.Service) error {
	credentials, err := ui.RunAuthForm()
	if err != nil {
		return err
	}
	if err := service.Authenticate(ctx, credentials.StudentID, credentials.Password); err != nil {
		return err
	}

	fmt.Printf("저장 완료: %s\n", credentials.StudentID)
	return nil
}

func runUser(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap user <list|select|rm>")
	}

	switch args[0] {
	case "list":
		users, err := service.Users(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Println("저장된 유저가 없습니다")
			return nil
		}
		for _, user := range users {
			prefix := " "
			if user.Current {
				prefix = "*"
			}
			fmt.Printf("%s %s\n", prefix, user.User.StudentID)
		}
		return nil
	case "select":
		if len(args) != 2 {
			return errors.New("usage: klap user select <학번>")
		}
		if err := service.SelectUser(ctx, args[1]); err != nil {
			return err
		}
		fmt.Printf("현재 유저: %s\n", args[1])
		return nil
	case "rm":
		if len(args) != 2 {
			return errors.New("usage: klap user rm <학번>")
		}
		if err := service.RemoveUser(ctx, args[1]); err != nil {
			return err
		}
		fmt.Printf("삭제 완료: %s\n", args[1])
		return nil
	default:
		return fmt.Errorf("unknown user command: %s", args[0])
	}
}

func runCourse(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap course list [--user <학번>]")
	}

	switch args[0] {
	case "list":
		terms, err := service.CourseList(ctx, app.UserOption{StudentID: userFlag(args[1:])})
		if err != nil {
			return err
		}
		printCourseList(terms)
		return nil
	default:
		return fmt.Errorf("unknown course command: %s", args[0])
	}
}

func runSubject(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap subject <search>")
	}

	switch args[0] {
	case "search":
		opts, err := subjectSearchOptions(args[1:])
		if err != nil {
			return err
		}
		result, err := service.SubjectSearch(ctx, opts)
		if err != nil {
			return err
		}
		printSubjectSearch(result)
		return nil
	default:
		return fmt.Errorf("unknown subject command: %s", args[0])
	}
}

func runTerm(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap term <list|select>")
	}

	switch args[0] {
	case "list":
		rows, err := service.TermList(ctx, app.TermListOptions{
			User: app.UserOption{StudentID: userFlag(args[1:])},
		})
		if err != nil {
			return err
		}
		printTermRows(rows)
		return nil
	case "select":
		if len(args) < 2 {
			return errors.New("usage: klap term select <학기번호|학기값> [--user <학번>]")
		}
		settings, err := service.SelectTerm(ctx, args[1], app.UserOption{StudentID: userFlag(args[2:])})
		if err != nil {
			return err
		}
		fmt.Printf("현재 학기: %s (%s)\n", settings.Label, settings.Value)
		return nil
	default:
		return fmt.Errorf("unknown term command: %s", args[0])
	}
}

func runAssignment(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap assignment <list|detail|open|remind>")
	}

	switch args[0] {
	case "list":
		opts, err := assignmentListOptions(args[1:])
		if err != nil {
			return err
		}
		rows, err := service.AssignmentList(ctx, opts)
		if err != nil {
			return err
		}
		printAssignmentRows(rows)
		return nil
	case "detail":
		if len(args) != 2 {
			return errors.New("usage: klap assignment detail <과제ID>")
		}
		detail, err := service.AssignmentDetail(ctx, args[1], app.UserOption{})
		if err != nil {
			return err
		}
		printAssignmentDetail(detail)
		return nil
	case "open":
		if len(args) != 2 {
			return errors.New("usage: klap assignment open <과제ID>")
		}
		result, err := service.AssignmentOpenURL(ctx, args[1], app.UserOption{})
		return openAndPrintURL(result.URL, err)
	case "remind":
		return runAssignmentRemind(ctx, service, args[1:])
	default:
		return fmt.Errorf("unknown assignment command: %s", args[0])
	}
}

func runNotice(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap notice <list|detail|open>")
	}

	switch args[0] {
	case "list":
		opts, err := noticeListOptions(args[1:])
		if err != nil {
			return err
		}
		rows, err := service.NoticeList(ctx, opts)
		if err != nil {
			return err
		}
		printNoticeRows(rows)
		return nil
	case "detail":
		if len(args) != 2 {
			return errors.New("usage: klap notice detail <공지ID>")
		}
		detail, err := service.NoticeDetail(ctx, args[1], app.UserOption{})
		if err != nil {
			return err
		}
		printNoticeDetail(detail)
		return nil
	case "open":
		if len(args) != 2 {
			return errors.New("usage: klap notice open <공지ID>")
		}
		result, err := service.NoticeOpenURL(ctx, args[1], app.UserOption{})
		return openAndPrintURL(result.URL, err)
	default:
		return fmt.Errorf("unknown notice command: %s", args[0])
	}
}

func runTimetable(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown timetable command: %s", args[0])
	}

	result, err := service.Timetable(ctx, app.TimetableOptions{
		User: app.UserOption{StudentID: userFlag(args)},
	})
	if err != nil {
		return err
	}
	printTimetable(result)
	return nil
}

func runAttendance(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "detail" {
		if len(args) != 2 {
			return errors.New("usage: klap attendance detail <과목명|번호|학정번호>")
		}
		result, err := service.AttendanceDetail(ctx, app.AttendanceDetailOptions{
			User:     app.UserOption{},
			Selector: args[1],
		})
		if err != nil {
			return err
		}
		printAttendanceDetail(result)
		return nil
	}
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown attendance command: %s", args[0])
	}

	result, err := service.AttendanceList(ctx, app.AttendanceListOptions{
		User: app.UserOption{StudentID: userFlag(args)},
	})
	if err != nil {
		return err
	}
	printAttendanceList(result)
	return nil
}

func runGrade(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown grade command: %s", args[0])
	}

	result, err := service.Grade(ctx, app.GradeOptions{
		User: app.UserOption{StudentID: userFlag(args)},
	})
	if err != nil {
		return err
	}
	printGrade(result)
	return nil
}

func runSyllabus(ctx context.Context, service *app.Service, args []string) error {
	opts, err := syllabusOptions(args)
	if err != nil {
		return err
	}
	result, err := service.Syllabus(ctx, opts)
	if err != nil {
		return err
	}
	printSyllabus(result)
	return nil
}

func runAcademic(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap academic <list>")
	}

	switch args[0] {
	case "list":
		year, err := academicYearFlag(args[1:])
		if err != nil {
			return err
		}
		result, err := service.AcademicList(ctx, app.AcademicListOptions{Year: year})
		if err != nil {
			return err
		}
		printAcademicList(result)
		return nil
	default:
		return fmt.Errorf("unknown academic command: %s", args[0])
	}
}

func runLecture(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap lecture <list|download|attend>")
	}

	switch args[0] {
	case "list":
		opts, err := lectureListOptions(args[1:])
		if err != nil {
			return err
		}
		rows, err := service.LectureList(ctx, opts)
		if err != nil {
			return err
		}
		printLectureRows(rows)
		return nil
	case "status":
		opts, err := lectureListOptions(args[1:])
		if err != nil {
			return err
		}
		rows, err := service.LectureList(ctx, opts)
		if err != nil {
			return err
		}
		printLectureStatusRows(rows)
		return nil
	case "download":
		if len(args) < 2 {
			return errors.New("usage: klap lecture download <과목명|과목번호|강의ID> [--dir <경로>]")
		}
		dir, err := dirFlag(args[2:])
		if err != nil {
			return err
		}
		if !looksLikeLectureID(args[1]) {
			result, err := service.DownloadAllLectures(ctx, app.LectureDownloadAllOptions{
				User:         app.UserOption{StudentID: userFlag(args[2:])},
				CourseFilter: args[1],
				Dir:          dir,
			})
			if err != nil {
				return err
			}
			printLectureDownloadAllResult(result)
			return nil
		}
		result, err := service.DownloadLecture(ctx, args[1], app.LectureDownloadOptions{
			User: app.UserOption{StudentID: userFlag(args[2:])},
			Dir:  dir,
		})
		if err != nil {
			return err
		}
		fmt.Printf("다운로드 완료: %s (%s)\n", result.Path, formatBytes(result.Bytes))
		return nil
	case "attend":
		return runLectureAttend(ctx, service, args[1:])
	case "open":
		if len(args) != 2 {
			return errors.New("usage: klap lecture open <강의ID>")
		}
		result, err := service.LectureOpenURL(ctx, args[1], app.UserOption{})
		return openAndPrintURL(result.URL, err)
	default:
		return fmt.Errorf("unknown lecture command: %s", args[0])
	}
}

func runLectureAttend(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap lecture attend <강의ID|all> [--course <과목명|번호>] [--interval <초>]")
	}

	interval, err := intervalFlag(args[1:])
	if err != nil {
		return err
	}
	onProgress := printLectureProgress

	if args[0] == "all" {
		course, err := courseFilter(args[1:])
		if err != nil {
			return err
		}
		result, err := service.AttendAllLectures(ctx, app.LectureAttendAllOptions{
			User:         app.UserOption{StudentID: userFlag(args[1:])},
			CourseFilter: course,
			Interval:     interval,
			OnProgress:   onProgress,
		})
		if err != nil {
			return err
		}
		printLectureAttendAllResult(result)
		return nil
	}

	result, err := service.AttendLecture(ctx, args[0], app.LectureAttendOptions{
		User:       app.UserOption{StudentID: userFlag(args[1:])},
		Interval:   interval,
		OnProgress: onProgress,
	})
	if err != nil {
		return err
	}
	fmt.Printf("수강 완료: %s | %s\n", formatLectureProgress(result.Lecture, result.Progress), result.Lecture.Lecture.Title)
	return nil
}

func runAttend(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap attend <all|과목명|과목번호> [--interval <초>]")
	}

	courseFilter := ""
	flagArgs := args[1:]
	if args[0] != "all" {
		courseFilter = args[0]
	}

	interval, err := intervalFlag(flagArgs)
	if err != nil {
		return err
	}
	result, err := service.AttendAllLectures(ctx, app.LectureAttendAllOptions{
		User:         app.UserOption{StudentID: userFlag(flagArgs)},
		CourseFilter: courseFilter,
		Interval:     interval,
		OnProgress:   printLectureProgress,
	})
	if err != nil {
		return err
	}
	printLectureAttendAllResult(result)
	return nil
}

func runAssignmentRemind(ctx context.Context, service *app.Service, args []string) error {
	auto := hasFlag(args, "--auto")
	if !auto {
		return syncAssignmentReminders(ctx, service, args)
	}

	fmt.Println("과제 reminder 자동 동기화를 시작합니다. 종료하려면 Ctrl+C를 누르세요.")
	for {
		if err := syncAssignmentReminders(ctx, service, args); err != nil {
			fmt.Printf("동기화 실패: %v\n", err)
		}

		timer := time.NewTimer(30 * time.Minute)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func syncAssignmentReminders(ctx context.Context, service *app.Service, args []string) error {
	opts, err := assignmentListOptions(args)
	if err != nil {
		return err
	}

	result, err := service.SyncAssignmentReminders(ctx, opts)
	if err != nil {
		return err
	}
	if result.EligibleCount == 0 {
		fmt.Println("등록할 과제 reminder가 없습니다")
		return nil
	}

	fmt.Printf("Reminder 동기화 완료: 생성 %d, 갱신 %d, 완료 %d, 제외 %d\n",
		result.Result.Created,
		result.Result.Updated,
		result.Result.Completed,
		result.Result.Skipped,
	)
	return nil
}

func assignmentListOptions(args []string) (app.AssignmentListOptions, error) {
	course, err := courseFilter(args)
	if err != nil {
		return app.AssignmentListOptions{}, err
	}
	return app.AssignmentListOptions{
		User:         app.UserOption{StudentID: userFlag(args)},
		CourseFilter: course,
	}, nil
}

func noticeListOptions(args []string) (app.NoticeListOptions, error) {
	course, err := courseFilter(args)
	if err != nil {
		return app.NoticeListOptions{}, err
	}
	return app.NoticeListOptions{
		User:         app.UserOption{StudentID: userFlag(args)},
		CourseFilter: course,
	}, nil
}

func lectureListOptions(args []string) (app.LectureListOptions, error) {
	course, err := courseFilter(args)
	if err != nil {
		return app.LectureListOptions{}, err
	}
	return app.LectureListOptions{
		User:         app.UserOption{StudentID: userFlag(args)},
		CourseFilter: course,
	}, nil
}

func syllabusOptions(args []string) (app.SyllabusOptions, error) {
	if len(args) == 0 {
		return app.SyllabusOptions{}, errors.New("usage: klap syllabus <과목명|과목번호|학정번호> [--term YYYY-S] [--user <학번>]")
	}

	opts := app.SyllabusOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.SyllabusOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--user":
			if i+1 >= len(args) {
				return app.SyllabusOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.SyllabusOptions{}, fmt.Errorf("unknown syllabus option: %s", args[i])
			}
			if opts.Selector != "" {
				return app.SyllabusOptions{}, errors.New("강의계획서 조회 대상은 하나만 지정할 수 있습니다")
			}
			opts.Selector = args[i]
		}
	}
	if strings.TrimSpace(opts.Selector) == "" {
		return app.SyllabusOptions{}, errors.New("usage: klap syllabus <과목명|과목번호|학정번호> [--term YYYY-S] [--user <학번>]")
	}
	return opts, nil
}

func subjectSearchOptions(args []string) (app.SubjectSearchOptions, error) {
	opts := app.SubjectSearchOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return app.SubjectSearchOptions{}, errors.New("--name에는 과목명이 필요합니다")
			}
			opts.Name = args[i+1]
			i++
		case "--professor":
			if i+1 >= len(args) {
				return app.SubjectSearchOptions{}, errors.New("--professor에는 교수명이 필요합니다")
			}
			opts.Professor = args[i+1]
			i++
		case "--term":
			if i+1 >= len(args) {
				return app.SubjectSearchOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		default:
			return app.SubjectSearchOptions{}, fmt.Errorf("unknown subject search option: %s", args[i])
		}
	}
	if strings.TrimSpace(opts.Name) == "" && strings.TrimSpace(opts.Professor) == "" {
		return app.SubjectSearchOptions{}, errors.New("usage: klap subject search [--name <과목명>] [--professor <교수명>] [--term YYYY-S]")
	}
	return opts, nil
}

func dirFlag(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--dir" {
			continue
		}
		if i+1 >= len(args) {
			return "", errors.New("--dir에는 다운로드 폴더 경로가 필요합니다")
		}
		return args[i+1], nil
	}
	return "", nil
}

func intervalFlag(args []string) (time.Duration, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--interval" {
			continue
		}
		if i+1 >= len(args) {
			return 0, errors.New("--interval에는 초 단위 숫자가 필요합니다")
		}
		seconds, err := strconv.Atoi(args[i+1])
		if err != nil || seconds <= 0 {
			return 0, errors.New("--interval에는 1 이상의 초 단위 숫자가 필요합니다")
		}
		return time.Duration(seconds) * time.Second, nil
	}
	return 60 * time.Second, nil
}

func userFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "--user" {
			continue
		}
		if i+1 >= len(args) {
			return ""
		}
		return args[i+1]
	}
	return ""
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

func academicYearFlag(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--year" {
			continue
		}
		if i+1 >= len(args) {
			return "", errors.New("--year에는 YYYY 형식의 연도가 필요합니다")
		}
		year := strings.TrimSpace(args[i+1])
		if len(year) != 4 {
			return "", errors.New("--year에는 YYYY 형식의 연도가 필요합니다")
		}
		for _, r := range year {
			if r < '0' || r > '9' {
				return "", errors.New("--year에는 YYYY 형식의 연도가 필요합니다")
			}
		}
		return year, nil
	}
	return "", nil
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}

func looksLikeLectureID(value string) bool {
	return strings.Contains(strings.TrimSpace(value), ":")
}

func printHelp() {
	fmt.Println(`KLAP CLI

Usage:
  klap auth              KLAS 로그인 검증 후 계정 저장
  klap user list         저장된 학번 목록 출력
  klap user select <학번> 현재 유저 선택
  klap user rm <학번>    저장된 계정 삭제
  klap term list         수강 학기 목록 출력
  klap term select <학기번호|학기값> 현재 학기 선택
  klap course list       현재 학기 수업 목록 출력
  klap subject search    과목 검색
  klap assignment list   과제 목록 출력
  klap assignment detail <과제ID> 과제 상세 출력
  klap assignment open <과제ID> 과제 원문 열기
  klap assignment remind 과제 마감 reminder 동기화
  klap notice list       강의 공지 목록 출력
  klap notice detail <공지ID> 강의 공지 상세 출력
  klap notice open <공지ID> 강의 공지 원문 열기
  klap timetable         현재 학기 시간표 출력
  klap attendance        출석 현황 출력
  klap attendance detail <과목명|번호|학정번호> 주차별 출석 상세 출력
  klap grade             성적 조회
  klap syllabus <과목명|과목번호|학정번호> 강의계획서 출력
  klap academic list     학사일정 목록 출력
  klap lecture list      온라인 강의 목록 출력
  klap lecture status    온라인 강의 수강 상태 출력
  klap lecture download <과목명|과목번호|강의ID> 온라인 강의 다운로드
  klap lecture attend <강의ID> 특정 온라인 강의 자동 수강
  klap lecture open <강의ID> 온라인 강의 열기
  klap attend <all|과목명|과목번호> 온라인 강의와 학습활동 자동 수강
  klap config reminder  reminder 설정 확인/변경`)
}

func printReminderSettings(settings app.ReminderSettings) {
	fmt.Printf("Reminder list: %s\n", settings.ListName)
	if settings.UseExistingList {
		fmt.Println("List mode: existing only")
	} else {
		fmt.Println("List mode: create if missing")
	}
	fmt.Printf("Alarm before: %d분\n", settings.AlarmBeforeMin)
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

func printSubjectSearch(result app.SubjectSearchResult) {
	fmt.Printf("%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Rows) == 0 {
		fmt.Println("검색된 과목이 없습니다")
		return
	}

	fmt.Println("학정번호 | 과목명 | 교수 | 강의시간")
	for _, row := range result.Rows {
		times := formatSyllabusTimes(row.Times)
		if times == "" {
			times = "강의시간 확인 필요"
		}
		if row.Err != nil {
			times = "강의시간 확인 실패"
		}
		fmt.Printf("%s | %s | %s | %s\n",
			emptyFallback(row.CourseCode, "-"),
			emptyFallback(row.Name, "-"),
			emptyFallback(row.Professor, "-"),
			times,
		)
	}
}

func printTermRows(rows []app.TermRow) {
	if len(rows) == 0 {
		fmt.Println("수강 학기가 없습니다")
		return
	}

	for _, row := range rows {
		prefix := " "
		if row.Current {
			prefix = "*"
		}
		fmt.Printf("%s %d. %s (%s)\n", prefix, row.Index, row.Term.Label, row.Term.Value)
	}
}

func printAssignmentRows(rows []app.AssignmentRow) {
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

func printAssignmentDetail(result app.AssignmentDetailResult) {
	detail := result.Detail
	status := "미제출"
	if detail.Submitted {
		status = "제출"
	}

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("과목: %s\n", result.CourseName)
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
		fmt.Printf("\n%s\n", linkifyForTerminal(detail.ContentText))
	}
	if detail.SubmittedText != "" || detail.SubmittedTitle != "" {
		fmt.Println("\n내 제출")
		if detail.SubmittedTitle != "" {
			fmt.Printf("제목: %s\n", detail.SubmittedTitle)
		}
		if detail.SubmittedText != "" {
			fmt.Println(linkifyForTerminal(detail.SubmittedText))
		}
	}
	if detail.FinalScore != "" && detail.FinalScore != "<nil>" {
		fmt.Printf("\n점수: %s\n", detail.FinalScore)
	}
	if detail.TutorText != "" {
		fmt.Printf("\n피드백:\n%s\n", linkifyForTerminal(detail.TutorText))
	}
	if result.DetailURL != "" {
		fmt.Printf("\n원문: %s\n", linkifyForTerminal(result.DetailURL))
	}
}

func printNoticeRows(rows []app.NoticeRow) {
	if len(rows) == 0 {
		fmt.Println("강의 공지가 없습니다")
		return
	}

	for _, row := range rows {
		prefix := " "
		if row.Notice.Top {
			prefix = "!"
		}
		fmt.Printf("%s %s | %s | %s | %s | %s\n",
			prefix,
			row.ID,
			formatNoticeTime(row.Notice.Registered),
			row.CourseName,
			row.Notice.Author,
			row.Notice.Title,
		)
	}
}

func printNoticeDetail(result app.NoticeDetailResult) {
	detail := result.Detail

	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("과목: %s\n", result.CourseName)
	fmt.Printf("제목: %s\n", detail.Title)
	if detail.Author != "" {
		fmt.Printf("작성자: %s\n", detail.Author)
	}
	fmt.Printf("작성일: %s\n", formatNoticeTime(detail.Registered))
	if detail.Top {
		fmt.Println("중요: 예")
	}
	if detail.ReadCount != "" {
		fmt.Printf("조회수: %s\n", detail.ReadCount)
	}
	if detail.Attachment != "" {
		fmt.Printf("첨부 묶음: %s\n", detail.Attachment)
	}
	fmt.Printf("원문: %s\n", linkifyForTerminal(result.DetailURL))
	if detail.ContentText != "" {
		fmt.Printf("\n%s\n", linkifyForTerminal(detail.ContentText))
	}
}

func printTimetable(result app.TimetableResult) {
	fmt.Printf("%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Entries) == 0 {
		fmt.Println("시간표가 없습니다")
		return
	}

	printedOnlineHeader := false
	currentWeekday := 0
	for _, entry := range result.Entries {
		if entry.Online {
			if !printedOnlineHeader {
				fmt.Println("\n온라인/미지정")
				printedOnlineHeader = true
			}
			fmt.Printf("  %s | %s | %s | %s\n",
				formatPeriod(entry.Period, entry.Span),
				entry.SubjectName,
				emptyFallback(entry.Room, "강의실 미지정"),
				emptyFallback(entry.Professor, "교수 미지정"),
			)
			continue
		}

		if entry.Weekday != currentWeekday {
			if currentWeekday != 0 {
				fmt.Println()
			}
			currentWeekday = entry.Weekday
			fmt.Println(weekdayLabel(entry.Weekday))
		}
		fmt.Printf("  %s | %s | %s | %s\n",
			formatPeriod(entry.Period, entry.Span),
			entry.SubjectName,
			emptyFallback(entry.Room, "강의실 미지정"),
			emptyFallback(entry.Professor, "교수 미지정"),
		)
	}
}

func printAttendanceList(result app.AttendanceListResult) {
	fmt.Printf("%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Rows) == 0 {
		fmt.Println("출석 현황이 없습니다")
		return
	}

	fmt.Println("번호 | 학정번호 | 과목 | 교수 | 강의시간 | 출석 요약")
	for _, row := range result.Rows {
		course := row.Course
		summary := attendanceSummary(row)
		fmt.Printf("%d. %s | %s | %s | %s | %s\n",
			row.Index,
			emptyFallback(course.CourseCode, "-"),
			emptyFallback(course.Name, "-"),
			emptyFallback(course.Professor, "-"),
			emptyFallback(course.Weekday, "확인 필요"),
			summary,
		)
	}
}

func printAttendanceDetail(result app.AttendanceDetailResult) {
	row := result.Row
	course := row.Course
	fmt.Printf("%s (%s)\n", result.Term.Label, result.Term.Value)
	fmt.Printf("과목: %s\n", course.Name)
	fmt.Printf("학정번호: %s\n", emptyFallback(course.CourseCode, "-"))
	fmt.Printf("교수: %s\n", emptyFallback(course.Professor, "-"))
	fmt.Printf("강의시간: %s\n", emptyFallback(course.Weekday, "확인 필요"))
	fmt.Printf("출석 요약: %s\n", attendanceSummary(row))
	if row.Err != nil {
		fmt.Printf("상세 오류: %v\n", row.Err)
		return
	}
	if len(row.Sessions) == 0 {
		fmt.Println("상세 출석 내역이 없습니다")
		return
	}

	fmt.Println("\n주차별 출석")
	for _, session := range row.Sessions {
		parts := make([]string, 0, len(session.Slots))
		for _, slot := range session.Slots {
			parts = append(parts, fmt.Sprintf("%d차시 %s %s", slot.Index, attendanceMarkLabel(slot.Mark), formatAttendanceDate(slot.Date)))
		}
		fmt.Printf("%s주차 | %s\n", emptyFallback(session.Week, "-"), strings.Join(parts, " / "))
	}
}

func printGrade(result app.GradeResult) {
	report := result.Report
	summary := report.Summary
	fmt.Println("성적")
	fmt.Printf("신청 학점: 전체 %d / 전공 %d / 교양 %d / 기타 %d\n",
		summary.AppliedCredits,
		summary.MajorAppliedCredits,
		summary.CultureAppliedCredits,
		summary.EtcAppliedCredits,
	)
	fmt.Printf("취득 학점: 전체 %d / 전공 %d / 교양 %d / 기타 %d\n",
		summary.EarnedCredits,
		summary.MajorEarnedCredits,
		summary.CultureEarnedCredits,
		summary.EtcEarnedCredits,
	)
	fmt.Printf("평점: %s / 재수강 반영 평점: %s\n",
		emptyFallback(summary.GPA, "-"),
		emptyFallback(summary.RetakeGPA, "-"),
	)
	if summary.DeletedCredits > 0 {
		fmt.Printf("삭제 학점: %d\n", summary.DeletedCredits)
	}

	if len(report.Terms) == 0 {
		fmt.Println("\n성적 내역이 없습니다")
		return
	}

	for _, term := range report.Terms {
		fmt.Printf("\n%s\n", emptyFallback(term.Label, "-"))
		fmt.Println("과목 | 이수구분 | 학점 | 성적 | 재수강 | 학정번호")
		for _, course := range term.Courses {
			fmt.Printf("%s | %s | %d | %s | %s | %s\n",
				course.Name,
				emptyFallback(course.CourseType, "-"),
				course.Credits,
				gradeLabel(course),
				yesNo(course.Retake),
				emptyFallback(course.CourseCode, "-"),
			)
		}
	}
}

func gradeLabel(course klas.GradeCourse) string {
	if strings.TrimSpace(course.Grade) != "" {
		return strings.TrimSpace(course.Grade)
	}
	if !course.TermCheckOpen && !course.TermFinished {
		return "미공개"
	}
	return "-"
}

func yesNo(value bool) string {
	if value {
		return "Y"
	}
	return "N"
}

func attendanceSummary(row app.AttendanceRow) string {
	if row.Err != nil {
		return "상세 확인 실패"
	}
	counts := map[string]int{}
	total := 0
	for _, session := range row.Sessions {
		for _, slot := range session.Slots {
			mark := emptyFallback(slot.Mark, slot.Status)
			if mark == "" {
				continue
			}
			counts[mark]++
			total++
		}
	}
	if total == 0 {
		return "상세 없음"
	}
	return fmt.Sprintf("출석 %d / 결석 %d / 지각 %d / 조퇴 %d / 공결 %d / 전체 %d",
		counts["O"],
		counts["X"],
		counts["L"],
		counts["R"],
		counts["A"],
		total,
	)
}

func attendanceMarkLabel(mark string) string {
	switch mark {
	case "O":
		return "출석"
	case "X":
		return "결석"
	case "L":
		return "지각"
	case "R":
		return "조퇴"
	case "A":
		return "공결"
	default:
		return emptyFallback(mark, "-")
	}
}

func formatAttendanceDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) != 8 {
		return value
	}
	return value[:4] + "-" + value[4:6] + "-" + value[6:]
}

func printSyllabus(result app.SyllabusResult) {
	syllabus := result.Syllabus
	fmt.Printf("%s (%s)\n", result.Term.Label, result.Term.Value)
	fmt.Printf("과목: %s\n", emptyFallback(syllabus.FullName, emptyFallback(syllabus.KoreanName, result.Course.Name)))
	fmt.Printf("학정번호: %s\n", emptyFallback(syllabus.CourseCode, "확인 필요"))
	fmt.Printf("과목ID: %s\n", result.SubjectID)
	if syllabus.CourseType != "" || syllabus.Credits != "" {
		fmt.Printf("이수/학점: %s / %s\n", emptyFallback(syllabus.CourseType, "-"), emptyFallback(syllabus.Credits, "-"))
	}
	if syllabus.Professor != "" {
		professor := syllabus.Professor
		if syllabus.ProfessorTitle != "" {
			professor += " (" + syllabus.ProfessorTitle + ")"
		}
		fmt.Printf("담당교수: %s\n", professor)
	}
	if len(syllabus.Times) > 0 {
		fmt.Printf("강의시간: %s\n", formatSyllabusTimes(syllabus.Times))
	}
	if syllabus.Operation != "" {
		fmt.Printf("운영방식: %s\n", syllabus.Operation)
	}
	if syllabus.Competency != "" {
		fmt.Printf("대표역량: %s\n", syllabus.Competency)
	}
	if syllabus.Summary != "" {
		fmt.Printf("\n개요\n%s\n", syllabus.Summary)
	}
	if syllabus.Purpose != "" {
		fmt.Printf("\n학습목표\n%s\n", syllabus.Purpose)
	}
	if syllabus.Outcome != "" {
		fmt.Printf("\n학습성과\n%s\n", syllabus.Outcome)
	}
	if syllabus.BookName != "" {
		fmt.Printf("\n교재: %s\n", syllabus.BookName)
	}
	fmt.Printf("\n평가: %s\n", formatSyllabusEvaluation(syllabus.Evaluation))
	if len(syllabus.Schedule) > 0 {
		fmt.Println("\n주차별 계획")
		for _, week := range syllabus.Schedule {
			fmt.Printf("  %d주차 | %s", week.Week, strings.ReplaceAll(week.Topic, "\n", " / "))
			if week.SubNote != "" {
				fmt.Printf(" | %s", strings.ReplaceAll(week.SubNote, "\n", " / "))
			}
			fmt.Println()
		}
	}
}

func printAcademicList(result app.AcademicListResult) {
	fmt.Printf("%s 학사일정\n", result.Year)
	if len(result.Events) == 0 {
		fmt.Println("학사일정이 없습니다")
		return
	}

	currentMonth := ""
	for _, event := range result.Events {
		if event.Month != currentMonth {
			if currentMonth != "" {
				fmt.Println()
			}
			currentMonth = event.Month
			fmt.Println(currentMonth)
		}
		fmt.Printf("  %s | %s", event.Date, event.Title)
		if event.Note != "" {
			fmt.Printf(" | %s", event.Note)
		}
		fmt.Println()
	}
	if result.SourceURL != "" {
		fmt.Printf("\n출처: %s\n", linkifyForTerminal(result.SourceURL))
	}
}

func formatSyllabusTimes(times []klas.SyllabusTime) string {
	parts := make([]string, 0, len(times))
	for _, item := range times {
		label := item.Weekday
		if len(item.Periods) > 0 {
			periodLabels := make([]string, 0, len(item.Periods))
			for _, period := range item.Periods {
				periodLabels = append(periodLabels, strconv.Itoa(period))
			}
			label += " " + strings.Join(periodLabels, ",") + "교시"
		}
		if item.Room != "" {
			label += " (" + item.Room + ")"
		}
		parts = append(parts, strings.TrimSpace(label))
	}
	return strings.Join(parts, ", ")
}

func formatSyllabusEvaluation(evaluation klas.SyllabusEvaluation) string {
	parts := []string{
		"출석 " + strconv.Itoa(evaluation.Attendance),
		"학습 " + strconv.Itoa(evaluation.Learning),
		"중간 " + strconv.Itoa(evaluation.Midterm),
		"기말 " + strconv.Itoa(evaluation.Final),
		"과제 " + strconv.Itoa(evaluation.Report),
		"퀴즈 " + strconv.Itoa(evaluation.Quiz),
		"기타 " + strconv.Itoa(evaluation.Other),
	}
	return strings.Join(parts, " / ")
}

func printLectureRows(rows []app.LectureRow) {
	if len(rows) == 0 {
		fmt.Println("온라인 강의가 없습니다")
		return
	}

	for _, row := range rows {
		id := row.ID
		status := "다운로드 가능"
		if row.Lecture.ContentID == "" {
			status = "학습활동"
			if row.Lecture.LearningSeq == "" {
				id = "-"
				status = "다운로드 불가"
			}
		}
		fmt.Printf("%s | %s | %s | %s | %s | %s | %s\n",
			id,
			formatLectureRange(row.Lecture.StartAt, row.Lecture.EndAt),
			emptyFallback(row.Lecture.Progress, "진도 확인 필요"),
			status,
			row.CourseName,
			emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요"),
			row.Lecture.Title,
		)
	}
}

func printLectureStatusRows(rows []app.LectureRow) {
	if len(rows) == 0 {
		fmt.Println("온라인 강의가 없습니다")
		return
	}

	for _, row := range rows {
		percent := lectureStatusPercent(row.Lecture)
		fmt.Printf("%s %s | %s | %s | %s | %s\n",
			renderProgressBar(percent, 28),
			formatLectureStatusMinutes(row.Lecture),
			lectureStatusLabel(row.Lecture, percent),
			row.CourseName,
			emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요"),
			row.Lecture.Title,
		)
	}
}

func printLectureDownloadAllResult(result app.LectureDownloadAllResult) {
	if len(result.Items) == 0 {
		fmt.Println("다운로드할 온라인 강의가 없습니다")
		return
	}

	downloaded := 0
	skipped := 0
	failed := 0
	for _, item := range result.Items {
		label := item.Lecture.Lecture.Title
		if item.Lecture.Lecture.ModuleTitle != "" {
			label = item.Lecture.Lecture.ModuleTitle + " - " + label
		}

		switch {
		case item.Err != nil && item.Skipped:
			skipped++
			fmt.Printf("건너뜀: %s (%v)\n", label, item.Err)
		case item.Err != nil:
			failed++
			fmt.Printf("실패: %s (%v)\n", label, item.Err)
		case item.Skipped:
			skipped++
			fmt.Printf("건너뜀: %s (이미 있음)\n", item.Path)
		default:
			downloaded++
			fmt.Printf("완료: %s (%s)\n", item.Path, formatBytes(item.Bytes))
		}
	}

	fmt.Printf("전체 다운로드 결과: 완료 %d, 건너뜀 %d, 실패 %d\n", downloaded, skipped, failed)
}

func printLectureAttendAllResult(result app.LectureAttendAllResult) {
	if len(result.Items) == 0 {
		fmt.Println("수강할 온라인 강의가 없습니다")
		return
	}

	completed := 0
	failed := 0
	for _, item := range result.Items {
		if item.Err != nil {
			failed++
			fmt.Printf("실패: %s (%v)\n", item.Lecture.Lecture.Title, item.Err)
			continue
		}
		completed++
		fmt.Printf("완료: %s | %s\n", formatLectureProgress(item.Lecture, item.Progress), item.Lecture.Lecture.Title)
	}
	fmt.Printf("전체 수강 결과: 완료 %d, 실패 %d\n", completed, failed)
}

func printLectureProgress(row app.LectureRow, progress klas.LectureProgress) {
	fmt.Printf("수강중: %s | %s\n", formatLectureProgress(row, progress), row.Lecture.Title)
}

func formatLectureProgress(row app.LectureRow, progress klas.LectureProgress) string {
	return fmt.Sprintf("%s %s | %s",
		renderProgressBar(progress.Progress, 28),
		formatProgressMinutes(progress),
		row.ID,
	)
}

func renderProgressBar(percent float64, width int) string {
	if width <= 0 {
		width = 28
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	bar := bubblesprogress.New(
		bubblesprogress.WithWidth(width),
		bubblesprogress.WithFillCharacters('█', '░'),
	)
	return bar.ViewAs(percent / 100)
}

func formatProgressMinutes(progress klas.LectureProgress) string {
	total := emptyFallback(progress.TotalTime, "?")
	required := emptyFallback(progress.PTime, "?")
	return total + "/" + required + "분"
}

func lectureStatusPercent(lecture klas.Lecture) float64 {
	if lecture.ContentID != "" {
		return boundedPercent(parseFloatOrDefault(lecture.Progress, 0))
	}

	achieved := parseFloatOrDefault(lecture.AchievedTime, 0)
	required := parseFloatOrDefault(lecture.RequiredTime, 0)
	if required <= 0 {
		return 0
	}
	return boundedPercent(achieved / required * 100)
}

func lectureStatusLabel(lecture klas.Lecture, percent float64) string {
	if percent >= 100 {
		return "완료"
	}
	now := time.Now()
	if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
		return "예정"
	}
	if lecture.EndAt != nil && now.After(*lecture.EndAt) {
		return "기간 종료"
	}
	return "미완료"
}

func formatLectureStatusMinutes(lecture klas.Lecture) string {
	if lecture.ContentID != "" {
		return emptyFallback(lecture.AchievedTime, "0") + "/" + emptyFallback(lecture.RequiredTime, "?") + "분"
	}
	return emptyFallback(lecture.AchievedTime, "0") + "/" + emptyFallback(lecture.RequiredTime, "?") + "분"
}

func parseFloatOrDefault(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func boundedPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func weekdayLabel(weekday int) string {
	switch weekday {
	case 1:
		return "월"
	case 2:
		return "화"
	case 3:
		return "수"
	case 4:
		return "목"
	case 5:
		return "금"
	case 6:
		return "토"
	default:
		return fmt.Sprintf("%d요일", weekday)
	}
}

func formatPeriod(period int, span int) string {
	if span <= 1 {
		return fmt.Sprintf("%d교시", period)
	}
	return fmt.Sprintf("%d-%d교시", period, period+span-1)
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func formatLectureRange(start *time.Time, end *time.Time) string {
	if start == nil && end == nil {
		return "기간 확인 필요"
	}
	if start == nil {
		return "~ " + end.Format("2006-01-02 15:04")
	}
	if end == nil {
		return start.Format("2006-01-02 15:04") + " ~"
	}
	return start.Format("2006-01-02 15:04") + " ~ " + end.Format("2006-01-02 15:04")
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor := int64(unit)
	unitLabel := "KB"
	for _, label := range []string{"MB", "GB", "TB"} {
		if value < divisor*unit {
			break
		}
		divisor *= unit
		unitLabel = label
	}
	return fmt.Sprintf("%.1f %s", float64(value)/float64(divisor), unitLabel)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "마감 확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}

func formatNoticeTime(value *time.Time) string {
	if value == nil {
		return "작성일 확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}

var urlPattern = regexp.MustCompile(`https?://[^\s<>"']+`)

func linkifyForTerminal(text string) string {
	if !terminalHyperlinksEnabled() {
		return text
	}
	return hyperlinkURLs(text)
}

func openAndPrintURL(url string, err error) error {
	if err != nil {
		return err
	}
	fmt.Printf("URL: %s\n", linkifyForTerminal(url))
	return openBrowser(url)
}

func openBrowser(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return errors.New("열 URL이 없습니다")
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("브라우저 열기 실패: %w", err)
	}
	return nil
}

func terminalHyperlinksEnabled() bool {
	if os.Getenv("KLAP_NO_HYPERLINKS") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("KLAP_FORCE_HYPERLINKS") != "" {
		return true
	}

	stdout, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return stdout.Mode()&os.ModeCharDevice != 0
}

func hyperlinkURLs(text string) string {
	return urlPattern.ReplaceAllStringFunc(text, func(rawURL string) string {
		visibleURL := strings.TrimRight(rawURL, ".,)]}")
		trailing := strings.TrimPrefix(rawURL, visibleURL)
		if visibleURL == "" {
			return rawURL
		}
		return "\x1b]8;;" + visibleURL + "\x1b\\" + visibleURL + "\x1b]8;;\x1b\\" + trailing
	})
}
