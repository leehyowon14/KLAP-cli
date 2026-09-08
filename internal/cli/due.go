package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
)

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
