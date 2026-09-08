package cli

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func (r Runner) runDashboard(ctx context.Context, service *app.Service, args []string) error {
	if unknown := firstUnknownDashboardArg(args); unknown != "" {
		return fmt.Errorf("unknown dashboard option: %s", unknown)
	}
	result, err := service.Dashboard(ctx, app.DashboardOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Refresh: dashboardRefreshFlag(args),
	})
	if err != nil {
		return err
	}
	r.printDashboard(result)
	return nil
}

func firstUnknownDashboardArg(args []string) string {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--refresh":
		case "--user":
			if i+1 >= len(args) {
				return "--user"
			}
			i++
		default:
			return args[i]
		}
	}
	return ""
}

func dashboardRefreshFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--refresh" {
			return true
		}
	}
	return false
}

func (r Runner) printDashboard(result app.DashboardResult) {
	_, _ = fmt.Fprintf(r.Out, "KLAP Dashboard | %s (%s)\n", result.Term.Label, result.Term.Value)
	if !result.GeneratedAt.IsZero() {
		_, _ = fmt.Fprintf(r.Out, "갱신: %s\n", result.GeneratedAt.Format("2006-01-02 15:04"))
	}
	if result.Cached {
		_, _ = fmt.Fprintf(r.Out, "캐시: 사용")
		if !result.CacheCreatedAt.IsZero() {
			_, _ = fmt.Fprintf(r.Out, " (%s)", result.CacheCreatedAt.Format("2006-01-02 15:04"))
		}
		_, _ = fmt.Fprintln(r.Out)
	}

	_, _ = fmt.Fprintln(r.Out, "\n과제")
	if len(result.Assignments) == 0 {
		_, _ = fmt.Fprintln(r.Out, "  예정된 미제출 과제가 없습니다")
	} else {
		for _, row := range result.Assignments {
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
				row.ID,
				formatTime(row.Assignment.DueAt),
				row.CourseName,
				row.Assignment.Title,
			)
		}
	}

	_, _ = fmt.Fprintln(r.Out, "\n온라인 강의")
	if len(result.Lectures) == 0 {
		_, _ = fmt.Fprintln(r.Out, "  수강할 온라인 강의/학습활동이 없습니다")
	} else {
		for _, row := range result.Lectures {
			percent := lectureStatusPercent(row.Lecture)
			_, _ = fmt.Fprintf(r.Out, "  %s %s | %s | %s | %s\n",
				renderProgressBar(percent, 18),
				formatLectureStatusMinutes(row.Lecture),
				row.ID,
				row.CourseName,
				row.Lecture.Title,
			)
		}
	}

	_, _ = fmt.Fprintln(r.Out, "\n공지")
	if len(result.Notices) == 0 {
		_, _ = fmt.Fprintln(r.Out, "  최근 강의 공지가 없습니다")
	} else {
		for _, row := range result.Notices {
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
				formatNoticeTime(row.Notice.Registered),
				row.ID,
				row.CourseName,
				row.Notice.Title,
			)
		}
	}

	_, _ = fmt.Fprintln(r.Out, "\n출석")
	attendance := result.Attendance
	if attendance.TotalCourses == 0 {
		_, _ = fmt.Fprintln(r.Out, "  출석 현황이 없습니다")
	} else {
		_, _ = fmt.Fprintf(r.Out, "  전체 과목 %d개 | 출석 %d / 결석 %d / 지각 %d / 조퇴 %d / 공결 %d / 미확인 %d\n",
			attendance.TotalCourses,
			attendance.Completed,
			attendance.Absent,
			attendance.Late,
			attendance.LeaveEarly,
			attendance.Excused,
			attendance.Unknown,
		)
		if attendance.DetailErrors > 0 {
			_, _ = fmt.Fprintf(r.Out, "  상세 확인 실패: %d개 과목\n", attendance.DetailErrors)
		}
		for _, row := range attendance.Rows {
			_, _ = fmt.Fprintf(r.Out, "  %d. %s | %s\n",
				row.Index,
				emptyFallback(row.Course.Name, "-"),
				dashboardAttendanceRowSummary(row),
			)
		}
	}

	_, _ = fmt.Fprintln(r.Out, "\n수업평가")
	evaluation := result.Evaluation
	if !evaluation.Enabled {
		_, _ = fmt.Fprintln(r.Out, "  수업평가 기간이 아닙니다. 중간/기말 차수는 평가 기간에만 표시됩니다")
	} else {
		_, _ = fmt.Fprintf(r.Out, "  %s | 완료 %d / 미완료 %d\n",
			emptyFallback(evaluation.Term.JudgeName, evaluation.Term.JudgeChasu),
			evaluation.Done,
			evaluation.Pending,
		)
		for _, row := range limitEvaluationRows(evaluation.Rows, 5) {
			extra := ""
			if row.Course.Engineering {
				extra = " | 공학인증문항 제외"
			}
			_, _ = fmt.Fprintf(r.Out, "  %d. %s%s\n", row.Index, row.Course.Name, extra)
		}
	}

	if len(result.SectionErrors) > 0 {
		_, _ = fmt.Fprintln(r.Out, "\n확인 실패")
		for _, sectionError := range result.SectionErrors {
			_, _ = fmt.Fprintf(r.Out, "  %s: %v\n", sectionError.Section, sectionError.Err)
		}
	}
}

func dashboardAttendanceRowSummary(row app.DashboardAttendanceRow) string {
	if row.Err != nil {
		return "상세 확인 실패"
	}
	total := row.Completed + row.Absent + row.Late + row.LeaveEarly + row.Excused + row.Unknown
	if total == 0 {
		return "상세 없음"
	}
	return fmt.Sprintf("출석 %d / 결석 %d / 지각 %d / 조퇴 %d / 공결 %d / 전체 %d",
		row.Completed,
		row.Absent,
		row.Late,
		row.LeaveEarly,
		row.Excused,
		total,
	)
}
