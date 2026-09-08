package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
	"testing"
)

type fakeSessionStore struct {
	loadPassword func(context.Context, string) (string, error)
	loadSession  func(context.Context, string) (klas.Session, error)
	saveSession  func(context.Context, string, klas.Session) error
}

func (s *fakeSessionStore) LoadPassword(ctx context.Context, studentID string) (string, error) {
	return s.loadPassword(ctx, studentID)
}

func (s *fakeSessionStore) LoadSession(ctx context.Context, studentID string) (klas.Session, error) {
	return s.loadSession(ctx, studentID)
}

func (s *fakeSessionStore) SaveSession(ctx context.Context, studentID string, session klas.Session) error {
	return s.saveSession(ctx, studentID, session)
}

func TestAuthenticatedClientUsesStoredSessionWithoutSaving(t *testing.T) {
	storedSession := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "saved"}}
	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	saveCalls := 0
	loginCalls := 0
	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return storedSession, nil
			},
			loadPassword: func(context.Context, string) (string, error) {
				return "", errors.New("LoadPassword should not be called")
			},
			saveSession: func(context.Context, string, klas.Session) error {
				saveCalls++
				return nil
			},
		},
		newKlasClient: func() (*klas.Client, error) { return client, nil },
		login: func(context.Context, *klas.Client, string, string) (klas.Session, error) {
			loginCalls++
			return klas.Session{}, nil
		},
	}

	got, err := service.authenticatedClient(context.Background(), "20260001")
	if err != nil {
		t.Fatalf("authenticatedClient() error = %v", err)
	}
	if got != client {
		t.Fatalf("authenticatedClient() client = %p, want %p", got, client)
	}
	if saveCalls != 0 || loginCalls != 0 {
		t.Fatalf("authenticatedClient() save calls = %d, login calls = %d", saveCalls, loginCalls)
	}
}

func TestAuthenticatedClientPropagatesRefreshedSessionSaveFailure(t *testing.T) {
	wantErr := errors.New("keychain unavailable")
	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	refreshedSession := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "refreshed"}}
	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return klas.Session{}, errors.New("stored session missing")
			},
			loadPassword: func(_ context.Context, studentID string) (string, error) {
				if studentID != "20260001" {
					t.Fatalf("LoadPassword() studentID = %q", studentID)
				}
				return "password", nil
			},
			saveSession: func(_ context.Context, studentID string, session klas.Session) error {
				if studentID != "20260001" || session.UserID != refreshedSession.UserID {
					t.Fatalf("SaveSession() = %q, %+v", studentID, session)
				}
				return wantErr
			},
		},
		newKlasClient: func() (*klas.Client, error) { return client, nil },
		login: func(_ context.Context, gotClient *klas.Client, studentID string, password string) (klas.Session, error) {
			if gotClient != client || studentID != "20260001" || password != "password" {
				t.Fatalf("login() = %p, %q, %q", gotClient, studentID, password)
			}
			return refreshedSession, nil
		},
	}

	got, err := service.authenticatedClient(context.Background(), "20260001")
	if got != nil {
		t.Fatalf("authenticatedClient() client = %p, want nil", got)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "갱신 세션 저장 실패") {
		t.Fatalf("authenticatedClient() error = %v", err)
	}
}

func TestRefreshedClientPropagatesSessionSaveFailure(t *testing.T) {
	wantErr := errors.New("keychain unavailable")
	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	refreshedSession := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "refreshed"}}
	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return klas.Session{}, errors.New("LoadSession should not be called")
			},
			loadPassword: func(context.Context, string) (string, error) {
				return "password", nil
			},
			saveSession: func(context.Context, string, klas.Session) error {
				return wantErr
			},
		},
		newKlasClient: func() (*klas.Client, error) { return client, nil },
		login: func(context.Context, *klas.Client, string, string) (klas.Session, error) {
			return refreshedSession, nil
		},
	}

	got, refreshed, err := service.refreshedClientAfterSessionError(context.Background(), "20260001", klas.ErrSessionExpired)
	if got != nil {
		t.Fatalf("refreshedClientAfterSessionError() client = %p, want nil", got)
	}
	if !refreshed {
		t.Fatal("refreshedClientAfterSessionError() refreshed = false, want true")
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "갱신 세션 저장 실패") {
		t.Fatalf("refreshedClientAfterSessionError() error = %v", err)
	}
}
