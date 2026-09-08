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

func (r Runner) runSearch(ctx context.Context, service *app.Service, args []string) error {
	opts, err := searchOptions(args)
	if err != nil {
		return err
	}
	result, err := service.Search(ctx, opts)
	if err != nil {
		return err
	}
	r.printSearch(result)
	return nil
}

func (r Runner) runDue(ctx context.Context, service *app.Service, args []string) error {
	opts, err := dueOptions(args)
	if err != nil {
		return err
	}
	result, err := service.Due(ctx, opts)
	if err != nil {
		return err
	}
	r.printDue(result)
	return nil
}

func (r Runner) runCache(ctx context.Context, service *app.Service, args []string) error {
	_ = ctx
	if len(args) == 0 {
		return errors.New("usage: klap cache <status|clear>")
	}
	switch args[0] {
	case "status":
		result, err := service.CacheStatus()
		if err != nil {
			return err
		}
		r.printCacheStatus(result)
		return nil
	case "clear":
		scope := ""
		if len(args) > 1 {
			scope = args[1]
		}
		result, err := service.ClearCacheScope(scope)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(r.Out, "캐시 삭제 완료: %d개\n", result.Removed)
		return nil
	default:
		return fmt.Errorf("unknown cache command: %s", args[0])
	}
}

func (r Runner) runCourse(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap course list [--user <학번>]")
	}

	switch args[0] {
	case "list":
		terms, err := service.CourseList(ctx, app.CourseListOptions{
			User:    app.UserOption{StudentID: userFlag(args[1:])},
			Refresh: hasFlag(args[1:], "--refresh"),
		})
		if err != nil {
			return err
		}
		r.printCourseList(terms)
		return nil
	default:
		return fmt.Errorf("unknown course command: %s", args[0])
	}
}

func (r Runner) runSubject(ctx context.Context, service *app.Service, args []string) error {
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
		r.printSubjectSearch(result)
		return nil
	default:
		return fmt.Errorf("unknown subject command: %s", args[0])
	}
}

func (r Runner) runTerm(ctx context.Context, service *app.Service, args []string) error {
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
		r.printTermRows(rows)
		return nil
	case "select":
		if len(args) < 2 {
			return errors.New("usage: klap term select <학기번호|학기값> [--user <학번>]")
		}
		settings, err := service.SelectTerm(ctx, args[1], app.UserOption{StudentID: userFlag(args[2:])})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(r.Out, "현재 학기: %s (%s)\n", settings.Label, settings.Value)
		return nil
	default:
		return fmt.Errorf("unknown term command: %s", args[0])
	}
}

func (r Runner) runTimetable(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown timetable command: %s", args[0])
	}

	result, err := service.Timetable(ctx, app.TimetableOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Refresh: hasFlag(args, "--refresh"),
	})
	if err != nil {
		return err
	}
	r.printTimetable(result)
	return nil
}

func (r Runner) runAttendance(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "cdp" {
		result, err := service.CdpAttendance(ctx, app.CdpAttendanceOptions{
			User: app.UserOption{StudentID: userFlag(args[1:])},
		})
		if err != nil {
			return err
		}
		r.printCdpAttendance(result)
		return nil
	}
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
		r.printAttendanceDetail(result)
		return nil
	}
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown attendance command: %s", args[0])
	}

	result, err := service.AttendanceList(ctx, app.AttendanceListOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Refresh: hasFlag(args, "--refresh"),
	})
	if err != nil {
		return err
	}
	r.printAttendanceList(result)
	return nil
}

func (r Runner) runGrade(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	}

	opts, err := gradeOptions(args)
	if err != nil {
		return err
	}
	result, err := service.Grade(ctx, opts)
	if err != nil {
		return err
	}
	r.printGrade(result)
	return nil
}

func (r Runner) runRank(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	}

	opts, err := rankOptions(args)
	if err != nil {
		return err
	}
	result, err := service.Rank(ctx, opts)
	if err != nil {
		return err
	}
	r.printRank(result)
	return nil
}

func (r Runner) runEvaluation(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap evaluation <list|submit>")
	}

	switch args[0] {
	case "list":
		opts := app.EvaluationListOptions{
			User:    app.UserOption{StudentID: userFlag(args[1:])},
			Refresh: hasFlag(args[1:], "--refresh"),
		}
		result, err := service.EvaluationList(ctx, opts)
		if err != nil {
			return err
		}
		r.printEvaluationList(result)
		return nil
	case "submit":
		opts, err := evaluationSubmitOptions(args[1:])
		if err != nil {
			return err
		}
		result, err := service.EvaluationSubmit(ctx, opts)
		if err != nil {
			return err
		}
		r.printEvaluationSubmitResult(result)
		return nil
	default:
		return fmt.Errorf("unknown evaluation command: %s", args[0])
	}
}

func (r Runner) runSyllabus(ctx context.Context, service *app.Service, args []string) error {
	opts, err := syllabusOptions(args)
	if err != nil {
		return err
	}
	result, err := service.Syllabus(ctx, opts)
	if err != nil {
		return err
	}
	r.printSyllabus(result)
	return nil
}

