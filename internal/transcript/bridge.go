package transcript

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

type Job struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	Locale     string `json:"locale,omitempty"`
}

type Request struct {
	Jobs []Job `json:"jobs"`
}

type Result struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	Text       string `json:"text"`
	Err        string `json:"error"`
}

type Response struct {
	Results []Result `json:"results"`
}

type MacOSBridge struct {
	scriptPath string
}

func NewMacOSBridge(scriptPath string) MacOSBridge {
	return MacOSBridge{scriptPath: scriptPath}
}

func (b MacOSBridge) Transcribe(request Request) (Response, error) {
	if runtime.GOOS != "darwin" {
		return Response{}, errors.New("강의 전사는 현재 macOS에서만 지원합니다")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("transcript payload 직렬화 실패: %w", err)
	}

	command := exec.Command("swift", b.scriptPath)
	command.Stdin = bytes.NewReader(payload)
	output, err := command.CombinedOutput()
	if err != nil {
		return Response{}, fmt.Errorf("Swift transcript bridge 실패: %w\n%s", err, string(output))
	}

	var response Response
	if err := json.Unmarshal(bytes.TrimSpace(output), &response); err != nil {
		return Response{}, fmt.Errorf("Swift transcript bridge 응답 파싱 실패: %w\n%s", err, string(output))
	}
	return response, nil
}
