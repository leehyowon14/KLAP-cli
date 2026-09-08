package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/category"
	"testing"
)

type fakeCategoryLister struct {
	list func(context.Context) (category.Options, error)
}

func (f fakeCategoryLister) List(ctx context.Context) (category.Options, error) { return f.list(ctx) }

func TestCategoryOptionsFallbackAndCancellation(t *testing.T) {
	for _, failure := range []error{errors.New("permission unavailable"), context.Canceled, context.DeadlineExceeded} {
		ctx := context.Background()
		service := &Service{categoryLister: fakeCategoryLister{list: func(got context.Context) (category.Options, error) {
			if got != ctx {
				t.Fatal("context not forwarded")
			}
			return category.Options{}, failure
		}}}
		got, err := service.CategoryOptionsContext(ctx)
		if errors.Is(failure, context.Canceled) || errors.Is(failure, context.DeadlineExceeded) {
			if !errors.Is(err, failure) {
				t.Fatalf("cancellation hidden: %v", err)
			}
		} else if err != nil || len(got.Reminders) == 0 || len(got.Calendars) == 0 {
			t.Fatalf("fallback=%+v err=%v", got, err)
		}
	}
}