func (r Runner) runRoom(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap room <free|busy|index|cache>")
	}
	switch args[0] {
	case "free":
		opts, err := roomQueryOptions(args[1:])
		if err != nil {
			return err
		}
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomFree(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomFree(result)
		return nil
	case "busy":
		opts, err := roomQueryOptions(args[1:])
		if err != nil {
			return err
		}
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomBusy(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomBusy(result)
		return nil
	case "available", "empty":
		opts, err := roomAvailableOptions(args[1:])
		if err != nil {
			return err
		}
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomAvailable(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomAvailable(result)
		return nil
	case "index":
		opts, err := roomIndexOptions(args[1:])
		if err != nil {
			return err
		}
		opts.Refresh = true
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomIndex(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomIndex(result)
		return nil
	case "cache":
		if len(args) == 2 && args[1] == "clear" {
			result, err := service.ClearRoomCache()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(r.Out, "room-index cache 삭제: %d개\n", result.Removed)
			return nil
		}
		return errors.New("usage: klap room cache clear")
	default:
		return fmt.Errorf("unknown room command: %s", args[0])
	}
}

func roomQueryOptions(args []string) (app.RoomQueryOptions, error) {
	opts := app.RoomQueryOptions{User: app.UserOption{StudentID: userFlag(args)}}
	roomParts := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.RoomQueryOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--building":
			if i+1 >= len(args) {
				return app.RoomQueryOptions{}, errors.New("--building에는 건물명이 필요합니다")
			}
			opts.Building = args[i+1]
			i++
		case "--refresh":
			opts.Refresh = true
		case "--user":
			if i+1 >= len(args) {
				return app.RoomQueryOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.RoomQueryOptions{}, fmt.Errorf("unknown room option: %s", args[i])
			}
			roomParts = append(roomParts, args[i])
		}
	}
	opts.Room = strings.Join(roomParts, " ")
	if strings.TrimSpace(opts.Room) == "" {
		return app.RoomQueryOptions{}, errors.New("usage: klap room <free|busy> <강의실명> [--term YYYY-S] [--building <건물명>] [--refresh]")
	}
	return opts, nil
}

func roomAvailableOptions(args []string) (app.RoomAvailableOptions, error) {
	opts := app.RoomAvailableOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--day":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--day에는 요일이 필요합니다")
			}
			opts.Day = args[i+1]
			i++
		case "--duration":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--duration에는 1-3 또는 5 같은 교시 범위가 필요합니다")
			}
			opts.Duration = args[i+1]
			i++
		case "--term":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--building":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--building에는 건물명이 필요합니다")
			}
			opts.Building = args[i+1]
			i++
		case "--refresh":
			opts.Refresh = true
		case "--user":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			return app.RoomAvailableOptions{}, fmt.Errorf("unknown room available option: %s", args[i])
		}
	}
	if strings.TrimSpace(opts.Day) == "" || strings.TrimSpace(opts.Duration) == "" {
		return app.RoomAvailableOptions{}, errors.New("usage: klap room available --day <요일> --duration <교시범위> [--term YYYY-S] [--building <건물명>] [--refresh]")
	}
	return opts, nil
}

func roomIndexOptions(args []string) (app.RoomIndexOptions, error) {
	opts := app.RoomIndexOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.RoomIndexOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--building":
			if i+1 >= len(args) {
				return app.RoomIndexOptions{}, errors.New("--building에는 건물명이 필요합니다")
			}
			opts.Building = args[i+1]
			i++
		case "--refresh":
			opts.Refresh = true
		case "--user":
			if i+1 >= len(args) {
				return app.RoomIndexOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			return app.RoomIndexOptions{}, fmt.Errorf("unknown room index option: %s", args[i])
		}
	}
	return opts, nil
}

func (r Runner) runAcademic(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap academic <list>")
	}

	switch args[0] {
	case "list":
		year, err := academicYearFlag(args[1:])
		if err != nil {
			return err
		}
		result, err := service.AcademicList(ctx, app.AcademicListOptions{
			Year:    year,
			Refresh: hasFlag(args[1:], "--refresh"),
		})
		if err != nil {
			return err
		}
		r.printAcademicList(result)
		return nil
	default:
		return fmt.Errorf("unknown academic command: %s", args[0])
	}
}

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

func gradeOptions(args []string) (app.GradeOptions, error) {
	opts := app.GradeOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.GradeOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			if opts.TermValue != "" {
				return app.GradeOptions{}, errors.New("학기는 하나만 지정할 수 있습니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--user":
			if i+1 >= len(args) {
				return app.GradeOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		case "--refresh":
			opts.Refresh = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.GradeOptions{}, fmt.Errorf("unknown grade option: %s", args[i])
			}
			if opts.TermValue != "" {
				return app.GradeOptions{}, errors.New("학기는 하나만 지정할 수 있습니다")
			}
			opts.TermValue = args[i]
		}
	}
	return opts, nil
}

