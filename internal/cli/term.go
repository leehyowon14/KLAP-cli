package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

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
