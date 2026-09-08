package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

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
