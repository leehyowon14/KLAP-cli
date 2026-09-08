package bootstrap

import (
	"os"
	"path/filepath"
	"runtime"
)

func defaultReminderBridgePath() string {
	return defaultBridgePath("KLAP_REMINDER_BRIDGE", "ReminderBridge")
}
func defaultCalendarBridgePath() string {
	return defaultBridgePath("KLAP_CALENDAR_BRIDGE", "CalendarBridge")
}
func defaultCategoryBridgePath() string {
	return defaultBridgePath("KLAP_CATEGORY_BRIDGE", "CategoryBridge")
}
func defaultTranscriptBridgePath() string {
	return defaultBridgePath("KLAP_TRANSCRIPT_BRIDGE", "TranscriptBridge")
}

func defaultBridgePath(environment, product string) string {
	if override := os.Getenv(environment); override != "" {
		return override
	}
	executable, _ := os.Executable()
	sourceRoot := ""
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		sourceRoot = filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	}
	return selectBridgePath(bridgeCandidates(product, executable, ".", sourceRoot))
}

// Installed artifacts take precedence over a potentially unrelated working tree.
func bridgeCandidates(product, executable, workingRoot, sourceRoot string) []string {
	var candidates []string
	if executable != "" {
		directory := executableDirectory(executable)
		candidates = append(candidates, filepath.Join(directory, "bridges", "macos", product), filepath.Join(directory, product))
	}
	roots := []string{workingRoot}
	if sourceRoot != "" && sourceRoot != workingRoot {
		roots = append(roots, sourceRoot)
	}
	for _, root := range roots {
		bridgeRoot := filepath.Join(root, "bridges", "macos")
		for _, build := range []string{"artifacts", filepath.Join("out", "Products", "Release"), "release", filepath.Join("out", "Products", "Debug"), "debug"} {
			candidates = append(candidates, filepath.Join(bridgeRoot, ".build", build, product))
		}
	}
	return candidates
}

func selectBridgePath(candidates []string) string {
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
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
