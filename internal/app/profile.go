package app

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
)

// StudentName reads the signed-in student's academic profile without submitting an evaluation.
func (s *Service) StudentName(ctx context.Context, user UserOption) (string, error) {
	id, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return "", err
	}
	client, err := s.authenticatedClient(ctx, id)
	if err != nil {
		return "", err
	}
	return executeSessionRequest(ctx, s, id, &client, func(c *klas.Client) (string, error) {
		profile, err := c.EvaluationStudent(ctx)
		if err != nil {
			return "", err
		}
		return checkedStudentName(id, profile.StudentID, profile.Name)
	})
}
func checkedStudentName(expected, actual, name string) (string, error) {
	if expected == "" || strings.TrimSpace(actual) != expected {
		return "", fmt.Errorf("학적 정보의 계정이 일치하지 않습니다")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("학적 정보에 이름이 없습니다")
	}
	return name, nil
}
