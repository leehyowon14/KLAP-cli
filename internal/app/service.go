package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
)

type Service struct {
	store                AccountStore
	sessions             sessionStore
	settingsStore        SettingsStore
	cacheStore           CacheStore
	syncStateStore       syncStateStore
	newKlasClient        func() (*klas.Client, error)
	login                func(context.Context, *klas.Client, string, string) (klas.Session, error)
	reminderSyncer       ReminderSyncer
	calendarSyncer       CalendarSyncer
	categoryLister       CategoryLister
	transcriptBridgePath string
}

type OpenURLResult struct {
	URL string
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func compactNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
