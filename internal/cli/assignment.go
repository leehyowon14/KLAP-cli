package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"time"
)

func (r Runner) runAssignment(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap assignment <list|detail|open|remind>")
	}

	switch args[0] {
	case "list":
		opts, err := assignmentListOptions(args[1:])
		if err != nil {
			return err
		}
		rows, err := service.AssignmentList(ctx, opts)
		if err != nil {
			return err
		}
		r.printAssignmentRows(rows)
		return nil
	case "detail":
		if len(args) != 2 {
			return errors.New("usage: klap assignment detail <과제ID>")
		}
		detail, err := service.AssignmentDetail(ctx, args[1], app.UserOption{})
		if err != nil {
			return err
		}
		r.printAssignmentDetail(detail)
		return nil
	case "open":
		if len(args) != 2 {
			return errors.New("usage: klap assignment open <과제ID>")
		}
		result, err := service.AssignmentOpenURL(ctx, args[1], app.UserOption{})
		return r.openAndPrintURL(result.URL, err)
	case "remind":
		return r.runAssignmentRemind(ctx, service, args[1:])
	default:
		return fmt.Errorf("unknown assignment command: %s", args[0])
	}
}

func (r Runner) runAssignmentRemind(ctx context.Context, service *app.Service, args []string) error {
	auto := hasFlag(args, "--auto")
	if !auto {
		return r.syncAssignmentReminders(ctx, service, args)
	}

	_, _ = fmt.Fprintln(r.Out, "과제 reminder 자동 동기화를 시작합니다. 종료하려면 Ctrl+C를 누르세요.")
	for {
		if err := r.syncAssignmentReminders(ctx, service, args); err != nil {
			_, _ = fmt.Fprintf(r.Out, "동기화 실패: %v\n", err)
		}

		timer := time.NewTimer(30 * time.Minute)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (r Runner) syncAssignmentReminders(ctx context.Context, service *app.Service, args []string) error {
	opts, err := assignmentListOptions(args)
	if err != nil {
		return err
	}

	result, err := service.SyncAssignmentReminders(ctx, app.AssignmentSyncOptions{Query: opts})
	if err != nil {
		return err
	}
	if result.EligibleCount == 0 {
		_, _ = fmt.Fprintln(r.Out, "등록할 과제 reminder가 없습니다")
		return nil
	}

	_, _ = fmt.Fprintf(r.Out, "Reminder 동기화 완료: 생성 %d, 갱신 %d, 완료 %d, 제외 %d\n",
		result.Result.Created,
		result.Result.Updated,
		result.Result.Completed,
		result.Result.Skipped,
	)
	return nil
}

func assignmentListOptions(args []string) (app.AssignmentListOptions, error) {
	course, err := courseFilter(args)
	if err != nil {
		return app.AssignmentListOptions{}, err
	}
	return app.AssignmentListOptions{
		User:         app.UserOption{StudentID: userFlag(args)},
		CourseFilter: course,
		Refresh:      hasFlag(args, "--refresh"),
	}, nil
}

func (r Runner) printAssignmentRows(rows []app.AssignmentRow) {
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "과제가 없습니다")
		return
	}

	for _, row := range rows {
		status := "미제출"
		if row.Assignment.Submitted {
			status = "제출"
		}
		_, _ = fmt.Fprintf(r.Out, "%s | %s | %s | %s | %s\n",
			row.ID,
			formatTime(row.Assignment.DueAt),
			status,
			row.CourseName,
			row.Assignment.Title,
		)
	}
}

func (r Runner) printAssignmentDetail(result app.AssignmentDetailResult) {
	detail := result.Detail
	status := "미제출"
	if detail.Submitted {
		status = "제출"
	}

	_, _ = fmt.Fprintf(r.Out, "ID: %s\n", result.ID)
	_, _ = fmt.Fprintf(r.Out, "과목: %s\n", result.CourseName)
	_, _ = fmt.Fprintf(r.Out, "제목: %s\n", detail.Title)
	_, _ = fmt.Fprintf(r.Out, "마감: %s\n", formatTime(detail.DueAt))
	_, _ = fmt.Fprintf(r.Out, "상태: %s\n", status)
	if detail.ReportType != "" {
		_, _ = fmt.Fprintf(r.Out, "제출 방식: %s\n", detail.ReportType)
	}
	if detail.SubmitFileType != "" {
		_, _ = fmt.Fprintf(r.Out, "파일 형식: %s\n", detail.SubmitFileType)
	}
	if detail.FileLimitMB != "" {
		_, _ = fmt.Fprintf(r.Out, "파일 제한: %sMB\n", detail.FileLimitMB)
	}
	if detail.ContentText != "" {
		_, _ = fmt.Fprintf(r.Out, "\n%s\n", r.linkifyForTerminal(detail.ContentText))
	}
	if detail.SubmittedText != "" || detail.SubmittedTitle != "" {
		_, _ = fmt.Fprintln(r.Out, "\n내 제출")
		if detail.SubmittedTitle != "" {
			_, _ = fmt.Fprintf(r.Out, "제목: %s\n", detail.SubmittedTitle)
		}
		if detail.SubmittedText != "" {
			_, _ = fmt.Fprintln(r.Out, r.linkifyForTerminal(detail.SubmittedText))
		}
	}
	if detail.FinalScore != "" && detail.FinalScore != "<nil>" {
		_, _ = fmt.Fprintf(r.Out, "\n점수: %s\n", detail.FinalScore)
	}
	if detail.TutorText != "" {
		_, _ = fmt.Fprintf(r.Out, "\n피드백:\n%s\n", r.linkifyForTerminal(detail.TutorText))
	}
	if result.DetailURL != "" {
		_, _ = fmt.Fprintf(r.Out, "\n원문: %s\n", r.linkifyForTerminal(result.DetailURL))
	}
}
