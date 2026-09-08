package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestSyllabusModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/std/cps/atnlc/LectrePlanData.do":
			body = `[{"openMajorCode":"0000","openGrade":"3","openGwamokNo":"1234","bunbanNo":"01","gwamokkename":"full","gwamokKname":"course","gwamokEname":"English","codeName1":"major","hakjumNum":3,"currentNum":20,"memberName":"professor","jikgeubName":"title","summary":"summary","purpose":"purpose","result1":"outcome","gwamokAble":"competency","bookName":"book","face100Opt":"Y","attendBiyul":10,"learnBiyul":5,"middleBiyul":20,"lastBiyul":30,"reportBiyul":15,"quizBiyul":10,"gitaBiyul":10,"week1Lecture":"topic","week1Subs":"note","email":"private@example.test","telNo":"private-phone"}]`
		case "/std/cps/atnlc/LectreTimeInfo.do":
			body = `[{"dayname1":"월","timeNo1":1,"timeNo2":"2","locHname":"room","code":"private-code"}]`
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	syllabus, err := client.SyllabusBySubjectID(context.Background(), "subject")
	if err != nil {
		t.Fatal(err)
	}
	model := syllabusModel(syllabus)
	if model.KoreanName != "course" || model.CurrentNum != "20" || model.Evaluation.Final != 30 || len(model.Schedule) == 0 || model.Schedule[0].Topic != "topic" || len(model.Times) != 1 || len(model.Times[0].Periods) != 2 {
		t.Fatalf("model=%+v", model)
	}
	assertModelJSONParity(t, syllabus, model)
	assertModelJSONParity(t, syllabus.Evaluation, model.Evaluation)
	assertModelJSONParity(t, syllabus.Schedule[0], model.Schedule[0])
	assertModelJSONParity(t, syllabus.Times[0], model.Times[0])
	encoded, err := json.Marshal(model)
	if err != nil || strings.Contains(string(encoded), "private") {
		t.Fatalf("private wire data leaked: %s error=%v", encoded, err)
	}
	model.Times[0].Periods[0] = 99
	if syllabus.Times[0].Periods[0] == 99 {
		t.Fatal("model aliases adapter period slice")
	}
	for _, value := range []klas.Syllabus{{}, {Schedule: []klas.SyllabusWeek{}, Times: []klas.SyllabusTime{{Periods: []int{}}}}} {
		assertModelJSONParity(t, value, syllabusModel(value))
	}
}
