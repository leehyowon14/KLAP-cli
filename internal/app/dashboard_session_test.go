package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

type aggregateSessionCounts struct{ users, clients, sessions, courses int }

func aggregateSessionFixture(t *testing.T) (*Service, *aggregateSessionCounts) {
	t.Helper()
	counts := &aggregateSessionCounts{}
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"errorCount":1}`
		if req.URL.Path == "/std/cmn/frame/YearhakgiAtnlcSbjectList.do" {
			counts.courses++
			body = `[{"value":"2026,1","label":"term","subjList":[{"value":"subject","name":"course"}]}]`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	deps := testDependencies(t)
	deps.Academic = &fakeAcademicSource{fetch: func(context.Context, string) (AcademicListResult, error) {
		return AcademicListResult{}, errors.New("fixture academic section unavailable")
	}}
	deps.Accounts = currentAccountStub{current: func() (string, error) { counts.users++; return "student", nil }}
	deps.Sessions = &fakeSessionStore{loadSession: func(context.Context, string) (klas.Session, error) { counts.sessions++; return klas.Session{}, nil }}
	deps.NewKlasClient = func() (*klas.Client, error) { counts.clients++; return klas.NewClient() }
	deps.Settings = readSettingsStub{}
	store, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	deps.Cache = store
	deps.AssignmentGateway = func(*klas.Client) AssignmentGateway {
		return assignmentGatewayStub{list: func(context.Context, string, Course) ([]klas.Assignment, error) { return nil, nil }}
	}
	s, err := NewService(deps)
	if err != nil {
		t.Fatal(err)
	}
	return s, counts
}

func assertAggregateSessionCounts(t *testing.T, counts *aggregateSessionCounts, want int) {
	t.Helper()
	if counts.users != want || counts.clients != want || counts.sessions != want || counts.courses != want {
		t.Fatalf("counts=%+v want=%d", counts, want)
	}
}

func TestDashboardSharesSessionAndPreservesPartialErrors(t *testing.T) {
	s, counts := aggregateSessionFixture(t)
	for run := 1; run <= 2; run++ {
		result, err := s.Dashboard(context.Background(), DashboardOptions{Refresh: true})
		if err != nil || len(result.SectionErrors) != 4 {
			t.Fatalf("error=%v sections=%+v", err, result.SectionErrors)
		}
		assertAggregateSessionCounts(t, counts, run)
	}
}
