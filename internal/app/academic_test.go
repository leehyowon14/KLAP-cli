package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"reflect"
	"testing"
)

type fakeAcademicSource struct {
	fetch func(context.Context, string) (AcademicListResult, error)
}

func (f *fakeAcademicSource) FetchAcademic(ctx context.Context, year string) (AcademicListResult, error) {
	return f.fetch(ctx, year)
}

func TestDependenciesRequireAcademic(t *testing.T) {
	for _, source := range []AcademicSource{nil, (*fakeAcademicSource)(nil)} {
		deps := testDependencies(t)
		deps.Academic = source
		if s, err := NewService(deps); s != nil || err == nil {
			t.Fatalf("service=%v err=%v", s, err)
		}
	}
}

func TestAcademicQueryCacheRefreshAndFailure(t *testing.T) {
	store, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	calls := 0
	expected := AcademicListResult{Year: "2026", SourceURL: "fixture", Events: []AcademicEvent{{Year: "2026", Month: "3월", Date: "2(월)", Title: "개강"}}}
	fail := false
	sentinel := errors.New("source unavailable")
	service := &Service{cacheStore: store, academic: &fakeAcademicSource{fetch: func(got context.Context, year string) (AcademicListResult, error) {
		calls++
		if got != ctx || year != "2026" {
			t.Fatalf("ctx=%v year=%q", got, year)
		}
		if fail {
			return AcademicListResult{}, sentinel
		}
		return expected, nil
	}}}
	for _, refresh := range []bool{false, false, true} {
		got, err := service.AcademicList(ctx, AcademicListOptions{Year: " 2026 ", Refresh: refresh})
		if err != nil || !reflect.DeepEqual(got, expected) {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	fail = true
	if _, err := service.AcademicList(ctx, AcademicListOptions{Year: "2026", Refresh: true}); !errors.Is(err, sentinel) {
		t.Fatalf("err=%v", err)
	}
	got, err := service.AcademicList(ctx, AcademicListOptions{Year: "2026"})
	if err != nil || !reflect.DeepEqual(got, expected) || calls != 3 {
		t.Fatalf("cache after failure=%+v err=%v calls=%d", got, err, calls)
	}
	for _, year := range []string{"26", "abcd", "20260"} {
		if _, err := service.AcademicList(ctx, AcademicListOptions{Year: year}); err == nil {
			t.Fatalf("accepted year=%q", year)
		}
	}
	if calls != 3 {
		t.Fatalf("invalid input fetched source: %d", calls)
	}
}
