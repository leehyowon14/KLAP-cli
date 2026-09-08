package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

func (r Runner) runRoom(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap room <free|busy|index|cache>")
	}
	switch args[0] {
	case "free":
		opts, err := roomQueryOptions(args[1:])
		if err != nil {
			return err
		}
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomFree(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomFree(result)
		return nil
	case "busy":
		opts, err := roomQueryOptions(args[1:])
		if err != nil {
			return err
		}
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomBusy(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomBusy(result)
		return nil
	case "available", "empty":
		opts, err := roomAvailableOptions(args[1:])
		if err != nil {
			return err
		}
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomAvailable(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomAvailable(result)
		return nil
	case "index":
		opts, err := roomIndexOptions(args[1:])
		if err != nil {
			return err
		}
		opts.Refresh = true
		progressed := false
		opts.OnProgress = func(done int, total int, label string) {
			progressed = true
			_, _ = fmt.Fprintf(r.Out, "\r인덱싱: %d/%d %s", done, total, truncateForLine(label, 32))
		}
		result, err := service.RoomIndex(ctx, opts)
		if progressed {
			_, _ = fmt.Fprintln(r.Out)
		}
		if err != nil {
			return err
		}
		r.printRoomIndex(result)
		return nil
	case "cache":
		if len(args) == 2 && args[1] == "clear" {
			result, err := service.ClearRoomCache()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(r.Out, "room-index cache 삭제: %d개\n", result.Removed)
			return nil
		}
		return errors.New("usage: klap room cache clear")
	default:
		return fmt.Errorf("unknown room command: %s", args[0])
	}
}

func roomQueryOptions(args []string) (app.RoomQueryOptions, error) {
	opts := app.RoomQueryOptions{User: app.UserOption{StudentID: userFlag(args)}}
	roomParts := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.RoomQueryOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--building":
			if i+1 >= len(args) {
				return app.RoomQueryOptions{}, errors.New("--building에는 건물명이 필요합니다")
			}
			opts.Building = args[i+1]
			i++
		case "--refresh":
			opts.Refresh = true
		case "--user":
			if i+1 >= len(args) {
				return app.RoomQueryOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return app.RoomQueryOptions{}, fmt.Errorf("unknown room option: %s", args[i])
			}
			roomParts = append(roomParts, args[i])
		}
	}
	opts.Room = strings.Join(roomParts, " ")
	if strings.TrimSpace(opts.Room) == "" {
		return app.RoomQueryOptions{}, errors.New("usage: klap room <free|busy> <강의실명> [--term YYYY-S] [--building <건물명>] [--refresh]")
	}
	return opts, nil
}

func roomAvailableOptions(args []string) (app.RoomAvailableOptions, error) {
	opts := app.RoomAvailableOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--day":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--day에는 요일이 필요합니다")
			}
			opts.Day = args[i+1]
			i++
		case "--duration":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--duration에는 1-3 또는 5 같은 교시 범위가 필요합니다")
			}
			opts.Duration = args[i+1]
			i++
		case "--term":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--building":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--building에는 건물명이 필요합니다")
			}
			opts.Building = args[i+1]
			i++
		case "--refresh":
			opts.Refresh = true
		case "--user":
			if i+1 >= len(args) {
				return app.RoomAvailableOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			return app.RoomAvailableOptions{}, fmt.Errorf("unknown room available option: %s", args[i])
		}
	}
	if strings.TrimSpace(opts.Day) == "" || strings.TrimSpace(opts.Duration) == "" {
		return app.RoomAvailableOptions{}, errors.New("usage: klap room available --day <요일> --duration <교시범위> [--term YYYY-S] [--building <건물명>] [--refresh]")
	}
	return opts, nil
}

func roomIndexOptions(args []string) (app.RoomIndexOptions, error) {
	opts := app.RoomIndexOptions{User: app.UserOption{StudentID: userFlag(args)}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--term":
			if i+1 >= len(args) {
				return app.RoomIndexOptions{}, errors.New("--term에는 YYYY-S 형식의 학기가 필요합니다")
			}
			opts.TermValue = args[i+1]
			i++
		case "--building":
			if i+1 >= len(args) {
				return app.RoomIndexOptions{}, errors.New("--building에는 건물명이 필요합니다")
			}
			opts.Building = args[i+1]
			i++
		case "--refresh":
			opts.Refresh = true
		case "--user":
			if i+1 >= len(args) {
				return app.RoomIndexOptions{}, errors.New("--user에는 학번이 필요합니다")
			}
			i++
		default:
			return app.RoomIndexOptions{}, fmt.Errorf("unknown room index option: %s", args[i])
		}
	}
	return opts, nil
}

