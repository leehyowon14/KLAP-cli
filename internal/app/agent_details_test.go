package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBatchInvalidAndUnsupportedPreserveIDs(t *testing.T) {
	id, _ := StableLectureID(CourseRef{TermValue: "2025,1", CourseID: "c"}, "x")
	s := &Service{}
	v, err := s.GetItems(context.Background(), UserOption{}, []string{"invalid", id, "invalid"}, []string{"content"})
	if err != nil || len(v.Items) != 2 || v.Items[0].Status != "error" || v.Items[1].Status != "unsupported" || v.Items[1].ID != id {
		t.Fatalf("%+v %v", v, err)
	}
	for _, sections := range [][]string{nil, {"bad"}} {
		if _, e := s.GetItems(context.Background(), UserOption{}, []string{id}, sections); e == nil {
			t.Fatal("accepted sections")
		}
	}
}

func TestBatchCompletionFreshAndGrouped(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	reads := 0
	progress := "0"
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{}`
		switch req.URL.Path {
		case "/std/cmn/frame/YearhakgiAtnlcSbjectList.do":
			body = `[{"value":"2025,1","subjList":[{"value":"c","name":"course"}]}]`
		case "/std/lis/evltn/SelectOnlineCntntsStdList.do":
			reads++
			body = `[{"subj":"c","lrnSn":"1","sbjt":"old","ptime":"10","achivTime":"` + progress + `"},{"subj":"c","lrnSn":"2","sbjt":"old2","ptime":"10","achivTime":"10"}]`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	s := &Service{settingsStore: readSettingsStub{}, newKlasClient: klas.NewClient, sessions: &fakeSessionStore{loadSession: func(context.Context, string) (klas.Session, error) { return klas.Session{}, nil }}}
	ref := CourseRef{TermValue: "2025,1", CourseID: "c"}
	a, _ := StableLectureID(ref, "lrn-1")
	b, _ := StableLectureID(ref, "lrn-2")
	missing, _ := StableLectureID(ref, "lrn-3")
	ids := []string{a, b, a, missing}
	v, e := s.GetItems(context.Background(), UserOption{StudentID: "student"}, ids, []string{"completion"})
	if e != nil || len(v.Items) != 3 || reads != 1 {
		t.Fatalf("%+v err=%v reads=%d", v, e, reads)
	}
	if v.Items[0].Sections["completion"].(Completion).Status != "incomplete" || v.Items[1].Sections["completion"].(Completion).Status != "completed" || v.Items[2].Status != "not_found" {
		t.Fatalf("%+v", v)
	}
	progress = "10"
	v, e = s.GetItems(context.Background(), UserOption{StudentID: "student"}, []string{a}, []string{"completion"})
	if e != nil || reads != 2 || v.Items[0].Sections["completion"].(Completion).Status != "completed" {
		t.Fatalf("not fresh: %+v %v reads=%d", v, e, reads)
	}
}
