package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
)

type SearchOptions struct {
	User    UserOption
	Query   string
	Type    string
	Refresh bool
}

type SearchResult struct {
	Query       string
	Type        string
	Courses     []SearchCourseResult
	Assignments []AssignmentRow
	Notices     []NoticeRow
	Lectures    []LectureRow
	Academics   []AcademicEvent
	Errors      []DashboardSectionError
}

type SearchCourseResult struct {
	Index  int
	Term   klas.Term
	Course klas.Course
}

func (s *Service) Search(ctx context.Context, opts SearchOptions) (SearchResult, error) {
	s = s.withRequestSession()
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return SearchResult{}, errors.New("검색어를 입력해야 합니다")
	}
	searchType := strings.ToLower(strings.TrimSpace(opts.Type))
	if searchType != "" && !validSearchType(searchType) {
		return SearchResult{}, fmt.Errorf("지원하지 않는 검색 타입입니다: %s", opts.Type)
	}

	result := SearchResult{
		Query: query,
		Type:  searchType,
	}
	user := UserOption{StudentID: strings.TrimSpace(opts.User.StudentID)}

	if searchType == "" || searchType == "course" {
		terms, err := s.CourseList(ctx, CourseListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "과목", Err: err})
		} else if len(terms) > 0 {
			for index, course := range terms[0].Courses {
				if textMatches(query, course.Name, course.Value) {
					result.Courses = append(result.Courses, SearchCourseResult{
						Index:  index + 1,
						Term:   terms[0],
						Course: course,
					})
				}
			}
		}
	}

	if searchType == "" || searchType == "assignment" {
		rows, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "과제", Err: err})
		} else {
			for _, row := range rows {
				if textMatches(query, row.ID, row.CourseName, row.Assignment.Title) {
					result.Assignments = append(result.Assignments, row)
				}
			}
		}
	}

	if searchType == "" || searchType == "notice" {
		rows, err := s.NoticeList(ctx, NoticeListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "공지", Err: err})
		} else {
			for _, row := range rows {
				if textMatches(query, row.ID, row.CourseName, row.Notice.Title, row.Notice.Author) {
					result.Notices = append(result.Notices, row)
				}
			}
		}
	}

	if searchType == "" || searchType == "lecture" {
		rows, err := s.LectureList(ctx, LectureListOptions{User: user, Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "온라인 강의", Err: err})
		} else {
			for _, row := range rows {
				if textMatches(query, row.ID, row.CourseName, row.Lecture.ModuleTitle, row.Lecture.Title) {
					result.Lectures = append(result.Lectures, row)
				}
			}
		}
	}

	if searchType == "" || searchType == "academic" {
		academic, err := s.AcademicList(ctx, AcademicListOptions{Refresh: opts.Refresh})
		if err != nil {
			result.Errors = append(result.Errors, DashboardSectionError{Section: "학사일정", Err: err})
		} else {
			for _, event := range academic.Events {
				if textMatches(query, event.Year, event.Month, event.Date, event.Title, event.Note) {
					result.Academics = append(result.Academics, event)
				}
			}
		}
	}

	return result, nil
}

func validSearchType(value string) bool {
	switch value {
	case "course", "assignment", "notice", "lecture", "academic":
		return true
	default:
		return false
	}
}

func textMatches(query string, values ...string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return false
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), query) {
			return true
		}
	}
	return false
}
