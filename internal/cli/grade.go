package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

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