func rankOptions(args []string) (app.RankOptions, error) {
	opts := app.RankOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.RankOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			if opts.TermValue != "" {
				return app.RankOptions{}, errors.New("학기는 하나만 지정할 수 있습니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--user":
			if i+1 >= len(args) {
				return app.RankOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		case "--refresh":
			opts.Refresh = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.RankOptions{}, fmt.Errorf("unknown rank option: %s", args[i])
			}
			if opts.TermValue != "" {
				return app.RankOptions{}, errors.New("학기는 하나만 지정할 수 있습니다")
			}
			opts.TermValue = args[i]
		}
	}
	return opts, nil
}

func evaluationSubmitOptions(args []string) (app.EvaluationSubmitOptions, error) {
	opts := app.EvaluationSubmitOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--yes":
			opts.Confirm = true
		case "--include-engineering":
			opts.IncludeEngineering = true
		case "--user":
			if i+1 >= len(args) {
				return app.EvaluationSubmitOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.EvaluationSubmitOptions{}, fmt.Errorf("unknown evaluation option: %s", args[i])
			}
			if opts.Selector != "" {
				return app.EvaluationSubmitOptions{}, errors.New("수업평가 제출 대상은 하나만 지정할 수 있습니다")
			}
			opts.Selector = args[i]
		}
	}
	if strings.TrimSpace(opts.Selector) == "" {
		return app.EvaluationSubmitOptions{}, errors.New("usage: klap evaluation submit <all|과목명|번호> [--yes] [--include-engineering] [--user <학번>]")
	}
	return opts, nil
}

func searchOptions(args []string) (app.SearchOptions, error) {
	opts := app.SearchOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Refresh: hasFlag(args, "--refresh"),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return app.SearchOptions{}, errors.New("--type에는 검색 타입이 필요합니다")
			}
			opts.Type = args[i+1]
			i++
		case "--user":
			if i+1 >= len(args) {
				return app.SearchOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		case "--refresh":
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.SearchOptions{}, fmt.Errorf("unknown search option: %s", args[i])
			}
			if opts.Query != "" {
				return app.SearchOptions{}, errors.New("검색어는 하나만 지정할 수 있습니다")
			}
			opts.Query = args[i]
		}
	}
	if strings.TrimSpace(opts.Query) == "" {
		return app.SearchOptions{}, errors.New("usage: klap search <키워드> [--type course|assignment|notice|lecture|academic] [--refresh]")
	}
	return opts, nil
}

