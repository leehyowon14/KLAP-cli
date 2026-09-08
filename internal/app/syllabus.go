package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
	"time"
)

type SyllabusOptions struct {
	User      UserOption
	Selector  string
	TermValue string
}

type SubjectSearchOptions struct {
	Name      string
	Professor string
	TermValue string
}

type SyllabusResult struct {
	Term      Term
	SubjectID string
	Course    Course
	Syllabus  klas.Syllabus
}

type SubjectSearchResult struct {
	Term Term
	Rows []SubjectSearchRow
}

type SubjectSearchRow struct {
	CourseCode string
	SubjectID  string
	Name       string
	Professor  string
	Times      []klas.SyllabusTime
	Err        error
}

func (s *Service) Syllabus(ctx context.Context, opts SyllabusOptions) (SyllabusResult, error) {
	selector := strings.TrimSpace(opts.Selector)
	if selector == "" {
		return SyllabusResult{}, errors.New("강의계획서 조회 대상이 없습니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return SyllabusResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return SyllabusResult{}, err
	}

	term, client, err := s.termForSyllabus(ctx, studentID, client, opts.TermValue)
	if err != nil {
		return SyllabusResult{}, err
	}

	course := Course{Name: selector}
	subjectID := ""
	if looksLikeSyllabusCourseCode(selector) {
		subjectID, err = klas.SyllabusSubjectIDFromCourseCode(term.Value, selector)
		if err != nil {
			return SyllabusResult{}, err
		}
	} else {
		courses, err := selectedCourses(term, selector)
		if err != nil {
			return SyllabusResult{}, err
		}
		if len(courses) != 1 {
			return SyllabusResult{}, fmt.Errorf("강의계획서 조회 대상이 여러 개입니다: %s", selector)
		}
		course = courses[0].Course
		subjectID = course.Value
	}

	syllabus, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.Syllabus, error) {
		return client.SyllabusBySubjectID(ctx, subjectID)
	})
	if err != nil {
		return SyllabusResult{}, err
	}
	if strings.TrimSpace(course.Name) == "" || course.Name == selector {
		course.Name = firstNonEmpty(syllabus.KoreanName, syllabus.FullName, selector)
	}
	if strings.TrimSpace(course.Value) == "" {
		course.Value = subjectID
	}

	return SyllabusResult{
		Term:      term,
		SubjectID: subjectID,
		Course:    course,
		Syllabus:  syllabus,
	}, nil
}

func (s *Service) SubjectSearch(ctx context.Context, opts SubjectSearchOptions) (SubjectSearchResult, error) {
	name := strings.TrimSpace(opts.Name)
	professor := strings.TrimSpace(opts.Professor)
	if name == "" && professor == "" {
		return SubjectSearchResult{}, errors.New("과목명 또는 교수명을 입력해야 합니다")
	}

	studentID, err := s.selectedStudentID(ctx, UserOption{})
	if err != nil {
		return SubjectSearchResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return SubjectSearchResult{}, err
	}
	term, err := s.termForSubjectSearch(opts.TermValue)
	if err != nil {
		return SubjectSearchResult{}, err
	}

	items, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.SyllabusListItem, error) {
		return client.SyllabusList(ctx, term.Value, name, professor)
	})
	if err != nil {
		return SubjectSearchResult{}, err
	}

	rows := make([]SubjectSearchRow, 0, len(items))
	for _, item := range items {
		subjectID, idErr := item.SubjectID()
		row := SubjectSearchRow{
			CourseCode: item.CourseCode(),
			SubjectID:  subjectID,
			Name:       strings.TrimSpace(item.KoreanName),
			Professor:  strings.TrimSpace(item.Professor),
			Err:        idErr,
		}
		if idErr == nil {
			syllabus, detailErr := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.Syllabus, error) {
				return client.SyllabusBySubjectID(ctx, subjectID)
			})
			row.Err = detailErr
			if row.Err == nil {
				row.Times = syllabus.Times
				if strings.TrimSpace(row.Name) == "" {
					row.Name = firstNonEmpty(syllabus.KoreanName, syllabus.FullName)
				}
				if strings.TrimSpace(row.Professor) == "" {
					row.Professor = syllabus.Professor
				}
			}
		}
		rows = append(rows, row)
	}

	return SubjectSearchResult{
		Term: term,
		Rows: rows,
	}, nil
}

func (s *Service) termForSyllabus(ctx context.Context, studentID string, client *klas.Client, termValue string) (Term, *klas.Client, error) {
	termValue, err := normalizeTermValue(termValue)
	if err != nil {
		return Term{}, client, err
	}
	if termValue == "" {
		return s.selectedTerm(ctx, studentID, client)
	}

	terms, client, err := s.courses(ctx, studentID, client)
	if err != nil {
		return Term{}, client, err
	}
	for _, term := range terms {
		if term.Value == termValue {
			return term, client, nil
		}
	}
	return Term{}, client, fmt.Errorf("학기를 찾을 수 없습니다: %s", termValue)
}

func (s *Service) termForSubjectSearch(termValue string) (Term, error) {
	termValue, err := normalizeTermValue(termValue)
	if err != nil {
		return Term{}, err
	}
	if termValue == "" {
		current, err := s.loadSettings()
		if err != nil {
			return Term{}, err
		}
		termValue = strings.TrimSpace(current.Term.Value)
	}
	if termValue == "" {
		termValue = currentAcademicTermValue(time.Now())
	}
	return Term{
		Value: termValue,
		Label: termLabel(termValue),
	}, nil
}

func looksLikeSyllabusCourseCode(value string) bool {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}
	return true
}
