package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"strings"
)

type UserOption struct {
	StudentID string
}

type UserRow struct {
	User    account.User
	Current bool
}

func (s *Service) Authenticate(ctx context.Context, studentID string, password string) error {
	if strings.TrimSpace(studentID) == "" || strings.TrimSpace(password) == "" {
		return errors.New("학번과 비밀번호를 모두 입력해야 합니다")
	}

	client, err := s.newKlasClient()
	if err != nil {
		return err
	}

	session, err := s.login(ctx, client, studentID, password)
	if err != nil {
		return fmt.Errorf("로그인 검증 실패: %w", err)
	}

	if err := s.store.Save(ctx, studentID, password, session); err != nil {
		return fmt.Errorf("계정 저장 실패: %w", err)
	}
	return nil
}

func (s *Service) Users(ctx context.Context) ([]UserRow, error) {
	users, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	currentStudentID, err := s.store.Current(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]UserRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, UserRow{
			User:    user,
			Current: user.StudentID == currentStudentID,
		})
	}
	return rows, nil
}

func (s *Service) SelectUser(ctx context.Context, studentID string) error {
	return s.store.Select(ctx, studentID)
}

func (s *Service) RemoveUser(ctx context.Context, studentID string) error {
	return s.store.Remove(ctx, studentID)
}

func (s *Service) selectedStudentID(ctx context.Context, user UserOption) (string, error) {
	if strings.TrimSpace(user.StudentID) != "" {
		return strings.TrimSpace(user.StudentID), nil
	}

	currentStudentID, err := s.store.Current(ctx)
	if err != nil {
		return "", err
	}
	if currentStudentID != "" {
		return currentStudentID, nil
	}

	users, err := s.store.List(ctx)
	if err != nil {
		return "", err
	}
	switch len(users) {
	case 0:
		return "", errors.New("저장된 유저가 없습니다. 먼저 klap auth를 실행하세요")
	case 1:
		return users[0].StudentID, nil
	default:
		return "", errors.New("저장된 유저가 여러 명입니다. klap user select <학번> 또는 --user <학번>을 지정하세요")
	}
}
