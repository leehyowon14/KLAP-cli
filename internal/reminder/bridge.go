package reminder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

type Assignment struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Course          string     `json:"course"`
	DueAt           *time.Time `json:"dueAt"`
	Submitted       bool       `json:"submitted"`
	DetailURL       string     `json:"detailUrl"`
	Notes           string     `json:"notes"`
	KnownSourceHash string     `json:"knownSourceHash,omitempty"`
	ForceUpdate     bool       `json:"forceUpdate,omitempty"`
}

type SyncRequest struct {
	ListName        string       `json:"listName"`
	UseExistingList bool         `json:"useExistingList"`
	AlarmBeforeMin  int          `json:"alarmBeforeMin"`
	Assignments     []Assignment `json:"assignments"`
}

type SyncResult struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Completed int      `json:"completed"`
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
		return SyncResult{}, errors.New("assignment remind는 현재 macOS에서만 지원합니다")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return SyncResult{}, fmt.Errorf("reminder payload 직렬화 실패: %w", err)
	}

	command := exec.Command("swift", b.scriptPath)
	command.Stdin = bytes.NewReader(payload)
	output, err := command.CombinedOutput()
	if err != nil {
		return SyncResult{}, fmt.Errorf("Swift reminder bridge 실패: %w\n%s", err, string(output))
	}

	var result SyncResult
	if err := json.Unmarshal(bytes.TrimSpace(output), &result); err != nil {
		return SyncResult{}, fmt.Errorf("Swift reminder bridge 응답 파싱 실패: %w\n%s", err, string(output))
	}
	return result, nil
}
