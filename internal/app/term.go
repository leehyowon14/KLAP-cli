package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strconv"
	"strings"
	"time"
)

type TermListOptions struct {
	User UserOption
}

type TermRow struct {
	Index   int
	Term    klas.Term
	Current bool
}

type TermSettings struct {
	Value string
	Label string
}

func (s *Service) TermList(ctx context.Context, opts TermListOptions) ([]TermRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, err
	}

	terms, _, err := s.courses(ctx, studentID, client)
	if err != nil {
		return nil, err
	}
	current, err := s.loadSettings()
	if err != nil {
		return nil, err
	}

	rows := make([]TermRow, 0, len(terms))
	for index, term := range terms {
		rows = append(rows, TermRow{
			Index:   index + 1,
			Term:    term,
			Current: current.Term.Value != "" && term.Value == current.Term.Value,
		})
	}
	if current.Term.Value == "" && len(rows) > 0 {
		rows[0].Current = true
	}
	return rows, nil
}

func (s *Service) SelectTerm(ctx context.Context, selector string, user UserOption) (TermSettings, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return TermSettings{}, errors.New("학기 번호 또는 값을 입력해야 합니다")
	}

	rows, err := s.TermList(ctx, TermListOptions{User: user})
	if err != nil {
		return TermSettings{}, err
	}
	if len(rows) == 0 {
		return TermSettings{}, errors.New("선택할 학기가 없습니다")
	}

	matched, err := selectTermRow(rows, selector)
	if err != nil {
		return TermSettings{}, err
	}

	current, err := s.loadSettings()
	if err != nil {
		return TermSettings{}, err
	}
	current.Term.Value = matched.Term.Value
	if err := s.saveSettings(current); err != nil {
		return TermSettings{}, err
	}
	return TermSettings{Value: matched.Term.Value, Label: matched.Term.Label}, nil
}

func selectTermRow(rows []TermRow, selector string) (TermRow, error) {
	selector = strings.TrimSpace(selector)
	if number, err := strconv.Atoi(selector); err == nil {
		if number < 1 || number > len(rows) {
			return TermRow{}, fmt.Errorf("학기 번호가 범위를 벗어났습니다: %d", number)
		}
		return rows[number-1], nil
	}

	var matched *TermRow
	for index := range rows {
		if rows[index].Term.Value == selector || strings.Contains(rows[index].Term.Label, selector) {
			if matched != nil {
				return TermRow{}, fmt.Errorf("학기명이 여러 개와 일치합니다. term list 번호를 사용하세요: %s", selector)
			}
			matched = &rows[index]
		}
	}
	if matched == nil {
		return TermRow{}, fmt.Errorf("학기를 찾을 수 없습니다: %s", selector)
	}
	return *matched, nil
}

func (s *Service) latestTerm(ctx context.Context, studentID string) (*klas.Client, klas.Term, error) {
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, klas.Term{}, err
	}
	term, client, err := s.selectedTerm(ctx, studentID, client)
	return client, term, err
}

func (s *Service) selectedTerm(ctx context.Context, studentID string, client *klas.Client) (klas.Term, *klas.Client, error) {
	if scope := s.requestSession; scope != nil {
		scope.mu.Lock()
		term, ok := scope.terms[studentID]
		scope.mu.Unlock()
		if ok {
			return term, client, nil
		}
	}
	term, client, err := s.uncachedSelectedTerm(ctx, studentID, client)
	if scope := s.requestSession; scope != nil && err == nil {
		scope.mu.Lock()
		scope.terms[studentID] = term
		scope.mu.Unlock()
	}
	return term, client, err
}

func (s *Service) uncachedSelectedTerm(ctx context.Context, studentID string, client *klas.Client) (klas.Term, *klas.Client, error) {
	terms, client, err := s.courses(ctx, studentID, client)
	if err != nil {
		return klas.Term{}, client, err
	}
	if len(terms) == 0 {
		return klas.Term{}, client, errors.New("수강 학기가 없습니다")
	}

	current, err := s.loadSettings()
	if err != nil {
		return klas.Term{}, client, err
	}
	if strings.TrimSpace(current.Term.Value) == "" {
		return terms[0], client, nil
	}
	for _, term := range terms {
		if term.Value == current.Term.Value {
			return term, client, nil
		}
	}
	return klas.Term{}, client, fmt.Errorf("선택된 학기를 현재 유저에서 찾을 수 없습니다: %s", current.Term.Value)
}

func normalizeTermValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(parts) != 2 {
		return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
	}
	year := strings.TrimSpace(parts[0])
	semester := strings.TrimSpace(parts[1])
	if len(year) != 4 {
		return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
	}
	for _, r := range year + semester {
		if r < '0' || r > '9' {
			return "", errors.New("학기 값은 YYYY-S 형식이어야 합니다")
		}
	}
	if semester < "1" || semester > "4" {
		return "", errors.New("학기는 1, 2, 3, 4 중 하나여야 합니다")
	}
	return year + "," + semester, nil
}

func currentAcademicTermValue(now time.Time) string {
	year := now.Year()
	switch now.Month() {
	case time.January, time.February:
		return fmt.Sprintf("%d,4", year-1)
	case time.March, time.April, time.May, time.June:
		return fmt.Sprintf("%d,1", year)
	case time.July, time.August:
		return fmt.Sprintf("%d,3", year)
	case time.December:
		return fmt.Sprintf("%d,4", year)
	default:
		return fmt.Sprintf("%d,2", year)
	}
}

func termLabel(termValue string) string {
	parts := strings.Split(strings.TrimSpace(termValue), ",")
	if len(parts) != 2 {
		return strings.TrimSpace(termValue)
	}
	switch parts[1] {
	case "1":
		return fmt.Sprintf("%s년도 1학기", parts[0])
	case "2":
		return fmt.Sprintf("%s년도 2학기", parts[0])
	case "3":
		return fmt.Sprintf("%s년도 여름학기", parts[0])
	case "4":
		return fmt.Sprintf("%s년도 겨울학기", parts[0])
	default:
		return strings.TrimSpace(termValue)
	}
}
