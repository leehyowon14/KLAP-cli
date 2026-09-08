package app

import "context"

type RoomAvailabilityForDaysOptions struct {
	Days    []int
	Periods []int
	Refresh bool
}

// RoomAvailabilityForDays owns the multi-day query order and refresh policy.
// A failed day returns no results, matching the interactive query contract.
func (s *Service) RoomAvailabilityForDays(ctx context.Context, opts RoomAvailabilityForDaysOptions) ([]RoomAvailableResult, error) {
	return roomAvailabilityForDays(ctx, opts, s.RoomAvailable)
}

func roomAvailabilityForDays(ctx context.Context, opts RoomAvailabilityForDaysOptions, query func(context.Context, RoomAvailableOptions) (RoomAvailableResult, error)) ([]RoomAvailableResult, error) {
	results := make([]RoomAvailableResult, 0, len(opts.Days))
	for index, weekday := range opts.Days {
		result, err := query(ctx, RoomAvailableOptions{Refresh: opts.Refresh && index == 0, Day: RoomWeekdayLabel(weekday), Periods: opts.Periods})
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}
