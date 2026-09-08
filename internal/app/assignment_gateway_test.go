package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
)

type assignmentGatewayStub struct {
	AssignmentGateway
	list func(context.Context, string, klas.Course) ([]klas.Assignment, error)
}

func (g assignmentGatewayStub) Assignments(ctx context.Context, term string, course klas.Course) ([]klas.Assignment, error) {
	return g.list(ctx, term, course)
}

type readSettingsStub struct{ SettingsStore }

func (readSettingsStub) Load() (settings.Settings, error) { return settings.Default(), nil }

func TestAssignmentListUsesInjectedGateway(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/std/cmn/frame/YearhakgiAtnlcSbjectList.do" {
			t.Fatalf("unexpected network request: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"value":"2026,1","label":"term","subjList":[{"value":"subject","name":"course"}]}]`))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	store, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	deps := testDependencies(t)
	deps.Sessions = &fakeSessionStore{loadSession: func(context.Context, string) (klas.Session, error) { return klas.Session{}, nil }}
	deps.Settings = readSettingsStub{}
	deps.Cache = store
	deps.NewKlasClient = func() (*klas.Client, error) { return client, nil }
	calls := 0
	deps.AssignmentGateway = func(c *klas.Client) AssignmentGateway {
		if c != client {
			t.Fatal("gateway received wrong session client")
		}
		return assignmentGatewayStub{list: func(_ context.Context, term string, course klas.Course) ([]klas.Assignment, error) {
			calls++
			if term != "2026,1" || course.Value != "subject" {
				t.Fatal("gateway arguments changed")
			}
			return []klas.Assignment{{OrdSeq: "42", Title: "injected"}}, nil
		}}
	}
	s, err := NewService(deps)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.AssignmentList(context.Background(), AssignmentListOptions{User: UserOption{StudentID: "student"}, Refresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(rows) != 1 || rows[0].Assignment.Title != "injected" || rows[0].LegacyID != "1:42" {
		t.Fatalf("calls=%d rows=%+v", calls, rows)
	}
}

func TestDependenciesRequireAssignmentGateway(t *testing.T) {
	deps := testDependencies(t)
	deps.AssignmentGateway = nil
	if s, err := NewService(deps); s != nil || err == nil {
		t.Fatalf("service=%v error=%v", s, err)
	}
}
