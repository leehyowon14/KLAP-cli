package transcript

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

type Job struct {
	InputPath         string   `json:"inputPath"`
	OutputPath        string   `json:"outputPath"`
	Locale            string   `json:"locale,omitempty"`
	ContextualStrings []string `json:"contextualStrings,omitempty"`
}

type Request struct {
	Jobs     []Job `json:"jobs"`
	Progress bool  `json:"progress,omitempty"`
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

type Progress struct {
	InputPath  string
	OutputPath string
	Progress   float64
	Err        string
}

type event struct {
	Type       string   `json:"type"`
	InputPath  string   `json:"inputPath"`
	OutputPath string   `json:"outputPath"`
	Progress   float64  `json:"progress"`
	Err        string   `json:"error"`
	Results    []Result `json:"results"`
}

type MacOSBridge struct {
	bridgePath string
}

func NewMacOSBridge(bridgePath string) MacOSBridge {
	return MacOSBridge{bridgePath: bridgePath}
}

func (b MacOSBridge) Transcribe(request Request) (Response, error) {
	if runtime.GOOS != "darwin" {
		return Response{}, errors.New("강의 전사는 현재 macOS에서만 지원합니다")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("transcript payload 직렬화 실패: %w", err)
	}

	commandName, commandArgs := b.commandSpec()
	command := exec.Command(commandName, commandArgs...)
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

func (b MacOSBridge) TranscribeWithProgress(request Request, onProgress func(Progress)) (Response, error) {
	if runtime.GOOS != "darwin" {
		return Response{}, errors.New("강의 전사는 현재 macOS에서만 지원합니다")
	}

	request.Progress = true
	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, fmt.Errorf("transcript payload 직렬화 실패: %w", err)
	}

	commandName, commandArgs := b.commandSpec()
	command := exec.Command(commandName, commandArgs...)
	command.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		return Response{}, fmt.Errorf("Swift transcript bridge stdout 연결 실패: %w", err)
	}
	if err := command.Start(); err != nil {
		return Response{}, fmt.Errorf("Swift transcript bridge 시작 실패: %w", err)
	}

	var response Response
	responseSeen := false
	decoder := json.NewDecoder(stdout)
	for {
		var item event
		if err := decoder.Decode(&item); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = command.Wait()
			return Response{}, fmt.Errorf("Swift transcript bridge 이벤트 파싱 실패: %w\n%s", err, stderr.String())
		}
		switch item.Type {
		case "progress":
			if onProgress != nil {
				onProgress(Progress{
					InputPath:  item.InputPath,
					OutputPath: item.OutputPath,
					Progress:   item.Progress,
					Err:        item.Err,
				})
			}
		case "response":
			response.Results = item.Results
			responseSeen = true
		}
	}
	if err := command.Wait(); err != nil {
		return Response{}, fmt.Errorf("Swift transcript bridge 실패: %w\n%s", err, stderr.String())
	}
	if !responseSeen {
		return Response{}, fmt.Errorf("Swift transcript bridge 응답이 없습니다\n%s", stderr.String())
	}
	return response, nil
}

func (b MacOSBridge) commandSpec() (string, []string) {
	bridgePath := strings.TrimSpace(b.bridgePath)
	if strings.HasSuffix(bridgePath, ".swift") {
		return "swift", []string{bridgePath}
	}
	return bridgePath, nil
}
