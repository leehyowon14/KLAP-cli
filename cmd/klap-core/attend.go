package main

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func init() {
	handlers["attend"] = func(ctx context.Context, s *app.Service, r request) error {
		users, err := s.Users(ctx)
		if err != nil {
			return err
		}
		user, account, err := notificationAccount(users)
		if err != nil {
			return err
		}
		if r.ExpectedAccount != "" && r.ExpectedAccount != account {
			return fmt.Errorf("알림을 받은 계정과 현재 계정이 다릅니다")
		}
		return runAttendance(ctx, s, r.IDs, user, emit)
	}
}

type attender interface {
	AttendLecture(context.Context, string, app.LectureAttendOptions) (app.LectureAttendResult, error)
}

func runAttendance(ctx context.Context, service attender, idsInput []string, user app.UserOption, send func(string, any)) error {
	var err error
	seen := map[string]bool{}
	ids := []string{}
	for _, id := range idsInput {
		if id != "" && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return fmt.Errorf("수강할 강의가 없습니다")
	}
	success, failed := 0, 0
	for i, id := range ids {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		send("start", map[string]any{"id": id, "current": i + 1, "total": len(ids)})
		v, e := service.AttendLecture(ctx, id, app.LectureAttendOptions{User: user, RequireEligible: true, OnProgress: func(row app.LectureRow, p app.LectureProgress) {
			send("progress", map[string]any{"id": id, "title": row.Lecture.Title, "percent": p.Progress, "achieved": p.TotalTime, "required": p.PTime, "current": i + 1, "total": len(ids)})
		}})
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		if e != nil {
			failed++
			send("failed", map[string]any{"id": id, "message": e.Error()})
			continue
		}
		if !v.Progress.Completed {
			failed++
			send("failed", map[string]any{"id": id, "message": "서버에서 수강 완료가 확인되지 않았습니다"})
			continue
		}
		success++
		send("completed", map[string]any{"id": id, "title": v.Lecture.Lecture.Title, "current": i + 1, "total": len(ids)})
	}
	if err == nil {
		send("done", map[string]any{"success": success, "failed": failed, "total": len(ids)})
	}
	return err
}
