package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

type GradeOptions struct {
	User      UserOption
	TermValue string
	Refresh   bool
}

type RankOptions struct {
	User      UserOption
	TermValue string
	Refresh   bool
}

type GradeResult struct {
	Report    klas.GradeReport
	TermValue string
}

type RankResult struct {
	Rows []klas.Rank
}

func (s *Service) Grade(ctx context.Context, opts GradeOptions) (GradeResult, error) {
	termValue, err := normalizeTermValue(opts.TermValue)
	if err != nil {
		return GradeResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return GradeResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return GradeResult{}, err
	}

	cacheKey := listCacheKeyVersion("grade", "v2", studentID, termValue, "")
	if !opts.Refresh {
		var cached GradeResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	report, err := client.Grades(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return GradeResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			report, err = client.Grades(ctx)
		}
	}
	if err != nil {
		return GradeResult{}, err
	}
	if termValue != "" {
		filtered := report.Terms[:0]
		for _, term := range report.Terms {
			if term.Year+","+term.Hakgi == termValue {
				filtered = append(filtered, term)
			}
		}
		report.Terms = filtered
	}
	result := GradeResult{Report: report, TermValue: termValue}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) Rank(ctx context.Context, opts RankOptions) (RankResult, error) {
	termValue, err := normalizeTermValue(opts.TermValue)
	if err != nil {
		return RankResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return RankResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return RankResult{}, err
	}

	cacheKey := listCacheKeyVersion("rank", "v2", studentID, termValue, "")
	if !opts.Refresh {
		var cached RankResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	rows, err := client.Ranks(ctx)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return RankResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			rows, err = client.Ranks(ctx)
		}
	}
	if err != nil {
		return RankResult{}, err
	}
	if termValue != "" {
		filtered := rows[:0]
		for _, row := range rows {
			if row.TermValue == termValue {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	result := RankResult{Rows: rows}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}
