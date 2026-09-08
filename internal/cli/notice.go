package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"time"
)

func (r Runner) runNotice(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap notice <list|detail|open>")
	}

	switch args[0] {
	case "list":
		opts, err := noticeListOptions(args[1:])
		if err != nil {
			return err
		}
		rows, err := service.NoticeList(ctx, opts)
		if err != nil {
			return err
		}
		r.printNoticeRows(rows)
		return nil
	case "detail":
		if len(args) != 2 {
			return errors.New("usage: klap notice detail <공지ID>")
		}
		detail, err := service.NoticeDetail(ctx, args[1], app.UserOption{})
		if err != nil {
			return err
		}
		r.printNoticeDetail(detail)
		return nil
	case "open":
		if len(args) != 2 {
			return errors.New("usage: klap notice open <공지ID>")
		}
		result, err := service.NoticeOpenURL(ctx, args[1], app.UserOption{})
		return r.openAndPrintURL(result.URL, err)
	default:
		return fmt.Errorf("unknown notice command: %s", args[0])
	}
}

func noticeListOptions(args []string) (app.NoticeListOptions, error) {
	course, err := courseFilter(args)
	if err != nil {
		return app.NoticeListOptions{}, err
	}
	return app.NoticeListOptions{
		User:         app.UserOption{StudentID: userFlag(args)},
		CourseFilter: course,
		Refresh:      hasFlag(args, "--refresh"),
	}, nil
}

func (r Runner) printNoticeRows(rows []app.NoticeRow) {
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "강의 공지가 없습니다")
		return
	}

	for _, row := range rows {
		prefix := " "
		if row.Notice.Top {
			prefix = "!"
		}
		_, _ = fmt.Fprintf(r.Out, "%s %s | %s | %s | %s | %s\n",
			prefix,
			row.ID,
			formatNoticeTime(row.Notice.Registered),
			row.CourseName,
			row.Notice.Author,
			row.Notice.Title,
		)
	}
}

func (r Runner) printNoticeDetail(result app.NoticeDetailResult) {
	detail := result.Detail

	_, _ = fmt.Fprintf(r.Out, "ID: %s\n", result.ID)
	_, _ = fmt.Fprintf(r.Out, "과목: %s\n", result.CourseName)
	_, _ = fmt.Fprintf(r.Out, "제목: %s\n", detail.Title)
	if detail.Author != "" {
		_, _ = fmt.Fprintf(r.Out, "작성자: %s\n", detail.Author)
	}
	_, _ = fmt.Fprintf(r.Out, "작성일: %s\n", formatNoticeTime(detail.Registered))
	if detail.Top {
		_, _ = fmt.Fprintln(r.Out, "중요: 예")
	}
	if detail.ReadCount != "" {
		_, _ = fmt.Fprintf(r.Out, "조회수: %s\n", detail.ReadCount)
	}
	if detail.Attachment != "" {
		_, _ = fmt.Fprintf(r.Out, "첨부 묶음: %s\n", detail.Attachment)
	}
	_, _ = fmt.Fprintf(r.Out, "원문: %s\n", r.linkifyForTerminal(result.DetailURL))
	if detail.ContentText != "" {
		_, _ = fmt.Fprintf(r.Out, "\n%s\n", r.linkifyForTerminal(detail.ContentText))
	}
}

func formatNoticeTime(value *time.Time) string {
	if value == nil {
		return "작성일 확인 필요"
	}
	return value.Format("2006-01-02 15:04")
}