func dueOptions(args []string) (app.DueOptions, error) {
	opts := app.DueOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Days:    14,
		Refresh: hasFlag(args, "--refresh"),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--week":
			opts.Days = 7
		case "--days":
			if i+1 >= len(args) {
				return app.DueOptions{}, errors.New("--days에는 일수가 필요합니다")
			}
			days, err := strconv.Atoi(args[i+1])
			if err != nil || days <= 0 {
				return app.DueOptions{}, errors.New("--days에는 1 이상의 숫자가 필요합니다")
			}
			opts.Days = days
			i++
		case "--user":
			if i+1 >= len(args) {
				return app.DueOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		case "--refresh":
		default:
			return app.DueOptions{}, fmt.Errorf("unknown due option: %s", args[i])
		}
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

func (r Runner) printRoomFree(result app.RoomQueryResult) {
	if !r.printRoomQueryHeader(result) {
		return
	}
	_, _ = fmt.Fprintf(r.Out, "%s 빈 시간 | %s%s\n", result.Room, displayRoomTermValue(result.TermValue), roomCacheLabel(result.Cached))
	byWeekday := make(map[int][]int)
	for _, slot := range result.FreeSlots {
		byWeekday[slot.Weekday] = append(byWeekday[slot.Weekday], slot.Period)
	}
	for weekday := 1; weekday <= 5; weekday++ {
		periods := byWeekday[weekday]
		if len(periods) == 0 {
			_, _ = fmt.Fprintf(r.Out, "%s: 없음\n", app.RoomWeekdayLabel(weekday))
			continue
		}
		values := make([]string, 0, len(periods))
		for _, period := range periods {
			values = append(values, strconv.Itoa(period))
		}
		_, _ = fmt.Fprintf(r.Out, "%s: %s교시\n", app.RoomWeekdayLabel(weekday), strings.Join(values, ", "))
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomBusy(result app.RoomQueryResult) {
	if !r.printRoomQueryHeader(result) {
		return
	}
	_, _ = fmt.Fprintf(r.Out, "%s 사용 목록 | %s%s\n", result.Room, displayRoomTermValue(result.TermValue), roomCacheLabel(result.Cached))
	if len(result.BusyRows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "사용 중인 수업이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return
	}
	for _, row := range result.BusyRows {
		course := emptyFallback(row.CourseName, row.CourseCode)
		professor := emptyFallback(row.Professor, "교수 미지정")
		_, _ = fmt.Fprintf(r.Out, "%s %s | %s | %s | %s\n", app.RoomWeekdayLabel(row.Weekday), roomPeriodLabel(row.Period, row.Span), course, professor, row.CourseCode)
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomAvailable(result app.RoomAvailableResult) {
	status := roomAvailabilityStatus(result.Weekday, result.Periods)
	_, _ = fmt.Fprintf(r.Out, "빈 강의실 | %s | %s%s\n", displayRoomTermValue(result.TermValue), status, roomCacheLabel(result.Cached))
	if len(result.Rooms) == 0 {
		_, _ = fmt.Fprintln(r.Out, "조건에 맞는 빈 강의실이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return
	}
	for index, room := range result.Rooms {
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %s\n", index+1, room.Room, status)
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomIndex(result app.RoomIndexResult) {
	_, _ = fmt.Fprintf(r.Out, "강의실 인덱스 | %s | %d개%s\n", displayRoomTermValue(result.TermValue), len(result.Rooms), roomCacheLabel(result.Cached))
	if len(result.Rooms) == 0 {
		_, _ = fmt.Fprintln(r.Out, "조회된 강의실이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return
	}
	for index, room := range result.Rooms {
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %d개 교시 사용\n", index+1, room.Room, room.BusyCount)
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomQueryHeader(result app.RoomQueryResult) bool {
	if result.Room != "" {
		return true
	}
	if len(result.Candidates) == 0 {
		_, _ = fmt.Fprintln(r.Out, "매칭되는 강의실이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return false
	}
	_, _ = fmt.Fprintln(r.Out, "여러 강의실이 매칭되었습니다. 정확한 강의실명으로 다시 조회하세요.")
	for index, room := range result.Candidates {
		_, _ = fmt.Fprintf(r.Out, "%d. %s\n", index+1, room)
	}
	r.printRoomWarnings(result.Warnings)
	return false
}

func (r Runner) printRoomWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	_, _ = fmt.Fprintf(r.Out, "\n경고: %d개 과목의 강의시간 조회에 실패했습니다\n", len(warnings))
	limit := len(warnings)
	if limit > 20 {
		limit = 20
	}
	for _, warning := range warnings[:limit] {
		_, _ = fmt.Fprintf(r.Out, "- %s\n", warning)
	}
	if len(warnings) > limit {
		_, _ = fmt.Fprintf(r.Out, "- ... %d개 생략\n", len(warnings)-limit)
	}
}

func displayRoomTermValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "학기 미지정"
	}
	return strings.ReplaceAll(value, ",", "-")
}

func roomCacheLabel(cached bool) string {
	if cached {
		return " | cache"
	}
	return ""
}

func roomAvailabilityStatus(weekday int, periods []int) string {
	return app.RoomWeekdayLabel(weekday) + " " + roomPeriodsLabel(periods) + " 비어있음"
}

func roomPeriodLabel(period int, span int) string {
	if span <= 1 {
		return strconv.Itoa(period) + "교시"
	}
	return strconv.Itoa(period) + "-" + strconv.Itoa(period+span-1) + "교시"
}

func roomPeriodsLabel(periods []int) string {
	if len(periods) == 0 {
		return "교시 미지정"
	}
	start := periods[0]
	end := periods[len(periods)-1]
	if start == end {
		return strconv.Itoa(start) + "교시"
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end) + "교시"
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

func (r Runner) printCacheStatus(result app.CacheStatusResult) {
	_, _ = fmt.Fprintf(r.Out, "캐시 경로: %s\n", emptyFallback(result.Dir, "-"))
	_, _ = fmt.Fprintf(r.Out, "파일 수: %d\n", result.Files)
	_, _ = fmt.Fprintf(r.Out, "크기: %s\n", formatBytes(result.Bytes))
}

func (r Runner) printSearch(result app.SearchResult) {
	_, _ = fmt.Fprintf(r.Out, "검색: %s", result.Query)
	if result.Type != "" {
		_, _ = fmt.Fprintf(r.Out, " (%s)", result.Type)
	}
	_, _ = fmt.Fprintln(r.Out)

	total := len(result.Courses) + len(result.Assignments) + len(result.Notices) + len(result.Lectures) + len(result.Academics)
	if total == 0 {
		_, _ = fmt.Fprintln(r.Out, "검색 결과가 없습니다")
	}

	if len(result.Courses) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n과목")
		for _, row := range result.Courses {
			_, _ = fmt.Fprintf(r.Out, "  %d. %s | %s (%s)\n",
				row.Index,
				emptyFallback(row.Course.Name, "-"),
				row.Term.Label,
				row.Term.Value,
			)
		}
	}

	if len(result.Assignments) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n과제")
		for _, row := range result.Assignments {
			status := "미제출"
			if row.Assignment.Submitted {
				status = "제출"
			}
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s | %s\n",
				row.ID,
				formatTime(row.Assignment.DueAt),
				status,
				row.CourseName,
				row.Assignment.Title,
			)
		}
	}

	if len(result.Notices) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n공지")
		for _, row := range result.Notices {
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
				row.ID,
				formatNoticeTime(row.Notice.Registered),
				row.CourseName,
				row.Notice.Title,
			)
		}
	}

	if len(result.Lectures) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n온라인 강의")
		for _, row := range result.Lectures {
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s | %s\n",
				row.ID,
				formatLectureRange(row.Lecture.StartAt, row.Lecture.EndAt),
				row.CourseName,
				emptyFallback(row.Lecture.ModuleTitle, "주차 확인 필요"),
				row.Lecture.Title,
			)
		}
	}

	if len(result.Academics) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n학사일정")
		for _, event := range result.Academics {
			_, _ = fmt.Fprintf(r.Out, "  %s %s | %s", event.Month, event.Date, event.Title)
			if event.Note != "" {
				_, _ = fmt.Fprintf(r.Out, " | %s", event.Note)
			}
			_, _ = fmt.Fprintln(r.Out)
		}
	}

	if len(result.Errors) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n검색 실패")
		for _, sectionError := range result.Errors {
			_, _ = fmt.Fprintf(r.Out, "  %s: %v\n", sectionError.Section, sectionError.Err)
		}
	}
}

func (r Runner) printDue(result app.DueResult) {
	_, _ = fmt.Fprintf(r.Out, "데드라인: %s ~ %s\n",
		result.From.Format("2006-01-02"),
		result.Until.Format("2006-01-02"),
	)
	if len(result.Items) == 0 {
		_, _ = fmt.Fprintln(r.Out, "예정된 데드라인이 없습니다")
	}
	for _, item := range result.Items {
		course := ""
		if item.CourseName != "" {
			course = " | " + item.CourseName
		}
		status := ""
		if item.Status != "" {
			status = " | " + item.Status
		}
		id := ""
		if item.ID != "" {
			id = " | " + item.ID
		}
		_, _ = fmt.Fprintf(r.Out, "%s | %s%s | %s%s%s\n",
			item.DueAt.Format("2006-01-02 15:04"),
			item.Kind,
			id,
			item.Title,
			course,
			status,
		)
	}
	if len(result.Errors) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n확인 실패")
		for _, sectionError := range result.Errors {
			_, _ = fmt.Fprintf(r.Out, "  %s: %v\n", sectionError.Section, sectionError.Err)
		}
	}
}

