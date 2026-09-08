package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

func (r Runner) runAttendance(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "cdp" {
		result, err := service.CdpAttendance(ctx, app.CdpAttendanceOptions{
			User: app.UserOption{StudentID: userFlag(args[1:])},
		})
		if err != nil {
			return err
		}
		r.printCdpAttendance(result)
		return nil
	}
	if len(args) > 0 && args[0] == "detail" {
		if len(args) != 2 {
			return errors.New("usage: klap attendance detail <과목명|번호|학정번호>")
		}
		result, err := service.AttendanceDetail(ctx, app.AttendanceDetailOptions{
			User:     app.UserOption{},
			Selector: args[1],
		})
		if err != nil {
			return err
		}
		r.printAttendanceDetail(result)
		return nil
	}
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown attendance command: %s", args[0])
	}

	result, err := service.AttendanceList(ctx, app.AttendanceListOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Refresh: hasFlag(args, "--refresh"),
	})
	if err != nil {
		return err
	}
	r.printAttendanceList(result)
	return nil
}

func (r Runner) printAttendanceList(result app.AttendanceListResult) {
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "출석 현황이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "번호 | 학정번호 | 과목 | 교수 | 강의시간 | 출석 요약")
	for _, row := range result.Rows {
		course := row.Course
		summary := attendanceSummary(row)
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %s | %s | %s | %s\n",
			row.Index,
			emptyFallback(course.CourseCode, "-"),
			emptyFallback(course.Name, "-"),
			emptyFallback(course.Professor, "-"),
			emptyFallback(course.Weekday, "확인 필요"),
			summary,
		)
	}
}

func (r Runner) printAttendanceDetail(result app.AttendanceDetailResult) {
	row := result.Row
	course := row.Course
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	_, _ = fmt.Fprintf(r.Out, "과목: %s\n", course.Name)
	_, _ = fmt.Fprintf(r.Out, "학정번호: %s\n", emptyFallback(course.CourseCode, "-"))
	_, _ = fmt.Fprintf(r.Out, "교수: %s\n", emptyFallback(course.Professor, "-"))
	_, _ = fmt.Fprintf(r.Out, "강의시간: %s\n", emptyFallback(course.Weekday, "확인 필요"))
	_, _ = fmt.Fprintf(r.Out, "출석 요약: %s\n", attendanceSummary(row))
	if row.Err != nil {
		_, _ = fmt.Fprintf(r.Out, "상세 오류: %v\n", row.Err)
		return
	}
	if len(row.Sessions) == 0 {
		_, _ = fmt.Fprintln(r.Out, "상세 출석 내역이 없습니다")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "\n주차별 출석")
	for _, session := range row.Sessions {
		parts := make([]string, 0, len(session.Slots))
		for _, slot := range session.Slots {
			parts = append(parts, fmt.Sprintf("%d차시 %s %s", slot.Index, attendanceMarkLabel(slot.Mark), formatAttendanceDate(slot.Date)))
		}
		_, _ = fmt.Fprintf(r.Out, "%s주차 | %s\n", emptyFallback(session.Week, "-"), strings.Join(parts, " / "))
	}
}

func (r Runner) printCdpAttendance(result app.CdpAttendanceResult) {
	report := result.Report
	_, _ = fmt.Fprintln(r.Out, "CDP 출석내역")
	_, _ = fmt.Fprintf(r.Out, "총 출석: %s회\n", emptyFallback(report.TotalCount, "0"))
	if len(report.Rows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "CDP 출석내역이 없습니다")
		_, _ = fmt.Fprintln(r.Out, "* 출석내역은 출석 후 약 일주일 후에 반영됩니다.")
		return
	}

	_, _ = fmt.Fprintln(r.Out, "날짜 | 회차 | 강의주제 | 강사명")
	for _, row := range report.Rows {
		_, _ = fmt.Fprintf(r.Out, "%s | %s | %s | %s\n",
			emptyFallback(formatCdpDate(row.Date), "-"),
			emptyFallback(row.Seq, "-"),
			emptyFallback(row.Title, "-"),
			emptyFallback(row.Speaker, "-"),
		)
	}
	_, _ = fmt.Fprintln(r.Out, "* 출석내역은 출석 후 약 일주일 후에 반영됩니다.")
}

func attendanceSummary(row app.AttendanceRow) string {
	if row.Err != nil {
		return "상세 확인 실패"
	}
	counts := map[string]int{}
	total := 0
	for _, session := range row.Sessions {
		for _, slot := range session.Slots {
			mark := emptyFallback(slot.Mark, slot.Status)
			if mark == "" {
				continue
			}
			counts[mark]++
			total++
		}
	}
	if total == 0 {
		return "상세 없음"
	}
	return fmt.Sprintf("출석 %d / 결석 %d / 지각 %d / 조퇴 %d / 공결 %d / 전체 %d",
		counts["O"],
		counts["X"],
		counts["L"],
		counts["R"],
		counts["A"],
		total,
	)
}

func attendanceMarkLabel(mark string) string {
	switch mark {
	case "O":
		return "출석"
	case "X":
		return "결석"
	case "L":
		return "지각"
	case "R":
		return "조퇴"
	case "A":
		return "공결"
	default:
		return emptyFallback(mark, "-")
	}
}

func formatAttendanceDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) != 8 {
		return value
	}
	return value[:4] + "-" + value[4:6] + "-" + value[6:]
}

func formatCdpDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 8 && numericString(value) {
		return value[:4] + "-" + value[4:6] + "-" + value[6:]
	}
	return value
}
