package core_test

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/pkg/core"
	"testing"
)

type memoryCredentials struct{}

func (memoryCredentials) Save(context.Context, string, string, core.Session) error { return nil }
func (memoryCredentials) List(context.Context) ([]core.SavedUser, error) {
	return []core.SavedUser{{StudentID: "test"}}, nil
}
func (memoryCredentials) Current(context.Context) (string, error)              { return "test", nil }
func (memoryCredentials) Select(context.Context, string) error                 { return nil }
func (memoryCredentials) Remove(context.Context, string) error                 { return nil }
func (memoryCredentials) LoadPassword(context.Context, string) (string, error) { return "", nil }
func (memoryCredentials) LoadSession(context.Context, string) (core.Session, error) {
	return core.Session{}, nil
}
func (memoryCredentials) SaveSession(context.Context, string, core.Session) error { return nil }
func TestPublicClientCanInjectCredentialsWithoutKeychain(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	t.Setenv("KLAP_CACHE_DIR", t.TempDir())
	s, err := core.NewWithCredentials(memoryCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Users(context.Background())
	if err != nil || len(rows) != 1 || rows[0].User.StudentID != "test" {
		t.Fatalf("%v %v", rows, err)
	}
	if _, err := core.NewWithCredentials(nil); err == nil {
		t.Fatal("nil credentials accepted")
	}
}
