package klas

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEvaluationCourseJSONOmitsRawPayload(t *testing.T) {
	course := EvaluationCourse{
		Name:         "정규화 과목",
		Professor:    "정규화 교수",
		OpenGwamokNo: "subject-1",
		Evaluated:    true,
		ThisYear:     "2026",
		Hakgi:        "1",
		Raw:          evaluationCourseItem{Name: "raw-secret-course"},
	}
	payload, err := json.Marshal(course)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(payload, []byte("raw-secret")) || bytes.Contains(payload, []byte(`"Raw"`)) {
		t.Fatalf("EvaluationCourse JSON contains Raw payload: %s", payload)
	}
	var roundTrip EvaluationCourse
	if err := json.Unmarshal(payload, &roundTrip); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if roundTrip.Name != course.Name || roundTrip.Professor != course.Professor || roundTrip.OpenGwamokNo != course.OpenGwamokNo || !roundTrip.Evaluated || roundTrip.ThisYear != course.ThisYear || roundTrip.Hakgi != course.Hakgi {
		t.Fatalf("EvaluationCourse round trip = %+v", roundTrip)
	}
}

func TestBuildEvaluationPayloadUsesRequestedDefaults(t *testing.T) {
	term := EvaluationTerm{Year: "2026", Hakgi: "1", JudgeChasu: "last", JudgeName: "기말평가"}
	course := EvaluationCourse{
		Name:          "오픈소스소프트웨어실습",
		ThisYear:      "2026",
		Hakgi:         "1",
		OpenMajorCode: "I040",
		OpenGrade:     "2",
		OpenGwamokNo:  "1234",
		BunbanNo:      "01",
	}
	form := EvaluationForm{
		Questions: map[string]any{
			"q01":     "강의 문항",
			"q02":     "강의 문항",
			"sq01":    "특수 문항",
			"tq01":    "소규모 문항",
			"fq01":    "원격 문항",
			"chamgo":  "서술형",
			"chamgo2": "기타2",
			"chamgo3": "원격 서술형",
		},
		BunbanOptions: map[string]any{
			"experimentOpt": "Y",
			"eng100Opt":     "Y",
			"recOpt":        "Y",
		},
	}

	payload := BuildEvaluationPayload(term, course, form, EvaluationAnswerOptions{})

	for _, key := range []string{"a01", "a02", "sa01", "fa01", "ta01"} {
		if payload[key] != "5" {
			t.Fatalf("%s = %v, want 5", key, payload[key])
		}
	}
	if payload["chamgo2Opt"] != "N" {
		t.Fatalf("chamgo2Opt = %v, want N", payload["chamgo2Opt"])
	}
	for _, key := range []string{"chamgo", "chamgo3"} {
		if payload[key] != "많은 도움 되었습니다. 한학기동안 감사했습니다." {
			t.Fatalf("%s = %v", key, payload[key])
		}
	}
	if payload["engOpt"] != "N" || payload["ea1"] != "" || payload["chamgoEng"] != "" {
		t.Fatalf("engineering fields should be disabled by default: engOpt=%v ea1=%v chamgoEng=%v", payload["engOpt"], payload["ea1"], payload["chamgoEng"])
	}
}

func TestBuildEvaluationPayloadSkipsEngineeringWithoutExplicitOptIn(t *testing.T) {
	term := EvaluationTerm{Year: "2026", Hakgi: "1", JudgeChasu: "last", JudgeName: "기말평가"}
	course := EvaluationCourse{Name: "공학과목", Engineering: true}
	form := EvaluationForm{
		Questions:       map[string]any{"q01": "강의 문항"},
		EngineeringView: "1",
		Engineering:     map[string]any{"level1": "공학인증 문항", "studyResult1": "학습성과"},
	}

	payload := BuildEvaluationPayload(term, course, form, EvaluationAnswerOptions{})
	if payload["engOpt"] != "Y" {
		t.Fatalf("engOpt = %v, want Y", payload["engOpt"])
	}
	if payload["ea1"] != "" || payload["ea21"] != "" || payload["ea31"] != "" {
		t.Fatalf("engineering answers should be empty without opt-in: ea1=%v ea21=%v ea31=%v", payload["ea1"], payload["ea21"], payload["ea31"])
	}

	payload = BuildEvaluationPayload(term, course, form, EvaluationAnswerOptions{IncludeEngineering: true})
	if payload["ea1"] != "5" || payload["ea21"] != "5" || payload["ea31"] != "5" || payload["chamgoEng"] == "" {
		t.Fatalf("engineering answers were not filled with opt-in: ea1=%v ea21=%v ea31=%v chamgoEng=%v", payload["ea1"], payload["ea21"], payload["ea31"], payload["chamgoEng"])
	}
}
