package app

import (
	"context"
	"errors"
	"strings"
	"time"
)

type AcademicSource interface {
	FetchAcademic(context.Context, string) (AcademicListResult, error)
}

type AcademicListOptions struct {
	Year    string
	Refresh bool
}

type AcademicEvent struct {
	Year  string
	Month string
	Date  string
	Title string
	Note  string
}

type AcademicListResult struct {
	Year      string
	SourceURL string
	Events    []AcademicEvent
}

func (s *Service) AcademicList(ctx context.Context, opts AcademicListOptions) (AcademicListResult, error) {
	year := strings.TrimSpace(opts.Year)
	if err := validateAcademicYear(year); err != nil {
		return AcademicListResult{}, err
	}
	if year == "" {
		year = time.Now().In(time.FixedZone("KST", 9*60*60)).Format("2006")
	}

	cacheKey := listCacheKey("academic", "", year, "")
	if !opts.Refresh {
		var cached AcademicListResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	result, err := s.academic.FetchAcademic(ctx, year)
	if err != nil {
		return AcademicListResult{}, err
	}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func validateAcademicYear(year string) error {
	if year == "" {
		return nil
	}
	if len(year) != 4 {
		return errors.New("연도에는 YYYY 형식의 연도가 필요합니다")
	}
	for _, r := range year {
		if r < '0' || r > '9' {
			return errors.New("연도에는 YYYY 형식의 연도가 필요합니다")
		}
	}
	return nil
}