func (r Runner) printRoomFree(result app.RoomQueryResult) {
	if !r.printRoomQueryHeader(result) {
		return
	}
	_, _ = fmt.Fprintf(r.Out, "%s 빈 시간 | %s%s\n", result.Room, displayRoomTermValue(result.TermValue), roomCacheLabel(result.Cached))
	byWeekday := make(map[int][]int)
	for _, slot := range result.FreeSlots {
		byWeekday[slot.Weekday] = append(byWeekday[slot.Weekday], slot.Period)
	}
	for weekday := 1; weekday <= 5; weekday++ {
		periods := byWeekday[weekday]
		if len(periods) == 0 {
			_, _ = fmt.Fprintf(r.Out, "%s: 없음\n", app.RoomWeekdayLabel(weekday))
			continue
		}
		values := make([]string, 0, len(periods))
		for _, period := range periods {
			values = append(values, strconv.Itoa(period))
		}
		_, _ = fmt.Fprintf(r.Out, "%s: %s교시\n", app.RoomWeekdayLabel(weekday), strings.Join(values, ", "))
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomBusy(result app.RoomQueryResult) {
	if !r.printRoomQueryHeader(result) {
		return
	}
	_, _ = fmt.Fprintf(r.Out, "%s 사용 목록 | %s%s\n", result.Room, displayRoomTermValue(result.TermValue), roomCacheLabel(result.Cached))
	if len(result.BusyRows) == 0 {
		_, _ = fmt.Fprintln(r.Out, "사용 중인 수업이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return
	}
	for _, row := range result.BusyRows {
		course := emptyFallback(row.CourseName, row.CourseCode)
		professor := emptyFallback(row.Professor, "교수 미지정")
		_, _ = fmt.Fprintf(r.Out, "%s %s | %s | %s | %s\n", app.RoomWeekdayLabel(row.Weekday), roomPeriodLabel(row.Period, row.Span), course, professor, row.CourseCode)
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomAvailable(result app.RoomAvailableResult) {
	status := roomAvailabilityStatus(result.Weekday, result.Periods)
	_, _ = fmt.Fprintf(r.Out, "빈 강의실 | %s | %s%s\n", displayRoomTermValue(result.TermValue), status, roomCacheLabel(result.Cached))
	if len(result.Rooms) == 0 {
		_, _ = fmt.Fprintln(r.Out, "조건에 맞는 빈 강의실이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return
	}
	for index, room := range result.Rooms {
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %s\n", index+1, room.Room, status)
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomIndex(result app.RoomIndexResult) {
	_, _ = fmt.Fprintf(r.Out, "강의실 인덱스 | %s | %d개%s\n", displayRoomTermValue(result.TermValue), len(result.Rooms), roomCacheLabel(result.Cached))
	if len(result.Rooms) == 0 {
		_, _ = fmt.Fprintln(r.Out, "조회된 강의실이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return
	}
	for index, room := range result.Rooms {
		_, _ = fmt.Fprintf(r.Out, "%d. %s | %d개 교시 사용\n", index+1, room.Room, room.BusyCount)
	}
	r.printRoomWarnings(result.Warnings)
}

func (r Runner) printRoomQueryHeader(result app.RoomQueryResult) bool {
	if result.Room != "" {
		return true
	}
	if len(result.Candidates) == 0 {
		_, _ = fmt.Fprintln(r.Out, "매칭되는 강의실이 없습니다")
		r.printRoomWarnings(result.Warnings)
		return false
	}
	_, _ = fmt.Fprintln(r.Out, "여러 강의실이 매칭되었습니다. 정확한 강의실명으로 다시 조회하세요.")
	for index, room := range result.Candidates {
		_, _ = fmt.Fprintf(r.Out, "%d. %s\n", index+1, room)
	}
	r.printRoomWarnings(result.Warnings)
	return false
}

func (r Runner) printRoomWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	_, _ = fmt.Fprintf(r.Out, "\n경고: %d개 과목의 강의시간 조회에 실패했습니다\n", len(warnings))
	limit := len(warnings)
	if limit > 20 {
		limit = 20
	}
	for _, warning := range warnings[:limit] {
		_, _ = fmt.Fprintf(r.Out, "- %s\n", warning)
	}
	if len(warnings) > limit {
		_, _ = fmt.Fprintf(r.Out, "- ... %d개 생략\n", len(warnings)-limit)
	}
}

func displayRoomTermValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "학기 미지정"
	}
	return strings.ReplaceAll(value, ",", "-")
}

func roomCacheLabel(cached bool) string {
	if cached {
		return " | cache"
	}
	return ""
}

func roomAvailabilityStatus(weekday int, periods []int) string {
	return app.RoomWeekdayLabel(weekday) + " " + roomPeriodsLabel(periods) + " 비어있음"
}

func roomPeriodLabel(period int, span int) string {
	if span <= 1 {
		return strconv.Itoa(period) + "교시"
	}
	return strconv.Itoa(period) + "-" + strconv.Itoa(period+span-1) + "교시"
}

func roomPeriodsLabel(periods []int) string {
	if len(periods) == 0 {
		return "교시 미지정"
	}
	start := periods[0]
	end := periods[len(periods)-1]
	if start == end {
		return strconv.Itoa(start) + "교시"
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end) + "교시"
}
