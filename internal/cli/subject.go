package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

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
