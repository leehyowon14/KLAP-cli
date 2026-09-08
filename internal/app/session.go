package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

type sessionStore interface {
	LoadPassword(context.Context, string) (string, error)
	LoadSession(context.Context, string) (klas.Session, error)
	SaveSession(context.Context, string, klas.Session) error
}

// executeSessionRequest retries only the failed request, at most once. Updating
// client lets subsequent requests in the same use case reuse the saved session.
func executeSessionRequest[T any](ctx context.Context, s *Service, studentID string, client **klas.Client, request func(*klas.Client) (T, error)) (T, error) {
	result, err := request(*client)
	if !errors.Is(err, klas.ErrSessionExpired) {
		return result, err
	}
	refreshed, _, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
	if refreshErr != nil {
		return result, refreshErr
	}
	*client = refreshed
	if s.requestSession != nil {
		s.requestSession.mu.Lock()
		s.requestSession.clients[studentID] = refreshed
		s.requestSession.mu.Unlock()
	}
	return request(refreshed)
}

func (s *Service) authenticatedClient(ctx context.Context, studentID string) (*klas.Client, error) {
	if scope := s.requestSession; scope != nil {
		scope.mu.Lock()
		defer scope.mu.Unlock()
		if client := scope.clients[studentID]; client != nil {
			return client, nil
		}
		client, err := s.newAuthenticatedClient(ctx, studentID)
		if err == nil {
			scope.clients[studentID] = client
		}
		return client, err
	}
	return s.newAuthenticatedClient(ctx, studentID)
}

func (s *Service) newAuthenticatedClient(ctx context.Context, studentID string) (*klas.Client, error) {
	client, err := s.newKlasClient()
	if err != nil {
		return nil, err
	}

	session, err := s.sessions.LoadSession(ctx, studentID)
	if err == nil {
		client.SetSession(session)
		return client, nil
	}

	password, err := s.sessions.LoadPassword(ctx, studentID)
	if err != nil {
		return nil, err
	}
	session, err = s.login(ctx, client, studentID, password)
	if err != nil {
		return nil, fmt.Errorf("재로그인 실패: %w", err)
	}
	if err := s.sessions.SaveSession(ctx, studentID, session); err != nil {
		return nil, fmt.Errorf("갱신 세션 저장 실패: %w", err)
	}
	return client, nil
}

func (s *Service) refreshedClientAfterSessionError(ctx context.Context, studentID string, err error) (*klas.Client, bool, error) {
	if !errors.Is(err, klas.ErrSessionExpired) {
		return nil, false, err
	}

	client, clientErr := s.newKlasClient()
	if clientErr != nil {
		return nil, true, clientErr
	}
	password, passwordErr := s.sessions.LoadPassword(ctx, studentID)
	if passwordErr != nil {
		return nil, true, passwordErr
	}
	session, loginErr := s.login(ctx, client, studentID, password)
	if loginErr != nil {
		return nil, true, fmt.Errorf("재로그인 실패: %w", loginErr)
	}
	if saveErr := s.sessions.SaveSession(ctx, studentID, session); saveErr != nil {
		return nil, true, fmt.Errorf("갱신 세션 저장 실패: %w", saveErr)
	}
	return client, true, nil
}