func (r Runner) printCourseList(terms []app.Term) {
	if len(terms) == 0 {
		_, _ = fmt.Fprintln(r.Out, "수강 학기가 없습니다")
		return
	}

	term := terms[0]
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", term.Label, term.Value)
	if len(term.Courses) == 0 {
		_, _ = fmt.Fprintln(r.Out, "수업이 없습니다")
		return
	}

	for index, course := range term.Courses {
		_, _ = fmt.Fprintf(r.Out, "%d. %s\n", index+1, strings.TrimSpace(course.Name))
	}
}

func (r Runner) printSubjectSearch(result app.SubjectSearchResult) {
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "검색된 과목이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "학정번호 | 과목명 | 교수 | 강의시간")
	for _, row := range result.Rows {
		times := formatSyllabusTimes(row.Times)
		if times == "" {
			times = "강의시간 확인 필요"
		}
		if row.Err != nil {
			times = "강의시간 확인 실패"
		}
		_, _ = fmt.Fprintf(r.Out, "%s | %s | %s | %s\n",
			emptyFallback(row.CourseCode, "-"),
			emptyFallback(row.Name, "-"),
			emptyFallback(row.Professor, "-"),
			times,
		)
	}
}

func (r Runner) printTermRows(rows []app.TermRow) {
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "수강 학기가 없습니다")
		return
	}

	for _, row := range rows {
		prefix := " "
		if row.Current {
			prefix = "*"
		}
		_, _ = fmt.Fprintf(r.Out, "%s %d. %s (%s)\n", prefix, row.Index, row.Term.Label, row.Term.Value)
	}
}

func (r Runner) printTimetable(result app.TimetableResult) {
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Entries) == 0 {
		_, _ = fmt.Fprintln(r.Out, "시간표가 없습니다")
		return
	}

	printedOnlineHeader := false
	currentWeekday := 0
	for _, entry := range result.Entries {
		if entry.Online {
			if !printedOnlineHeader {
				_, _ = fmt.Fprintln(r.Out, "\n온라인/미지정")
				printedOnlineHeader = true
			}
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
				formatPeriod(entry.Period, entry.Span),
				entry.SubjectName,
				emptyFallback(entry.Room, "강의실 미지정"),
				emptyFallback(entry.Professor, "교수 미지정"),
			)
			continue
		}

		if entry.Weekday != currentWeekday {
			if currentWeekday != 0 {
				_, _ = fmt.Fprintln(r.Out)
			}
			currentWeekday = entry.Weekday
			_, _ = fmt.Fprintln(r.Out, weekdayLabel(entry.Weekday))
		}
		_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
			formatPeriod(entry.Period, entry.Span),
			entry.SubjectName,
			emptyFallback(entry.Room, "강의실 미지정"),
			emptyFallback(entry.Professor, "교수 미지정"),
		)
	}
}

