package calendar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

type Event struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	StartAt         time.Time  `json:"startAt"`
	EndAt           time.Time  `json:"endAt"`
	AllDay          bool       `json:"allDay"`
	Notes           string     `json:"notes"`
	URL             string     `json:"url"`
	Recurrence      string     `json:"recurrence,omitempty"`
	RecurrenceEnd   *time.Time `json:"recurrenceEnd,omitempty"`
	KnownSourceHash string     `json:"knownSourceHash,omitempty"`
	ForceUpdate     bool       `json:"forceUpdate,omitempty"`
}

type SyncRequest struct {
	CalendarName    string  `json:"calendarName"`
	UseExistingList bool    `json:"useExistingList"`
	Events          []Event `json:"events"`
}

type SyncResult struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Skipped   int      `json:"skipped"`
	SyncedIDs []string `json:"syncedIds"`
}

type MacOSBridge struct {
	scriptPath string
}

func NewMacOSBridge(scriptPath string) MacOSBridge {
	return MacOSBridge{scriptPath: scriptPath}
}

func (b MacOSBridge) Sync(request SyncRequest) (SyncResult, error) {
	if runtime.GOOS != "darwin" {
		return SyncResult{}, errors.New("academic calendar sync는 현재 macOS에서만 지원합니다")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return SyncResult{}, fmt.Errorf("calendar payload 직렬화 실패: %w", err)
	}

	command := exec.Command("swift", b.scriptPath)
	command.Stdin = bytes.NewReader(payload)
	output, err := command.CombinedOutput()
	if err != nil {
		return SyncResult{}, fmt.Errorf("Swift calendar bridge 실패: %w\n%s", err, string(output))
	}

	var result SyncResult
	if err := json.Unmarshal(bytes.TrimSpace(output), &result); err != nil {
		return SyncResult{}, fmt.Errorf("Swift calendar bridge 응답 파싱 실패: %w\n%s", err, string(output))
	}
	return result, nil
}
