package klas

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type TimetableEntry struct {
	SubjectID   string
	SubjectName string
	Weekday     int
	Period      int
	Span        int
	Room        string
	Professor   string
	Online      bool
	Raw         map[string]any `json:"-"`
}

type timetableRow map[string]any

func (c *Client) Timetable(ctx context.Context, yearHakgi string) ([]TimetableEntry, error) {
	year, hakgi := splitYearHakgi(yearHakgi)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/TimetableStdList.do", map[string]any{
		"searchYear":  year,
		"searchHakgi": hakgi,
		"searchPgmNo": "",
	})
	if err != nil {
		return nil, err
	}

	var response []timetableRow
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, fmt.Errorf("시간표 목록 응답 파싱 실패: %w", err)
	}

	return parseTimetableEntries(response), nil
}

func parseTimetableEntries(response []timetableRow) []TimetableEntry {
	entries := make([]TimetableEntry, 0)
	for _, row := range response {
		if !strings.EqualFold(rowString(row, "wtHasSchedule"), "Y") {
			continue
		}
		period, err := strconv.Atoi(rowString(row, "wtTime"))
		if err != nil {
			continue
		}

		for weekday := 1; weekday <= 6; weekday++ {
			suffix := "_" + strconv.Itoa(weekday)
			subjectID := rowString(row, "wtSubj"+suffix)
			subjectName := rowString(row, "wtSubjNm"+suffix)
			if subjectID == "" && subjectName == "" {
				continue
			}
			if subjectName == "" {
				subjectName = subjectID
			}

			span := 1
			if parsedSpan, err := strconv.Atoi(rowString(row, "wtSpan"+suffix)); err == nil && parsedSpan > 0 {
				span = parsedSpan
			}

			room := rowString(row, "wtLocHname"+suffix)
			entries = append(entries, TimetableEntry{
				SubjectID:   subjectID,
				SubjectName: subjectName,
				Weekday:     weekday,
				Period:      period,
				Span:        span,
				Room:        room,
				Professor:   rowString(row, "wtProfNm"+suffix),
				Online:      period > 8 || room == "",
				Raw:         map[string]any(row),
			})
		}
	}
	return entries
}
