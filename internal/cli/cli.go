package cli

import (
	"context"
	"errors"
	"fmt"
	bubblesprogress "github.com/charmbracelet/bubbles/progress"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (r Runner) runLecture(ctx context.Context, service *app.Service, args []string) error {
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
		r.printLectureRows(rows)
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
		r.printLectureStatusRows(rows)
		return nil
	case "download":
		if len(args) < 2 {
			return errors.New("usage: klap lecture download <status|open|과목명|과목번호|강의ID> [--dir <경로>]")
		}
		if args[1] == "status" {
			dir, err := dirFlag(args[2:])
			if err != nil {
				return err
			}
			result, err := service.DownloadStatus(dir)
			if err != nil {
				return err
			}
			r.printDownloadStatus(result)
			return nil
		}
		if args[1] == "open" {
			dir, err := dirFlag(args[2:])
			if err != nil {
				return err
			}
			result, err := service.DownloadStatus(dir)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(r.Out, "다운로드 폴더: %s\n", result.Dir)
			return r.openExternal(result.Dir, "폴더")
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
			r.printLectureDownloadAllResult(result)
			return nil
		}
		result, err := service.DownloadLecture(ctx, args[1], app.LectureDownloadOptions{
			User: app.UserOption{StudentID: userFlag(args[2:])},
			Dir:  dir,
		})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(r.Out, "다운로드 완료: %s (%s)\n", result.Path, formatBytes(result.Bytes))
		return nil
	case "attend":
		return r.runLectureAttend(ctx, service, args[1:])
	case "open":
		if len(args) != 2 {
			return errors.New("usage: klap lecture open <강의ID>")
		}
		result, err := service.LectureOpenURL(ctx, args[1], app.UserOption{})
		return r.openAndPrintURL(result.URL, err)
	default:
		return fmt.Errorf("unknown lecture command: %s", args[0])
	}
}

func (r Runner) runLectureAttend(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap lecture attend <강의ID|all> [--course <과목명|번호>] [--interval <초>]")
	}

	interval, err := intervalFlag(args[1:])
	if err != nil {
		return err
	}
	progressPrinter := newLectureProgressPrinter(r.Out, r.stdoutSupportsInPlaceProgress())

	if args[0] == "all" {
		course, err := courseFilter(args[1:])
		if err != nil {
			return err
		}
		result, err := service.AttendAllLectures(ctx, app.LectureAttendAllOptions{
			User:         app.UserOption{StudentID: userFlag(args[1:])},
			CourseFilter: course,
			Interval:     interval,
			OnProgress:   progressPrinter.Print,
		})
		progressPrinter.Clear()
		if err != nil {
			return err
		}
		r.printLectureAttendAllResult(result)
		return nil
	}

	result, err := service.AttendLecture(ctx, args[0], app.LectureAttendOptions{
		User:       app.UserOption{StudentID: userFlag(args[1:])},
		Interval:   interval,
		OnProgress: progressPrinter.Print,
	})
	progressPrinter.Clear()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(r.Out, "수강 완료: %s | %s\n", formatLectureProgress(result.Lecture, result.Progress), result.Lecture.Lecture.Title)
	return nil
}

func (r Runner) runAttend(ctx context.Context, service *app.Service, args []string) error {
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
	progressPrinter := newLectureProgressPrinter(r.Out, r.stdoutSupportsInPlaceProgress())
	result, err := service.AttendAllLectures(ctx, app.LectureAttendAllOptions{
		User:         app.UserOption{StudentID: userFlag(flagArgs)},
		CourseFilter: courseFilter,
		Interval:     interval,
		OnProgress:   progressPrinter.Print,
	})
	progressPrinter.Clear()
	if err != nil {
		return err
	}
	r.printLectureAttendAllResult(result)
	return nil
}

