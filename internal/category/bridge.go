package category

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

type Options struct {
	Reminders []string `json:"reminders"`
	Calendars []string `json:"calendars"`
}

type MacOSBridge struct {
	scriptPath string
}

func NewMacOSBridge(scriptPath string) MacOSBridge {
	return MacOSBridge{scriptPath: scriptPath}
}

func (b MacOSBridge) List() (Options, error) {
	if runtime.GOOS != "darwin" {
		return Options{}, errors.New("category list는 현재 macOS에서만 지원합니다")
	}
	command := exec.Command("swift", b.scriptPath)
	output, err := command.CombinedOutput()
	if err != nil {
		return Options{}, fmt.Errorf("Swift category bridge 실패: %w\n%s", err, string(output))
	}
	var result Options
	if err := json.Unmarshal(bytes.TrimSpace(output), &result); err != nil {
		return Options{}, fmt.Errorf("Swift category bridge 응답 파싱 실패: %w\n%s", err, string(output))
	}
	return result, nil
}
