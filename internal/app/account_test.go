package app

import (
	"context"
	"errors"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

type accountSaveStub struct {
	AccountStore
	save func(context.Context, string, string, klas.Session) error
}

func (s accountSaveStub) Save(ctx context.Context, id, password string, session klas.Session) error {
	return s.save(ctx, id, password, session)
}

func TestAuthenticateUsesInjectedDependencies(t *testing.T) {
	wantErr := errors.New("injected failure")
	for _, stage := range []string{"client", "login", "save", "success"} {
		t.Run(stage, func(t *testing.T) {
			client := &klas.Client{}
			clientCalls, loginCalls, saveCalls := 0, 0, 0
			s := &Service{
				newKlasClient: func() (*klas.Client, error) {
					clientCalls++
					if stage == "client" {
						return nil, wantErr
					}
					return client, nil
				},
				login: func(_ context.Context, c *klas.Client, id, password string) (klas.Session, error) {
					loginCalls++
					if c != client || id != "student" || password != "password" {
						t.Fatal("login arguments changed")
					}
					if stage == "login" {
						return klas.Session{}, wantErr
					}
					return klas.Session{}, nil
				},
				store: accountSaveStub{save: func(_ context.Context, id, password string, _ klas.Session) error {
					saveCalls++
					if id != "student" || password != "password" {
						t.Fatal("save arguments changed")
					}
					if stage == "save" {
						return wantErr
					}
					return nil
				}},
			}
			err := s.Authenticate(context.Background(), "student", "password")
			if stage == "success" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, wantErr) {
				t.Fatalf("error=%v", err)
			}
			wantLogin, wantSave := 1, 1
			if stage == "client" {
				wantLogin = 0
				wantSave = 0
			}
			if stage == "login" {
				wantSave = 0
			}
			if clientCalls != 1 || loginCalls != wantLogin || saveCalls != wantSave {
				t.Fatalf("calls=%d/%d/%d", clientCalls, loginCalls, saveCalls)
			}
		})
	}
}

func TestAuthenticateRejectsBlankCredentialsBeforeIO(t *testing.T) {
	for _, values := range [][2]string{{"", "password"}, {"student", " "}, {" ", ""}} {
		s := &Service{newKlasClient: func() (*klas.Client, error) { t.Fatal("client factory called for blank credentials"); return nil, nil }}
		if err := s.Authenticate(context.Background(), values[0], values[1]); err == nil {
			t.Fatal("expected validation error")
		}
	}
}
