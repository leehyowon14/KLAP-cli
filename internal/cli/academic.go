package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

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
