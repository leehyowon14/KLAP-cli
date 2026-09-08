package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type AssignmentListOptions struct {
	User          UserOption
	CourseFilter  string
	Refresh       bool
	SyncDecisions map[string]SyncDecision
}

type AssignmentGateway interface {
	Assignments(context.Context, string, klas.Course) ([]klas.Assignment, error)
	AssignmentDetail(context.Context, string, klas.Course, string) (klas.AssignmentDetail, error)
}

type AssignmentRow struct {
	ID         string
	LegacyID   string `json:"-"`
	TermValue  string
	CourseName string
	DetailURL  string
	Assignment Assignment
}

type AssignmentDetailResult struct {
	ID         string
	TermValue  string
	CourseName string
	DetailURL  string
	Detail     AssignmentDetail
}

func (s *Service) AssignmentList(ctx context.Context, opts AssignmentListOptions) ([]AssignmentRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return nil, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return nil, err
	}

	cacheKey := courseResourceListCacheKeyVersion("assignment", "v2", studentID, term.Value, courses)
	if !opts.Refresh {
		var cached []AssignmentRow
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			rows, migrated, migrationErr := normalizeCachedAssignmentRows(cached, term)
			if migrationErr == nil && resourceIDsMatchSelected(assignmentRowIDs(rows), "assignment", 1, courses, true) {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
				}
				return rows, nil
			}
		}
	}

	rows := make([]AssignmentRow, 0)
	for _, selectedCourse := range courses {
		assignments, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.Assignment, error) {
			return s.assignmentGateway(client).Assignments(ctx, term.Value, selectedCourse.Course)
		})
		if err != nil {
			return nil, err
		}
		for _, assignment := range assignments {
			ref, err := NewCourseRef(term.Value, selectedCourse.Course)
			if err != nil {
				return nil, err
			}
			id, err := StableAssignmentID(ref, assignment.OrdSeq)
			if err != nil {
				return nil, err
			}
			rows = append(rows, AssignmentRow{
				ID:         id,
				LegacyID:   AssignmentID(selectedCourse.Index, assignment.OrdSeq),
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				DetailURL:  assignmentDetailURL(term.Value, selectedCourse.Course, assignment.OrdSeq),
				Assignment: assignmentModel(assignment),
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		left := rows[i].Assignment.DueAt
		right := rows[j].Assignment.DueAt
		if left == nil && right == nil {
			return rows[i].ID < rows[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return left.Before(*right)
	})

	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
	return rows, nil
}

func (s *Service) AssignmentDetail(ctx context.Context, id string, user UserOption) (AssignmentDetailResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return AssignmentDetailResult{}, err
	}

	locator, ordSeq, err := parseAssignmentResourceID(id)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	var term klas.Term
	if locator.Stable {
		term, client, err = s.termForSyllabus(ctx, studentID, client, locator.Ref.TermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	course, err := resolveResourceCourse(term, locator)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	ref, err := NewCourseRef(term.Value, course)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	canonicalID, err := StableAssignmentID(ref, ordSeq)
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	detail, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.AssignmentDetail, error) {
		return s.assignmentGateway(client).AssignmentDetail(ctx, term.Value, course, ordSeq)
	})
	if err != nil {
		return AssignmentDetailResult{}, err
	}
	if detail.DueAt == nil || detail.StartAt == nil {
		assignments, listErr := s.assignmentGateway(client).Assignments(ctx, term.Value, course)
		if listErr == nil {
			for _, assignment := range assignments {
				if strings.TrimSpace(assignment.OrdSeq) != ordSeq {
					continue
				}
				if detail.DueAt == nil {
					detail.DueAt = assignment.DueAt
				}
				if detail.StartAt == nil {
					detail.StartAt = assignment.StartAt
				}
				break
			}
		}
	}

	return AssignmentDetailResult{
		ID:         canonicalID,
		TermValue:  term.Value,
		CourseName: course.Name,
		DetailURL:  assignmentDetailURL(term.Value, course, ordSeq),
		Detail:     assignmentDetailModel(detail),
	}, nil
}

func (s *Service) AssignmentOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	detail, err := s.AssignmentDetail(ctx, id, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	return OpenURLResult{URL: detail.DetailURL}, nil
}

func assignmentDetailURL(yearHakgi string, course klas.Course, ordSeq string) string {
	values := url.Values{}
	values.Set("selectYearhakgi", yearHakgi)
	values.Set("selectSubj", course.Value)
	values.Set("ordseq", strings.TrimSpace(ordSeq))
	return "https://klas.kw.ac.kr/std/lis/evltn/TaskViewStdPage.do?" + values.Encode()
}

func assignmentRowIDs(rows []AssignmentRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func limitAssignments(rows []AssignmentRow, limit int) []AssignmentRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func AssignmentID(courseIndex int, ordSeq string) string {
	return fmt.Sprintf("%d:%s", courseIndex, strings.TrimSpace(ordSeq))
}

func StableAssignmentID(ref CourseRef, ordSeq string) (string, error) {
	return stableCourseResourceID("assignment", ref, ordSeq)
}

func parseAssignmentResourceID(id string) (courseResourceLocator, string, error) {
	ref, remoteParts, stable, err := parseStableCourseResourceID("assignment", id, 1)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	if stable {
		return courseResourceLocator{Ref: ref, Stable: true}, remoteParts[0], nil
	}
	courseIndex, ordSeq, err := ParseAssignmentID(id)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	return courseResourceLocator{CourseIndex: courseIndex}, ordSeq, nil
}

func normalizeCachedAssignmentRows(rows []AssignmentRow, term klas.Term) ([]AssignmentRow, bool, error) {
	migrated := false
	for index := range rows {
		locator, ordSeq, err := parseAssignmentResourceID(rows[index].ID)
		if err != nil {
			return nil, false, err
		}
		if locator.Stable {
			course, err := resolveResourceCourse(term, locator)
			if err != nil {
				return nil, false, err
			}
			for courseIndex, candidate := range term.Courses {
				if strings.TrimSpace(candidate.Value) == strings.TrimSpace(course.Value) {
					rows[index].LegacyID = AssignmentID(courseIndex+1, ordSeq)
					break
				}
			}
			continue
		}
		course, err := resolveLegacyCachedCourse(term, locator.CourseIndex, rows[index].CourseName)
		if err != nil {
			return nil, false, err
		}
		ref, err := NewCourseRef(term.Value, course)
		if err != nil {
			return nil, false, err
		}
		stableID, err := StableAssignmentID(ref, ordSeq)
		if err != nil {
			return nil, false, err
		}
		rows[index].LegacyID = rows[index].ID
		rows[index].ID = stableID
		rows[index].TermValue = term.Value
		migrated = true
	}
	return rows, migrated, nil
}

func ParseAssignmentID(id string) (int, string, error) {
	parts := strings.SplitN(strings.TrimSpace(id), ":", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("과제ID는 course list 번호와 ordseq를 조합한 <과목번호>:<ordseq> 형식이어야 합니다")
	}

	courseIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("과목 번호 파싱 실패: %w", err)
	}
	ordSeq := strings.TrimSpace(parts[1])
	if ordSeq == "" {
		return 0, "", errors.New("과제ID에 ordseq가 없습니다")
	}
	return courseIndex, ordSeq, nil
}
