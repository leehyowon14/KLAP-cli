package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestExecuteSessionRequest(t *testing.T) {
	networkErr := errors.New("network")
	saveErr := errors.New("save")
	for _, tt := range []struct {
		name                      string
		first, second, save, want error
		calls, logins             int
	}{
		{name: "normal", calls: 1},
		{name: "expired then success", first: fmt.Errorf("wrapped: %w", klas.ErrSessionExpired), calls: 2, logins: 1},
		{name: "second expiry", first: klas.ErrSessionExpired, second: klas.ErrSessionExpired, want: klas.ErrSessionExpired, calls: 2, logins: 1},
		{name: "non session failure", first: networkErr, want: networkErr, calls: 1},
		{name: "save failure prevents retry", first: klas.ErrSessionExpired, save: saveErr, want: saveErr, calls: 1, logins: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			initial, refreshed := &klas.Client{}, &klas.Client{}
			client := initial
			calls, logins, saves := 0, 0, 0
			s := &Service{
				newKlasClient: func() (*klas.Client, error) { return refreshed, nil },
				sessions: &fakeSessionStore{
					loadPassword: func(context.Context, string) (string, error) { return "password", nil },
					saveSession:  func(context.Context, string, klas.Session) error { saves++; return tt.save },
				},
				login: func(context.Context, *klas.Client, string, string) (klas.Session, error) {
					logins++
					return klas.Session{}, nil
				},
			}
			got, err := executeSessionRequest(context.Background(), s, "student", &client, func(c *klas.Client) (int, error) {
				calls++
				if calls == 1 {
					if c != initial {
						t.Fatal("initial client replaced before request")
					}
					return calls, tt.first
				}
				if c != refreshed {
					t.Fatal("retry did not use refreshed client")
				}
				return calls, tt.second
			})
			if !errors.Is(err, tt.want) || got != tt.calls || calls != tt.calls || logins != tt.logins || saves != tt.logins {
				t.Fatalf("result=%d error=%v calls=%d logins=%d saves=%d", got, err, calls, logins, saves)
			}
			if (client == refreshed) != (tt.calls == 2) {
				t.Fatal("client reuse state mismatch")
			}
		})
	}
}
