package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

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
