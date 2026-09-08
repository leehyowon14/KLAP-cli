package macos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/transcript"
	"io"
	"runtime"
)

type TranscriptBridge struct{ path string }

func NewTranscriptBridge(path string) TranscriptBridge { return TranscriptBridge{path: path} }

func (b TranscriptBridge) Transcribe(ctx context.Context, request transcript.Request) (transcript.Response, error) {
	if runtime.GOOS != "darwin" {
		return transcript.Response{}, errors.New("강의 전사는 현재 macOS에서만 지원합니다")
	}
	var result transcript.Response
	if err := runJSONBridge(ctx, b.path, "transcript", request, &result); err != nil {
		return transcript.Response{}, err
	}
	return result, nil
}

type transcriptEvent struct {
	Type       string              `json:"type"`
	InputPath  string              `json:"inputPath"`
	OutputPath string              `json:"outputPath"`
	Progress   float64             `json:"progress"`
	Err        string              `json:"error"`
	Results    []transcript.Result `json:"results"`
}

func (b TranscriptBridge) TranscribeWithProgress(ctx context.Context, request transcript.Request, onProgress func(transcript.Progress)) (transcript.Response, error) {
	if runtime.GOOS != "darwin" {
		return transcript.Response{}, errors.New("강의 전사는 현재 macOS에서만 지원합니다")
	}
	request.Progress = true
	payload, err := json.Marshal(request)
	if err != nil {
		return transcript.Response{}, fmt.Errorf("transcript payload 직렬화 실패: %w", err)
	}
	var response transcript.Response
	responseSeen := false
	err = (ProcessRunner{}).Run(ctx, bridgeSpec(b.path, payload), func(stdout io.Reader) error {
		decoder := json.NewDecoder(stdout)
		for {
			var item transcriptEvent
			if err := decoder.Decode(&item); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("Swift transcript bridge 이벤트 파싱 실패: %w", err)
			}
			switch item.Type {
			case "progress":
				if onProgress != nil {
					onProgress(transcript.Progress{InputPath: item.InputPath, OutputPath: item.OutputPath, Progress: item.Progress, Err: item.Err})
				}
			case "response":
				response.Results = item.Results
				responseSeen = true
			}
		}
		if !responseSeen {
			return errors.New("Swift transcript bridge 응답이 없습니다")
		}
		return nil
	})
	if err != nil {
		return transcript.Response{}, fmt.Errorf("Swift transcript bridge 실패: %w", err)
	}
	return response, nil
}
