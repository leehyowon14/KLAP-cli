package app

import (
	"encoding/json"
	"testing"
)

func TestLectureTransferStageSerialization(t *testing.T) {
	for _, tt := range []struct {
		stage LectureTransferStage
		want  string
	}{
		{LectureStageResolve, `"resolve"`},
		{LectureStageDownload, `"download"`},
		{LectureStageDone, `"done"`},
		{LectureStageSkip, `"skip"`},
		{LectureStageError, `"error"`},
		{LectureStageTranscribe, `"transcribe"`},
		{LectureStageTranscribed, `"transcribed"`},
		{LectureStageTranscriptError, `"transcript-error"`},
	} {
		payload, err := json.Marshal(tt.stage)
		if err != nil || string(payload) != tt.want {
			t.Fatalf("stage=%s payload=%s err=%v", tt.stage, payload, err)
		}
	}
}