func (r Runner) printAttendanceList(result app.AttendanceListResult) {
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "출석 현황이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "번호 | 학정번호 | 과목 | 교수 | 강의시간 | 출석 요약")
	for _, row := range result.Rows {
		course := row.Course
		summary := attendanceSummary(row)
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %s | %s | %s | %s\n",
			row.Index,
			emptyFallback(course.CourseCode, "-"),
			emptyFallback(course.Name, "-"),
			emptyFallback(course.Professor, "-"),
			emptyFallback(course.Weekday, "확인 필요"),
			summary,
		)
	}
}

func (r Runner) printAttendanceDetail(result app.AttendanceDetailResult) {
	row := result.Row
	course := row.Course
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	_, _ = fmt.Fprintf(r.Out, "과목: %s\n", course.Name)
	_, _ = fmt.Fprintf(r.Out, "학정번호: %s\n", emptyFallback(course.CourseCode, "-"))
	_, _ = fmt.Fprintf(r.Out, "교수: %s\n", emptyFallback(course.Professor, "-"))
	_, _ = fmt.Fprintf(r.Out, "강의시간: %s\n", emptyFallback(course.Weekday, "확인 필요"))
	_, _ = fmt.Fprintf(r.Out, "출석 요약: %s\n", attendanceSummary(row))
	if row.Err != nil {
		_, _ = fmt.Fprintf(r.Out, "상세 오류: %v\n", row.Err)
		return
	}
	if len(row.Sessions) == 0 {
		_, _ = fmt.Fprintln(r.Out, "상세 출석 내역이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "\n주차별 출석")
	for _, session := range row.Sessions {
		parts := make([]string, 0, len(session.Slots))
		for _, slot := range session.Slots {
			parts = append(parts, fmt.Sprintf("%d차시 %s %s", slot.Index, attendanceMarkLabel(slot.Mark), formatAttendanceDate(slot.Date)))
		}
		_, _ = fmt.Fprintf(r.Out, "%s주차 | %s\n", emptyFallback(session.Week, "-"), strings.Join(parts, " / "))
	}
}

func (r Runner) printCdpAttendance(result app.CdpAttendanceResult) {
	report := result.Report
	_, _ = fmt.Fprintln(r.Out, "CDP 출석내역")
	_, _ = fmt.Fprintf(r.Out, "총 출석: %s회\n", emptyFallback(report.TotalCount, "0"))
	if len(report.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "CDP 출석내역이 없습니다")
		_, _ = fmt.Fprintln(r.Out, "* 출석내역은 출석 후 약 일주일 후에 반영됩니다.")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "날짜 | 회차 | 강의주제 | 강사명")
	for _, row := range report.Rows {
		_, _ = fmt.Fprintf(r.Out, "%s | %s | %s | %s\n",
			emptyFallback(formatCdpDate(row.Date), "-"),
			emptyFallback(row.Seq, "-"),
			emptyFallback(row.Title, "-"),
			emptyFallback(row.Speaker, "-"),
		)
	}
	_, _ = fmt.Fprintln(r.Out, "* 출석내역은 출석 후 약 일주일 후에 반영됩니다.")
}

func (r Runner) printGrade(result app.GradeResult) {
	report := result.Report
	summary := report.Summary
	_, _ = fmt.Fprintln(r.Out, "성적")
	if strings.TrimSpace(result.TermValue) == "" {
		_, _ = fmt.Fprintf(r.Out, "신청 학점: 전체 %d / 전공 %d / 교양 %d / 기타 %d\n",
			summary.AppliedCredits,
			summary.MajorAppliedCredits,
			summary.CultureAppliedCredits,
			summary.EtcAppliedCredits,
		)
		_, _ = fmt.Fprintf(r.Out, "취득 학점: 전체 %d / 전공 %d / 교양 %d / 기타 %d\n",
			summary.EarnedCredits,
			summary.MajorEarnedCredits,
			summary.CultureEarnedCredits,
			summary.EtcEarnedCredits,
		)
		_, _ = fmt.Fprintf(r.Out, "평점: 학적부 기준 %s / 성적증명서 기준 %s\n",
			emptyFallback(summary.GPA, "-"),
			emptyFallback(summary.RetakeGPA, "-"),
		)
		if summary.DeletedCredits > 0 {
			_, _ = fmt.Fprintf(r.Out, "삭제 학점: %d\n", summary.DeletedCredits)
		}
	}

	if len(report.Terms) == 0 {
		_, _ = fmt.Fprintln(r.Out, "\n성적 내역이 없습니다")
		return
	}

	showSummary := strings.TrimSpace(result.TermValue) == ""
	for index, term := range report.Terms {
		if showSummary || index > 0 {
			_, _ = fmt.Fprintln(r.Out)
		}
		_, _ = fmt.Fprintf(r.Out, "%s\n", emptyFallback(term.Label, "-"))
		_, _ = fmt.Fprintln(r.Out, "과목 | 이수구분 | 학점 | 성적 | 재수강 | 학정번호")
		for _, course := range term.Courses {
			_, _ = fmt.Fprintf(r.Out, "%s | %s | %d | %s | %s | %s\n",
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

func gradeLabel(course app.GradeCourse) string {
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

func (r Runner) printRank(result app.RankResult) {
	_, _ = fmt.Fprintln(r.Out, "석차")
	if len(result.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "석차 내역이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "학기 | 신청학점 | 총점 | 평점 | 백분율 | 학과석차 | 학사경고")
	for _, row := range result.Rows {
		_, _ = fmt.Fprintf(r.Out, "%s | %s | %s | %s | %s | %s | %s\n",
			emptyFallback(row.TermLabel, row.TermValue),
			emptyFallback(row.AppliedCredits, "-"),
			emptyFallback(row.TotalScore, "-"),
			emptyFallback(row.GPA, "-"),
			emptyFallback(row.Percentile, "-"),
			formatClassRank(row),
			emptyFallback(row.Warning, "-"),
		)
	}
}

func (r Runner) printEvaluationList(result app.EvaluationListResult) {
	_, _ = fmt.Fprintln(r.Out, "수업평가")
	if !result.Term.TermEnabled {
		_, _ = fmt.Fprintln(r.Out, "수업평가 기간이 아닙니다")
		return
	}
	_, _ = fmt.Fprintf(r.Out, "%s (%s) | %s | %s ~ %s\n",
		emptyFallback(result.Term.Label, result.Term.Value),
		emptyFallback(result.Term.Value, "-"),
		emptyFallback(result.Term.JudgeName, result.Term.JudgeChasu),
		formatCompactDate(result.Term.FromDate),
		formatCompactDate(result.Term.ToDate),
	)
	if len(result.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "수업평가 과목이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "번호 | 상태 | 공학인증 | 과목 | 교수 | 이수/학점")
	for _, row := range result.Rows {
		course := row.Course
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %s | %s | %s | %s\n",
			row.Index,
			evaluationStatusLabel(course),
			yesNo(course.Engineering),
			emptyFallback(course.Name, "-"),
			emptyFallback(course.Professor, "-"),
			evaluationCourseType(course),
		)
	}
}

func (r Runner) printEvaluationSubmitResult(result app.EvaluationSubmitResult) {
	if !result.Term.TermEnabled {
		_, _ = fmt.Fprintln(r.Out, "수업평가 기간이 아닙니다")
		return
	}
	if !result.Submitted {
		_, _ = fmt.Fprintln(r.Out, "수업평가 미리보기")
		_, _ = fmt.Fprintln(r.Out, "실제 제출하려면 같은 명령에 --yes를 붙이세요.")
	} else {
		_, _ = fmt.Fprintln(r.Out, "수업평가 제출 결과")
	}
	if len(result.Items) == 0 {
		_, _ = fmt.Fprintln(r.Out, "처리할 수업평가 과목이 없습니다")
		return
	}

	ready := 0
	skipped := 0
	failed := 0
	done := 0
	for _, item := range result.Items {
		label := fmt.Sprintf("%d. %s", item.Row.Index, emptyFallback(item.Row.Course.Name, "-"))
		switch {
		case item.Err != nil:
			failed++
			_, _ = fmt.Fprintf(r.Out, "실패: %s (%v)\n", label, item.Err)
		case item.Skipped:
			skipped++
			_, _ = fmt.Fprintf(r.Out, "건너뜀: %s (%s)\n", label, item.Reason)
		case result.Submitted:
			done++
			_, _ = fmt.Fprintf(r.Out, "완료: %s\n", label)
		default:
			ready++
			parts := []string{
				"답변=정말그렇다",
				"기타2=아니오",
				"서술형=많은 도움 되었습니다. 한학기동안 감사했습니다.",
			}
			if item.Row.Course.Engineering {
				parts = append(parts, "공학인증문항=제외")
			}
			_, _ = fmt.Fprintf(r.Out, "제출 가능: %s | %s\n", label, strings.Join(parts, " | "))
		}
	}
	if result.Submitted {
		_, _ = fmt.Fprintf(r.Out, "요약: 완료 %d, 건너뜀 %d, 실패 %d\n", done, skipped, failed)
		return
	}
	_, _ = fmt.Fprintf(r.Out, "요약: 제출 가능 %d, 건너뜀 %d, 실패 %d\n", ready, skipped, failed)
}

func formatClassRank(row app.Rank) string {
	if strings.TrimSpace(row.ClassRank) == "" && strings.TrimSpace(row.ClassSize) == "" {
		return "-"
	}
	return emptyFallback(row.ClassRank, "-") + " / " + emptyFallback(row.ClassSize, "-")
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

func formatCdpDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 && numericString(value) {
		return value[:4] + "-" + value[4:6] + "-" + value[6:]
	}
	return value
}

func formatCompactDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 && numericString(value) {
		return value[:4] + "-" + value[4:6] + "-" + value[6:]
	}
	return emptyFallback(value, "-")
}

func evaluationStatusLabel(course app.EvaluationCourse) string {
	if course.Evaluated {
		return "완료"
	}
	return "미완료"
}

func evaluationCourseType(course app.EvaluationCourse) string {
	parts := []string{}
	if strings.TrimSpace(course.CourseType) != "" {
		parts = append(parts, strings.TrimSpace(course.CourseType))
	}
	if strings.TrimSpace(course.Credits) != "" {
		parts = append(parts, strings.TrimSpace(course.Credits)+"학점")
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " / ")
}

func limitEvaluationRows(rows []app.EvaluationRow, limit int) []app.EvaluationRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func numericString(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func (r Runner) printSyllabus(result app.SyllabusResult) {
	syllabus := result.Syllabus
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	_, _ = fmt.Fprintf(r.Out, "과목: %s\n", emptyFallback(syllabus.FullName, emptyFallback(syllabus.KoreanName, result.Course.Name)))
	_, _ = fmt.Fprintf(r.Out, "학정번호: %s\n", emptyFallback(syllabus.CourseCode, "확인 필요"))
	_, _ = fmt.Fprintf(r.Out, "과목ID: %s\n", result.SubjectID)
	if syllabus.CourseType != "" || syllabus.Credits != "" {
		_, _ = fmt.Fprintf(r.Out, "이수/학점: %s / %s\n", emptyFallback(syllabus.CourseType, "-"), emptyFallback(syllabus.Credits, "-"))
	}
	if syllabus.Professor != "" {
		professor := syllabus.Professor
		if syllabus.ProfessorTitle != "" {
			professor += " (" + syllabus.ProfessorTitle + ")"
		}
		_, _ = fmt.Fprintf(r.Out, "담당교수: %s\n", professor)
	}
	if len(syllabus.Times) > 0 {
		_, _ = fmt.Fprintf(r.Out, "강의시간: %s\n", formatSyllabusTimes(syllabus.Times))
	}
	if syllabus.Operation != "" {
		_, _ = fmt.Fprintf(r.Out, "운영방식: %s\n", syllabus.Operation)
	}
	if syllabus.Competency != "" {
		_, _ = fmt.Fprintf(r.Out, "대표역량: %s\n", syllabus.Competency)
	}
	if syllabus.Summary != "" {
		_, _ = fmt.Fprintf(r.Out, "\n개요\n%s\n", syllabus.Summary)
	}
	if syllabus.Purpose != "" {
		_, _ = fmt.Fprintf(r.Out, "\n학습목표\n%s\n", syllabus.Purpose)
	}
	if syllabus.Outcome != "" {
		_, _ = fmt.Fprintf(r.Out, "\n학습성과\n%s\n", syllabus.Outcome)
	}
	if syllabus.BookName != "" {
		_, _ = fmt.Fprintf(r.Out, "\n교재: %s\n", syllabus.BookName)
	}
	_, _ = fmt.Fprintf(r.Out, "\n평가: %s\n", formatSyllabusEvaluation(syllabus.Evaluation))
	if len(syllabus.Schedule) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n주차별 계획")
		for _, week := range syllabus.Schedule {
			_, _ = fmt.Fprintf(r.Out, "  %d주차 | %s", week.Week, strings.ReplaceAll(week.Topic, "\n", " / "))
			if week.SubNote != "" {
				_, _ = fmt.Fprintf(r.Out, " | %s", strings.ReplaceAll(week.SubNote, "\n", " / "))
			}
			_, _ = fmt.Fprintln(r.Out)
		}
	}
}

func (r Runner) printAcademicList(result app.AcademicListResult) {
	_, _ = fmt.Fprintf(r.Out, "%s 학사일정\n", result.Year)
	if len(result.Events) == 0 {
		_, _ = fmt.Fprintln(r.Out, "학사일정이 없습니다")
		return
	}

	currentMonth := ""
	for _, event := range result.Events {
		if event.Month != currentMonth {
			if currentMonth != "" {
				_, _ = fmt.Fprintln(r.Out)
			}
			currentMonth = event.Month
			_, _ = fmt.Fprintln(r.Out, currentMonth)
		}
		_, _ = fmt.Fprintf(r.Out, "  %s | %s", event.Date, event.Title)
		if event.Note != "" {
			_, _ = fmt.Fprintf(r.Out, " | %s", event.Note)
		}
		_, _ = fmt.Fprintln(r.Out)
	}
	if result.SourceURL != "" {
		_, _ = fmt.Fprintf(r.Out, "\n출처: %s\n", r.linkifyForTerminal(result.SourceURL))
	}
}

func formatSyllabusTimes(times []app.SyllabusTime) string {
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

func formatSyllabusEvaluation(evaluation app.SyllabusEvaluation) string {
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
