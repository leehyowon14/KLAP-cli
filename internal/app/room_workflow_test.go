package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRoomAvailabilityForDaysOwnsOrderAndRefresh(t *testing.T) {
	for _, refresh := range []bool{false, true} {
		ctx := context.Background()
		var queries []RoomAvailableOptions
		results, err := roomAvailabilityForDays(ctx, RoomAvailabilityForDaysOptions{Days: []int{3, 1, 5}, Periods: []int{2, 3}, Refresh: refresh}, func(gotCtx context.Context, opts RoomAvailableOptions) (RoomAvailableResult, error) {
			if gotCtx != ctx {
				t.Fatal("context changed")
			}
			queries = append(queries, opts)
			return RoomAvailableResult{Weekday: len(queries)}, nil
		})
		if err != nil || len(results) != 3 {
			t.Fatalf("results=%#v err=%v", results, err)
		}
		for i, day := range []string{"수", "월", "금"} {
			if queries[i].Day != day || queries[i].Refresh != (refresh && i == 0) || !reflect.DeepEqual(queries[i].Periods, []int{2, 3}) || results[i].Weekday != i+1 {
				t.Fatalf("query=%#v result=%#v", queries[i], results[i])
			}
		}
	}
}

func TestRoomAvailabilityForDaysStopsOnFailure(t *testing.T) {
	wantErr := errors.New("day failed")
	for _, failAt := range []int{1, 2, 3} {
		calls := 0
		results, err := roomAvailabilityForDays(context.Background(), RoomAvailabilityForDaysOptions{Days: []int{1, 2, 3}}, func(context.Context, RoomAvailableOptions) (RoomAvailableResult, error) {
			calls++
			if calls == failAt {
				return RoomAvailableResult{}, wantErr
			}
			return RoomAvailableResult{Weekday: calls}, nil
		})
		if results != nil || !errors.Is(err, wantErr) || calls != failAt {
			t.Fatalf("results=%#v err=%v calls=%d", results, err, calls)
		}
	}
	results, err := roomAvailabilityForDays(context.Background(), RoomAvailabilityForDaysOptions{}, func(context.Context, RoomAvailableOptions) (RoomAvailableResult, error) {
		t.Fatal("empty query called")
		return RoomAvailableResult{}, nil
	})
	if err != nil || results == nil || len(results) != 0 {
		t.Fatalf("empty=%#v err=%v", results, err)
	}
}
