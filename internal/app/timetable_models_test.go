package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

func TestTimetableModelsMapAdapterFixturesWithoutWireFields(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/std/cps/atnlc/TimetableStdList.do" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		body := `[{"wtHasSchedule":"Y","wtTime":2,"wtSubj_1":"subject","wtSubjNm_1":"course","wtSpan_1":2,"wtLocHname_1":"room","wtProfNm_1":"professor","private":"omit"},{"wtHasSchedule":"Y","wtTime":9,"wtSubj_2":"online","wtSubjNm_2":"online course"}]`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	client, err := klas.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := client.Timetable(context.Background(), "2026,1")
	if err != nil {
		t.Fatal(err)
	}
	models := timetableEntryModels(entries)
	if len(models) != 2 || models[0] != (TimetableEntry{SubjectID: "subject", SubjectName: "course", Weekday: 1, Period: 2, Span: 2, Room: "room", Professor: "professor"}) || !models[1].Online {
		t.Fatalf("models=%+v", models)
	}
	for index, entry := range entries {
		assertModelJSONParity(t, entry, models[index])
	}
	if timetableEntryModels(nil) != nil {
		t.Fatal("nil slice changed")
	}
	if models := timetableEntryModels([]klas.TimetableEntry{}); models == nil || len(models) != 0 {
		t.Fatal("empty slice changed")
	}
}
