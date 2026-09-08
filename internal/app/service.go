package app

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	store                *account.Store
	sessions             sessionStore
	settingsStore        *settings.Store
	cacheStore           *cache.Store
	syncStateStore       syncStateStore
	newKlasClient        func() (*klas.Client, error)
	login                func(context.Context, *klas.Client, string, string) (klas.Session, error)
	reminderBridgePath   string
	calendarBridgePath   string
	categoryBridgePath   string
	transcriptBridgePath string
}

type OpenURLResult struct {
	URL string
}

type serviceStoreFactories struct {
	settings  func() (*settings.Store, error)
	cache     func() (*cache.Store, error)
	syncState func() (*syncstate.Store, error)
}

func NewService(store *account.Store) (*Service, error) {
	return newService(store, serviceStoreFactories{
		settings:  settings.NewStore,
		cache:     cache.NewStore,
		syncState: syncstate.NewStore,
	})
}

func newService(store *account.Store, factories serviceStoreFactories) (*Service, error) {
	settingsStore, err := factories.settings()
	if err != nil {
		return nil, fmt.Errorf("settings store 초기화 실패: %w", err)
	}
	cacheStore, err := factories.cache()
	if err != nil {
		return nil, fmt.Errorf("cache store 초기화 실패: %w", err)
	}
	syncStateStore, err := factories.syncState()
	if err != nil {
		return nil, fmt.Errorf("sync state store 초기화 실패: %w", err)
	}
	return &Service{
		store:          store,
		sessions:       store,
		settingsStore:  settingsStore,
		cacheStore:     cacheStore,
		syncStateStore: syncStateStore,
		newKlasClient:  klas.NewClient,
		login: func(ctx context.Context, client *klas.Client, studentID string, password string) (klas.Session, error) {
			return client.Login(ctx, studentID, password)
		},
		reminderBridgePath:   defaultReminderBridgePath(),
		calendarBridgePath:   defaultCalendarBridgePath(),
		categoryBridgePath:   defaultCategoryBridgePath(),
		transcriptBridgePath: defaultTranscriptBridgePath(),
	}, nil
}

func (s *Service) startCaffeinate(ctx context.Context) func() {
	if runtime.GOOS != "darwin" {
		return func() {}
	}
	current, err := s.loadSettings()
	if err != nil || !settings.DownloadCaffeinateEnabled(current.Download) {
		return func() {}
	}

	cmd := exec.CommandContext(ctx, "caffeinate", "-dims")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return func() {}
	}
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
}

var academicDatePattern = regexp.MustCompile(`(?:(\d{1,2})\s*[./]\s*)?(\d{1,2})\s*(?:일|\([^)]*\))?`)

func academicEventDueAt(event AcademicEvent) (time.Time, bool) {
	startAt, _, ok := academicEventRange(event)
	return startAt, ok
}

func AcademicEventRange(event AcademicEvent) (time.Time, time.Time, bool) {
	return academicEventRange(event)
}

func academicEventRange(event AcademicEvent) (time.Time, time.Time, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(event.Year))
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	monthText := strings.TrimSuffix(strings.TrimSpace(event.Month), "월")
	month, err := strconv.Atoi(monthText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	dateText := strings.TrimSpace(event.Date)
	matches := academicDatePattern.FindAllStringSubmatch(dateText, -1)
	if len(matches) == 0 {
		return time.Time{}, time.Time{}, false
	}
	dates := make([]time.Time, 0, len(matches))
	for _, match := range matches {
		eventMonth := month
		if strings.TrimSpace(match[1]) != "" {
			parsedMonth, monthErr := strconv.Atoi(match[1])
			if monthErr != nil {
				return time.Time{}, time.Time{}, false
			}
			eventMonth = parsedMonth
		}
		day, dayErr := strconv.Atoi(match[2])
		if dayErr != nil {
			return time.Time{}, time.Time{}, false
		}
		dates = append(dates, time.Date(year, time.Month(eventMonth), day, 0, 0, 0, 0, time.Local))
	}
	startAt := dates[0]
	endAt := dates[len(dates)-1]
	if endAt.Before(startAt) {
		endAt = endAt.AddDate(1, 0, 0)
	}
	return startAt, endAt.AddDate(0, 0, 1), true
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

func defaultReminderBridgePath() string {
	if override := os.Getenv("KLAP_REMINDER_BRIDGE"); override != "" {
		return override
	}

	candidates := []string{
		filepath.Join("bridges", "macos", "reminder.swift"),
	}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, filepath.Join(repoRoot, "bridges", "macos", "reminder.swift"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(executableDirectory(executable), "bridges", "macos", "reminder.swift"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func defaultCalendarBridgePath() string {
	if override := os.Getenv("KLAP_CALENDAR_BRIDGE"); override != "" {
		return override
	}

	candidates := []string{
		filepath.Join("bridges", "macos", "calendar.swift"),
	}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, filepath.Join(repoRoot, "bridges", "macos", "calendar.swift"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(executableDirectory(executable), "bridges", "macos", "calendar.swift"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func defaultCategoryBridgePath() string {
	if override := os.Getenv("KLAP_CATEGORY_BRIDGE"); override != "" {
		return override
	}

	candidates := []string{
		filepath.Join("bridges", "macos", "categories.swift"),
	}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, filepath.Join(repoRoot, "bridges", "macos", "categories.swift"))
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(executableDirectory(executable), "bridges", "macos", "categories.swift"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func defaultTranscriptBridgePath() string {
	if override := os.Getenv("KLAP_TRANSCRIPT_BRIDGE"); override != "" {
		return override
	}

	candidates := transcriptBridgeCandidates(filepath.Join("bridges", "macos"))
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
		candidates = append(candidates, transcriptBridgeCandidates(filepath.Join(repoRoot, "bridges", "macos"))...)
	}
	if executable, err := os.Executable(); err == nil {
		executableDir := executableDirectory(executable)
		candidates = append(candidates, filepath.Join(executableDir, "TranscriptBridge"))
		candidates = append(candidates, transcriptBridgeCandidates(filepath.Join(executableDir, "bridges", "macos"))...)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

func executableDirectory(executable string) string {
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return filepath.Dir(executable)
}

func transcriptBridgeCandidates(root string) []string {
	return []string{
		filepath.Join(root, ".build", "release", "TranscriptBridge"),
		filepath.Join(root, ".build", "debug", "TranscriptBridge"),
		filepath.Join(root, "transcribe.swift"),
	}
}
