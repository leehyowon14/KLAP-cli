package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

type currentAccountStub struct {
	AccountStore
	current func() (string, error)
}

func (s currentAccountStub) Current(context.Context) (string, error) { return s.current() }

func TestRequestSessionReusesLookupsAndRefreshedClient(t *testing.T) {
	ctx := context.Background()
	currentCalls, clients, loads, courseCalls, saves := 0, 0, 0, 0, 0
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/std/cmn/frame/YearhakgiAtnlcSbjectList.do" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		courseCalls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"value":"2026,1","label":"term","subjList":[]}]`))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	s := &Service{
		store:         currentAccountStub{current: func() (string, error) { currentCalls++; return "student", nil }},
		settingsStore: readSettingsStub{},
		newKlasClient: func() (*klas.Client, error) { clients++; return klas.NewClient() },
		sessions: &fakeSessionStore{
			loadSession:  func(context.Context, string) (klas.Session, error) { loads++; return klas.Session{}, nil },
			loadPassword: func(context.Context, string) (string, error) { return "password", nil },
			saveSession:  func(context.Context, string, klas.Session) error { saves++; return nil },
		},
		login: func(context.Context, *klas.Client, string, string) (klas.Session, error) { return klas.Session{}, nil },
	}
	query := s.withRequestSession()
	if query.withRequestSession() != query {
		t.Fatal("nested query discarded its scope")
	}
	for range 2 {
		student, err := query.selectedStudentID(ctx, UserOption{})
		if err != nil || student != "student" {
			t.Fatalf("student=%s err=%v", student, err)
		}
		_, term, err := query.latestTerm(ctx, student)
		if err != nil || term.Value != "2026,1" {
			t.Fatalf("term=%v err=%v", term, err)
		}
	}
	if currentCalls != 1 || clients != 1 || loads != 1 || courseCalls != 1 {
		t.Fatalf("current=%d clients=%d loads=%d courses=%d", currentCalls, clients, loads, courseCalls)
	}
	client, _ := query.authenticatedClient(ctx, "student")
	if _, _, err := query.selectedTerm(ctx, "student", client); err != nil || courseCalls != 1 {
		t.Fatalf("direct selectedTerm missed memo: error=%v courses=%d", err, courseCalls)
	}
	initial := client
	calls := 0
	_, err := executeSessionRequest(ctx, query, "student", &client, func(*klas.Client) (struct{}, error) {
		calls++
		if calls == 1 {
			return struct{}{}, klas.ErrSessionExpired
		}
		return struct{}{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	reused, _, err := query.latestTerm(ctx, "student")
	if err != nil || reused != client || reused == initial || saves != 1 || courseCalls != 1 {
		t.Fatalf("refreshed client not reused: err=%v saves=%d courses=%d", err, saves, courseCalls)
	}
	other, err := query.authenticatedClient(ctx, "other")
	if err != nil || other == client {
		t.Fatal("different users shared client")
	}
	next := s.withRequestSession()
	_, _, err = next.latestTerm(ctx, "student")
	if err != nil || next == query || s.requestSession != nil || courseCalls != 2 || loads != 3 {
		t.Fatalf("scope leaked between requests: err=%v courses=%d loads=%d", err, courseCalls, loads)
	}
}

func TestRequestSessionDoesNotCacheSelectionFailure(t *testing.T) {
	failure := errors.New("store unavailable")
	calls := 0
	s := (&Service{store: currentAccountStub{current: func() (string, error) {
		calls++
		if calls == 1 {
			return "", failure
		}
		return "student", nil
	}}}).withRequestSession()
	if _, err := s.selectedStudentID(context.Background(), UserOption{}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if student, err := s.selectedStudentID(context.Background(), UserOption{}); err != nil || student != "student" {
		t.Fatalf("student=%s err=%v", student, err)
	}
	if student, err := s.selectedStudentID(context.Background(), UserOption{StudentID: "other"}); err != nil || student != "other" {
		t.Fatal("explicit user overridden")
	}
}
