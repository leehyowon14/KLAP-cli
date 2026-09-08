package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestLectureModelsFromAdapterFixture(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{}`
		if strings.HasSuffix(request.URL.Path, "SelectOnlineCntntsStdList.do") {
			body = `[{"subj":"subject","lrnSn":42,"fileId":9,"weekNo":1,"weeklyseq":2,"moduletitle":" 1주차 ","sbjt":" 강의 ","prog":25,"achivTime":2,"rcognTime":8,"sdateY":"2026-09-01","sdateH":"09","sdateM":"00","edateY":"2026-09-10","edateH":"23","edateM":"59","grcode":"private-group"}]`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	values, err := client.Lectures(context.Background(), "2026,1", Course{Name: "과목", Value: "subject"})
	if err != nil {
		t.Fatal(err)
	}
	models := lectureModels(values)
	if len(models) != 1 || models[0].LearningSeq != "42" || models[0].Title != "강의" || models[0].AchievedTime != "2" || models[0].RequiredTime != "8" {
		t.Fatalf("models=%#v", models)
	}
	assertModelJSONParity(t, values[0], models[0])
	assertModelJSONParity(t, klas.Lecture{}, lectureModel(klas.Lecture{}))
	progress := klas.LectureProgress{TotalTime: "8", PTime: "8", Progress: 100, Completed: true}
	assertModelJSONParity(t, progress, lectureProgressModel(progress))
	if lectureModels(nil) != nil || lectureModels([]klas.Lecture{}) == nil {
		t.Fatal("nil/empty distinction lost")
	}
}
