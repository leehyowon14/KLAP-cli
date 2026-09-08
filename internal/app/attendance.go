package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strconv"
	"strings"
)

type AttendanceListOptions struct {
	User    UserOption
	Refresh bool
}

type AttendanceDetailOptions struct {
	User     UserOption
	Selector string
}

type CdpAttendanceOptions struct {
	User UserOption
}

type AttendanceListResult struct {
	Term Term
	Rows []AttendanceRow
}

type AttendanceRow struct {
	Index    int
	Course   klas.AttendanceCourse
	Sessions []klas.AttendanceSession
	Err      error
}

type AttendanceDetailResult struct {
	Term Term
	Row  AttendanceRow
}

type CdpAttendanceResult struct {
	Report klas.CdpAttendanceReport
}

func (s *Service) AttendanceList(ctx context.Context, opts AttendanceListOptions) (AttendanceListResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return AttendanceListResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return AttendanceListResult{}, err
	}

	cacheKey := listCacheKeyVersion("attendance", "v2", studentID, term.Value, "")
	if !opts.Refresh {
		var cached AttendanceListResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	courses, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.AttendanceCourse, error) {
		return client.AttendanceCourses(ctx, term.Value)
	})
	if err != nil {
		return AttendanceListResult{}, err
	}

	rows := make([]AttendanceRow, 0, len(courses))
	for index, course := range courses {
		sessions, detailErr := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.AttendanceSession, error) {
			return client.AttendanceSessions(ctx, term.Value, course)
		})
		rows = append(rows, AttendanceRow{
			Index:    index + 1,
			Course:   course,
			Sessions: sessions,
			Err:      detailErr,
		})
	}
	result := AttendanceListResult{
		Term: term,
		Rows: rows,
	}
	if attendanceCacheable(result) {
		_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	}
	return result, nil
}

func (s *Service) AttendanceDetail(ctx context.Context, opts AttendanceDetailOptions) (AttendanceDetailResult, error) {
	selector := strings.TrimSpace(opts.Selector)
	if selector == "" {
		return AttendanceDetailResult{}, errors.New("출석 상세 조회 대상이 없습니다")
	}
	result, err := s.AttendanceList(ctx, AttendanceListOptions{User: opts.User})
	if err != nil {
		return AttendanceDetailResult{}, err
	}
	rows := selectAttendanceRows(result.Rows, selector)
	if len(rows) == 0 {
		return AttendanceDetailResult{}, fmt.Errorf("출석 현황 과목을 찾을 수 없습니다: %s", selector)
	}
	if len(rows) > 1 {
		return AttendanceDetailResult{}, fmt.Errorf("출석 현황 과목이 여러 개와 일치합니다. 번호 또는 학정번호를 사용하세요: %s", selector)
	}
	return AttendanceDetailResult{
		Term: result.Term,
		Row:  rows[0],
	}, nil
}

func (s *Service) CdpAttendance(ctx context.Context, opts CdpAttendanceOptions) (CdpAttendanceResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return CdpAttendanceResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return CdpAttendanceResult{}, err
	}

	report, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.CdpAttendanceReport, error) {
		return client.CdpAttendance(ctx)
	})
	if err != nil {
		return CdpAttendanceResult{}, err
	}
	return CdpAttendanceResult{Report: report}, nil
}

func attendanceCacheable(result AttendanceListResult) bool {
	for _, row := range result.Rows {
		if row.Err != nil {
			return false
		}
	}
	return true
}

func selectAttendanceRows(rows []AttendanceRow, selector string) []AttendanceRow {
	selector = strings.TrimSpace(selector)
	if number, err := strconv.Atoi(selector); err == nil {
		if number >= 1 && number <= len(rows) {
			return []AttendanceRow{rows[number-1]}
		}
		return nil
	}

	matches := make([]AttendanceRow, 0)
	normalizedSelector := strings.ToLower(selector)
	for _, row := range rows {
		course := row.Course
		if strings.EqualFold(course.CourseCode, selector) ||
			strings.Contains(strings.ToLower(course.Name), normalizedSelector) ||
			strings.Contains(strings.ToLower(course.Professor), normalizedSelector) {
			matches = append(matches, row)
		}
	}
	return matches
}
