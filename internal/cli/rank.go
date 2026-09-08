package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

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

func formatClassRank(row app.Rank) string {
	if strings.TrimSpace(row.ClassRank) == "" && strings.TrimSpace(row.ClassSize) == "" {
		return "-"
	}
	return emptyFallback(row.ClassRank, "-") + " / " + emptyFallback(row.ClassSize, "-")
}
