package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestGradeModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/std/cps/inqire/AtnlcScreSungjukTot.do":
			body = `{"applyHakjum":30,"majorApplyHakjum":20,"cultureApplyHakjum":8,"etcApplyHakjum":2,"chidukHakjum":27,"majorChidukHakjum":18,"cultureChidukHakjum":7,"etcChidukHakjum":2,"delHakjum":3,"hwakinScoresum":4.25,"jaechulScoresum":4.1}`
		case "/std/cps/inqire/AtnlcScreSungjukInfo.do":
			body = `[{"thisYear":"2026","hakgi":"1","sungjukList":[{"gwamokKname":" course ","codeName1":"major","hakjumNum":3,"getGrade":"A+","hakgwa":"department","hakjungNo":"subject","finishOpt":"Y","retakeOpt":"Y","retakeGetGrade":"B+","termCheck":"Y","termFinish":"Y"}]}]`
		case "/std/cps/inqire/StandStdList.do":
			body = `[{"thisYear":"2026","hakgi":"1","applyHakjum":18,"applySum":76.5,"applyPoint":4.25,"classOrder":2,"manNum":30,"warningOpt":"N","pcnt":95}]`
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
	report, err := client.Grades(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	model := gradeReportModel(report)
	if model.Summary.GPA != "4.25" || len(model.Terms) != 1 || len(model.Terms[0].Courses) != 1 {
		t.Fatalf("report=%+v", model)
	}
	course := model.Terms[0].Courses[0]
	if course.Name != "course" || !course.GradePublic || !course.Retake || course.RetakeGrade != "B+" || !course.TermFinished {
		t.Fatalf("course=%+v", course)
	}
	assertModelJSONParity(t, report, model)
	assertModelJSONParity(t, report.Summary, model.Summary)
	assertModelJSONParity(t, report.Terms[0], model.Terms[0])
	assertModelJSONParity(t, report.Terms[0].Courses[0], course)
	ranks, err := client.Ranks(context.Background())
	if err != nil || len(ranks) != 1 {
		t.Fatalf("ranks=%+v error=%v", ranks, err)
	}
	rows := rankModels(ranks)
	if rows[0].ClassRank != "2" || rows[0].ClassSize != "30" || rows[0].TermValue != "2026,1" {
		t.Fatalf("rows=%+v", rows)
	}
	assertModelJSONParity(t, ranks[0], rows[0])
	for _, value := range []klas.GradeReport{{}, {Terms: []klas.GradeTerm{}}, {Terms: []klas.GradeTerm{{Courses: []klas.GradeCourse{}}}}} {
		assertModelJSONParity(t, value, gradeReportModel(value))
	}
	assertModelJSONParity(t, klas.Rank{}, rankModel(klas.Rank{}))
	if rankModels(nil) != nil || rankModels([]klas.Rank{}) == nil {
		t.Fatal("rank slice shape changed")
	}
}
