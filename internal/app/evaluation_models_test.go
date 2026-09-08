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

func evaluationModelFixture(t *testing.T) *int {
	t.Helper()
	submits := new(int)
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{}`
		switch req.URL.Path {
		case "/std/cps/inqire/LctreEvlTermCheck.do":
			body = `{"thisYear":"2026","hakgi":"1","judgeChasu":"last","fromDate":"2026-06-01","toDate":"2026-06-30"}`
		case "/std/cps/inqire/LctreEvlsugangList.do":
			body = `[{"thisYear":"2026","hakgi":"1","openMajorCode":"0000","openGrade":"3","openGwamokNo":"1111","bunbanNo":"01","gwamokKname":"done","memberName":"professor","hakjumNum":3,"isuGubun":"major","engIsuGubun":"engineering","engOpt":"N","judgeOpt":"Y"},{"thisYear":"2026","hakgi":"1","openMajorCode":"0000","openGrade":"3","openGwamokNo":"2222","bunbanNo":"02","gwamokKname":" target ","memberName":" professor ","hakjumNum":3,"isuGubun":"major","engIsuGubun":"engineering","engOpt":"Y","judgeOpt":"N"}]`
		case "/std/cps/inqire/LctreEvlGetHakjuk.do":
		case "/std/cps/inqire/lctreEvlCheck.do":
			body = `[{}]`
		case "/std/cps/inqire/LctreEvlGetque.do", "/std/cps/inqire/LctreEvlBunbanCheck.do", "/std/cps/inqire/LctreEvlBunbanSmallCheck.do", "/std/cps/inqire/LctreEvlEngQuestion.do":
		case "/std/cps/inqire/insertEvl.do":
			var payload map[string]any
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["openGwamokNo"] != "2222" || payload["bunbanNo"] != "02" || payload["gwamokKname"] != "target" {
				t.Fatalf("wrong selected course payload: %+v", payload)
			}
			*submits++
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	return submits
}

func TestEvaluationModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	evaluationModelFixture(t)
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	term, err := client.EvaluationTerm(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	model := evaluationTermModel(term)
	if !model.TermEnabled || model.Value != "2026,1" || model.JudgeName != "기말평가" {
		t.Fatalf("term=%+v", model)
	}
	assertModelJSONParity(t, term, model)
	courses, err := client.EvaluationCourses(context.Background(), term)
	if err != nil || len(courses) != 2 {
		t.Fatalf("courses=%+v error=%v", courses, err)
	}
	rows := makeEvaluationRows(courses)
	if rows[1].Index != 2 || rows[1].Course.Name != "target" || !rows[1].Course.Engineering || rows[1].Course.Evaluated {
		t.Fatalf("rows=%+v", rows)
	}
	for index, course := range courses {
		assertModelJSONParity(t, course, rows[index].Course)
	}
	assertModelJSONParity(t, klas.EvaluationTerm{}, evaluationTermModel(klas.EvaluationTerm{}))
	assertModelJSONParity(t, klas.EvaluationCourse{}, evaluationCourseModel(klas.EvaluationCourse{}))
}

func TestEvaluationSubmitKeepsFreshSelectedAdapterCourse(t *testing.T) {
	submits := evaluationModelFixture(t)
	deps := testDependencies(t)
	deps.NewKlasClient = klas.NewClient
	deps.Sessions = &fakeSessionStore{loadSession: func(context.Context, string) (klas.Session, error) { return klas.Session{}, nil }}
	s, err := NewService(deps)
	if err != nil {
		t.Fatal(err)
	}
	for _, confirm := range []bool{false, true} {
		result, err := s.EvaluationSubmit(context.Background(), EvaluationSubmitOptions{User: UserOption{StudentID: "student"}, Selector: "2", Confirm: confirm})
		if err != nil || len(result.Items) != 1 || result.Items[0].Err != nil || result.Items[0].Row.Course.OpenGwamokNo != "2222" || result.Submitted != confirm {
			t.Fatalf("result=%+v error=%v", result, err)
		}
		want := 0
		if confirm {
			want = 1
		}
		if *submits != want {
			t.Fatalf("submits=%d want=%d", *submits, want)
		}
	}
}
