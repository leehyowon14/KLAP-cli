package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leehyowon14/KLAP-cli/internal/account"
)

type userListStub struct {
	AccountStore
	users               []account.User
	current             string
	listErr, currentErr error
}

func (s userListStub) List(context.Context) ([]account.User, error) { return s.users, s.listErr }
func (s userListStub) Current(context.Context) (string, error)      { return s.current, s.currentErr }

func TestUsersMapsStoreModels(t *testing.T) {
	users := []account.User{{StudentID: "student", SavedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), UserID: "remote"}, {StudentID: "other"}}
	s := &Service{store: userListStub{users: users, current: "student"}}
	rows, err := s.Users(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || !rows[0].Current || rows[1].Current {
		t.Fatalf("rows: %#v", rows)
	}
	for i := range users {
		assertModelJSONParity(t, users[i], rows[i].User)
	}
	assertModelJSONParity(t, account.User{}, userModel(account.User{}))
	for _, empty := range [][]account.User{nil, {}} {
		s.store = userListStub{users: empty}
		rows, err = s.Users(context.Background())
		if err != nil || rows == nil || len(rows) != 0 {
			t.Fatalf("empty rows=%#v err=%v", rows, err)
		}
	}
	wantErr := errors.New("store failure")
	for _, stub := range []userListStub{{listErr: wantErr}, {currentErr: wantErr}} {
		s.store = stub
		rows, err = s.Users(context.Background())
		if rows != nil || !errors.Is(err, wantErr) {
			t.Fatalf("failure rows=%#v err=%v", rows, err)
		}
	}
}
