package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestAssignmentModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{}`
		switch req.URL.Path {
		case "/std/lis/evltn/TaskStdList.do":
			body = `[{"ordseq":42,"weeklyseq":2,"weeklysubseq":3,"title":" title ","startdate":"2026-09-01 09:00:00","expiredate":"2026-09-10 18:00:00","submityn":"Y"}]`
		case "/std/lis/evltn/TaskStdView.do":
			body = `{"rpt":{"ordseq":42,"title":" title ","contents":"<p>body</p>","startdate":"2026-09-01 09:00:00","expiredate":"2026-09-10 18:00:00","submityn":"Y","reptype":"1","submitfiletype":"pdf","filelimit":20},"smt":{"title":"submitted","contents":"<p>answer</p>","finalscore":95,"tutorcontents":"<p>feedback</p>"}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := client.Assignments(context.Background(), "2026,1", Course{Value: "subject"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v error=%v", rows, err)
	}
	row := assignmentModel(rows[0])
	if row.OrdSeq != "42" || row.WeeklySeq != "2" || row.WeeklySubSeq != "3" || row.Title != "title" || row.StartAt == nil || row.DueAt == nil || !row.Submitted {
		t.Fatalf("row=%+v", row)
	}
	assertModelJSONParity(t, rows[0], row)
	detail, err := client.AssignmentDetail(context.Background(), "2026,1", Course{Value: "subject"}, "42")
	if err != nil {
		t.Fatal(err)
	}
	model := assignmentDetailModel(detail)
	if model.ContentText != "body" || model.ReportType != "개인" || model.SubmittedText != "answer" || model.TutorText != "feedback" || model.FinalScore != "95" {
		t.Fatalf("detail=%+v", model)
	}
	assertModelJSONParity(t, detail, model)
	assertModelJSONParity(t, klas.Assignment{}, assignmentModel(klas.Assignment{}))
	assertModelJSONParity(t, klas.AssignmentDetail{}, assignmentDetailModel(klas.AssignmentDetail{}))
}

func assertModelJSONParity(t *testing.T, adapter, model any) {
	t.Helper()
	if _, ok := reflect.TypeOf(model).FieldByName("Raw"); ok {
		t.Fatal("app model exposes wire data")
	}
	before, err := json.Marshal(adapter)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("JSON contract changed: before=%s after=%s", before, after)
	}
	roundTrip := reflect.New(reflect.TypeOf(model))
	if err := json.Unmarshal(after, roundTrip.Interface()); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(roundTrip.Interface())
	if err != nil || string(encoded) != string(after) {
		t.Fatalf("round trip=%s error=%v", encoded, err)
	}
}
