package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

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
	case "assignment":
		return runAssignment(ctx, service, args[1:])
	case "notice":
		return runNotice(ctx, service, args[1:])
	case "timetable":
		return runTimetable(ctx, service, args[1:])
	case "lecture":
		return runLecture(ctx, service, args[1:])
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

func runAssignment(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap assignment <list|detail|remind>")
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
	case "remind":
		return runAssignmentRemind(ctx, service, args[1:])
	default:
		return fmt.Errorf("unknown assignment command: %s", args[0])
	}
}

func runNotice(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap notice <list|detail>")
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

func runLecture(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap lecture <list|download>")
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
	case "download":
		if len(args) < 2 {
			return errors.New("usage: klap lecture download <강의ID> [--dir <경로>]")
		}
		dir, err := dirFlag(args[2:])
		if err != nil {
			return err
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
	default:
		return fmt.Errorf("unknown lecture command: %s", args[0])
	}
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

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
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
  klap assignment detail <과제ID> 과제 상세 출력
  klap assignment remind 과제 마감 reminder 동기화
  klap notice list       강의 공지 목록 출력
  klap notice detail <공지ID> 강의 공지 상세 출력
  klap timetable         최신 학기 시간표 출력
  klap lecture list      온라인 강의 목록 출력
  klap lecture download <강의ID> 온라인 강의 다운로드
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

func printLectureRows(rows []app.LectureRow) {
	if len(rows) == 0 {
		fmt.Println("온라인 강의가 없습니다")
		return
	}

	for _, row := range rows {
		id := row.ID
		status := "다운로드 가능"
		if row.Lecture.ContentID == "" {
			id = "-"
			status = "다운로드 불가"
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
