package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
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
