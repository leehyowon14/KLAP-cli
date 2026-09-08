package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strconv"
	"strings"
)

type EvaluationListOptions struct {
	User    UserOption
	Refresh bool
}

type EvaluationSubmitOptions struct {
	User               UserOption
	Selector           string
	Confirm            bool
	IncludeEngineering bool
}

type EvaluationListResult struct {
	Term EvaluationTerm
	Rows []EvaluationRow
}

type EvaluationRow struct {
	Index  int
	Course EvaluationCourse
}

type EvaluationSubmitResult struct {
	Term      EvaluationTerm
	Items     []EvaluationSubmitItem
	Submitted bool
}

type EvaluationSubmitItem struct {
	Row     EvaluationRow
	Skipped bool
	Reason  string
	Err     error
}

func (s *Service) EvaluationList(ctx context.Context, opts EvaluationListOptions) (EvaluationListResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return EvaluationListResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return EvaluationListResult{}, err
	}

	cacheKey := listCacheKeyVersion("evaluation", "v2", studentID, "", "")
	if !opts.Refresh {
		var cached EvaluationListResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	term, courses, _, err := s.evaluationCourses(ctx, studentID, client)
	if err != nil {
		return EvaluationListResult{}, err
	}
	rows := makeEvaluationRows(courses)
	result := EvaluationListResult{Term: evaluationTermModel(term), Rows: rows}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) EvaluationSubmit(ctx context.Context, opts EvaluationSubmitOptions) (EvaluationSubmitResult, error) {
	selector := strings.TrimSpace(opts.Selector)
	if selector == "" {
		return EvaluationSubmitResult{}, errors.New("수업평가 제출 대상이 없습니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}

	term, courses, client, err := s.evaluationCourses(ctx, studentID, client)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}
	targets, err := selectEvaluationRows(makeEvaluationRows(courses), selector)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}

	result := EvaluationSubmitResult{
		Term:      evaluationTermModel(term),
		Submitted: opts.Confirm,
		Items:     make([]EvaluationSubmitItem, 0, len(targets)),
	}
	answerOpts := klas.EvaluationAnswerOptions{
		Choice:             "5",
		Text:               "많은 도움 되었습니다. 한학기동안 감사했습니다.",
		DiscriminationOpt:  "N",
		IncludeEngineering: opts.IncludeEngineering,
	}
	for _, target := range targets {
		// The selection keeps the original one-based position; use the freshly
		// fetched adapter course for submission, never a presentation/cache DTO.
		course := courses[target.Index-1]
		item := EvaluationSubmitItem{Row: target}
		if target.Course.Evaluated {
			item.Skipped = true
			item.Reason = "이미 평가 완료"
			result.Items = append(result.Items, item)
			continue
		}

		form, formErr := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.EvaluationForm, error) {
			return client.EvaluationForm(ctx, term, course)
		})
		if formErr != nil {
			item.Err = formErr
			result.Items = append(result.Items, item)
			continue
		}
		if !opts.Confirm {
			result.Items = append(result.Items, item)
			continue
		}
		_, submitErr := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.EvaluationSubmitResult, error) {
			return client.SubmitEvaluation(ctx, term, course, form, answerOpts)
		})
		item.Err = submitErr
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (s *Service) evaluationCourses(ctx context.Context, studentID string, client *klas.Client) (klas.EvaluationTerm, []klas.EvaluationCourse, *klas.Client, error) {
	term, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (klas.EvaluationTerm, error) {
		return client.EvaluationTerm(ctx)
	})
	if err != nil {
		return klas.EvaluationTerm{}, nil, client, err
	}
	if !term.TermEnabled {
		return term, nil, client, nil
	}

	if _, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) (struct{}, error) {
		_, err := client.EvaluationStudent(ctx)
		return struct{}{}, err
	}); err != nil {
		return klas.EvaluationTerm{}, nil, client, err
	}

	courses, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.EvaluationCourse, error) {
		return client.EvaluationCourses(ctx, term)
	})
	if err != nil {
		return klas.EvaluationTerm{}, nil, client, err
	}
	return term, courses, client, nil
}

func makeEvaluationRows(courses []klas.EvaluationCourse) []EvaluationRow {
	rows := make([]EvaluationRow, 0, len(courses))
	for index, course := range courses {
		rows = append(rows, EvaluationRow{
			Index:  index + 1,
			Course: evaluationCourseModel(course),
		})
	}
	return rows
}

func selectEvaluationRows(rows []EvaluationRow, selector string) ([]EvaluationRow, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, errors.New("수업평가 제출 대상이 없습니다")
	}
	if strings.EqualFold(selector, "all") {
		return rows, nil
	}
	if number, err := strconv.Atoi(selector); err == nil {
		if number < 1 || number > len(rows) {
			return nil, fmt.Errorf("수업평가 과목 번호가 범위를 벗어났습니다: %d", number)
		}
		return []EvaluationRow{rows[number-1]}, nil
	}

	normalizedSelector := strings.ToLower(selector)
	var matches []EvaluationRow
	for _, row := range rows {
		name := strings.ToLower(strings.TrimSpace(row.Course.Name))
		professor := strings.ToLower(strings.TrimSpace(row.Course.Professor))
		if name == normalizedSelector {
			return []EvaluationRow{row}, nil
		}
		if strings.Contains(name, normalizedSelector) || strings.Contains(professor, normalizedSelector) {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("수업평가 과목이 여러 개와 일치합니다. evaluation list 번호를 사용하세요: %s", selector)
	}
	return nil, fmt.Errorf("수업평가 과목을 찾을 수 없습니다: %s", selector)
}
