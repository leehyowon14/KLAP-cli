package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

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
