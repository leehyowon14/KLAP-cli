package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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