func lectureListOptions(args []string) (app.LectureListOptions, error) {
	course, err := courseFilter(args)
	if err != nil {
		return app.LectureListOptions{}, err
	}
	return app.LectureListOptions{
		User:         app.UserOption{StudentID: userFlag(args)},
		CourseFilter: course,
		Refresh:      hasFlag(args, "--refresh"),
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

func truncateForLine(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if max <= 0 || len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func yesNo(value bool) string {
	if value {
		return "Y"
	}
	return "N"
}

func formatCompactDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 && numericString(value) {
		return value[:4] + "-" + value[4:6] + "-" + value[6:]
	}
	return emptyFallback(value, "-")
}

func numericString(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func (r Runner) printLectureRows(rows []app.LectureRow) {
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "온라인 강의가 없습니다")
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
		_, _ = fmt.Fprintf(r.Out, "%s | %s | %s | %s | %s | %s | %s\n",
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

func (r Runner) printLectureStatusRows(rows []app.LectureRow) {
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "온라인 강의가 없습니다")
		return
	}

	for _, row := range rows {
		percent := lectureStatusPercent(row.Lecture)
		_, _ = fmt.Fprintf(r.Out, "%s %s | %s | %s | %s | %s\n",
			renderProgressBar(percent, 28),
			formatLectureStatusMinutes(row.Lecture),
			r.lectureStatusLabel(row.Lecture, percent),
			row.CourseName,
			emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요"),
			row.Lecture.Title,
		)
	}
}

func (r Runner) printLectureDownloadAllResult(result app.LectureDownloadAllResult) {
	if len(result.Items) == 0 {
		_, _ = fmt.Fprintln(r.Out, "다운로드할 온라인 강의가 없습니다")
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
			_, _ = fmt.Fprintf(r.Out, "건너뜀: %s (%v)\n", label, item.Err)
		case item.Err != nil:
			failed++
			_, _ = fmt.Fprintf(r.Out, "실패: %s (%v)\n", label, item.Err)
		case item.Skipped:
			skipped++
			_, _ = fmt.Fprintf(r.Out, "건너뜀: %s (이미 있음)\n", item.Path)
		default:
			downloaded++
			_, _ = fmt.Fprintf(r.Out, "완료: %s (%s)\n", item.Path, formatBytes(item.Bytes))
		}
	}

	_, _ = fmt.Fprintf(r.Out, "전체 다운로드 결과: 완료 %d, 건너뜀 %d, 실패 %d\n", downloaded, skipped, failed)
}

func (r Runner) printDownloadStatus(result app.DownloadStatusResult) {
	_, _ = fmt.Fprintf(r.Out, "다운로드 폴더: %s\n", result.Dir)
	_, _ = fmt.Fprintf(r.Out, "파일 수: %d\n", result.Files)
	_, _ = fmt.Fprintf(r.Out, "크기: %s\n", formatBytes(result.Bytes))
	if result.PartialFiles > 0 {
		_, _ = fmt.Fprintf(r.Out, "부분 파일: %d개 (%s)\n", result.PartialFiles, formatBytes(result.PartialBytes))
	}
	if len(result.Items) == 0 {
		_, _ = fmt.Fprintln(r.Out, "다운로드된 파일이 없습니다")
		return
	}
	_, _ = fmt.Fprintln(r.Out, "\n최근 파일")
	for _, item := range result.Items {
		_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s\n",
			item.ModifiedAt.Format("2006-01-02 15:04"),
			formatBytes(item.Bytes),
			item.Path,
		)
	}
}

func (r Runner) printLectureAttendAllResult(result app.LectureAttendAllResult) {
	if len(result.Items) == 0 {
		_, _ = fmt.Fprintln(r.Out, "수강할 온라인 강의가 없습니다")
		return
	}

	completed := 0
	failed := 0
	for _, item := range result.Items {
		if item.Err != nil {
			failed++
			_, _ = fmt.Fprintf(r.Out, "실패: %s (%v)\n", item.Lecture.Lecture.Title, item.Err)
			continue
		}
		completed++
		_, _ = fmt.Fprintf(r.Out, "완료: %s | %s\n", formatLectureProgress(item.Lecture, item.Progress), item.Lecture.Lecture.Title)
	}
	_, _ = fmt.Fprintf(r.Out, "전체 수강 결과: 완료 %d, 실패 %d\n", completed, failed)
}

type lectureProgressPrinter struct {
	writer io.Writer
	inline bool
	active bool
}

func newLectureProgressPrinter(writer io.Writer, inline bool) *lectureProgressPrinter {
	return &lectureProgressPrinter{writer: writer, inline: inline}
}

func (p *lectureProgressPrinter) Print(row app.LectureRow, progress app.LectureProgress) {
	line := fmt.Sprintf("수강중: %s | %s", formatLectureProgress(row, progress), row.Lecture.Title)
	if !p.inline {
		_, _ = fmt.Fprintln(p.writer, line)
		return
	}
	_, _ = fmt.Fprintf(p.writer, "\r\x1b[2K%s", line)
	p.active = true
}

func (p *lectureProgressPrinter) Clear() {
	if !p.inline || !p.active {
		return
	}
	_, _ = fmt.Fprint(p.writer, "\r\x1b[2K")
	p.active = false
}

func formatLectureProgress(row app.LectureRow, progress app.LectureProgress) string {
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

func formatProgressMinutes(progress app.LectureProgress) string {
	total := emptyFallback(progress.TotalTime, "?")
	required := emptyFallback(progress.PTime, "?")
	return total + "/" + required + "분"
}

func lectureStatusPercent(lecture app.Lecture) float64 {
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

func formatLectureStatusMinutes(lecture app.Lecture) string {
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

var urlPattern = regexp.MustCompile(`https?://[^\s<>"']+`)

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
