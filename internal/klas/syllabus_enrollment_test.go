package klas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSyllabusEnrollment(t *testing.T) {
	for _, tc := range []struct {
		body, want string
		status     int
	}{
		{`{"currentNum":"42"}`, "42", 200},
		{`{"currentNum":12}`, "12", 200},
		{`{"currentNum":"0"}`, "0", 200},
		{`{}`, "", 200}, {`{"currentNum":null}`, "", 200},
		{`{"currentNum":-1}`, "", 200}, {`{"currentNum":1.5}`, "", 200},
		{`<html>login</html>`, "", 200}, {`{}`, "", 500},
	} {
		t.Run(tc.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/std/cps/atnlc/popup/LectrePlanStdCrtNum.do" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				var payload map[string]string
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if payload["selectYear"] != "2026" || payload["selecthakgi"] != "2" || payload["selectSubj"] != "subject" || payload["selectYearHakgi"] != "2026,2" {
					t.Errorf("payload=%v", payload)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			base, _ := url.Parse(server.URL)
			c := &Client{httpClient: server.Client(), baseURL: base}
			got, err := c.SyllabusEnrollment(context.Background(), "2026,2", "subject", "자료구조")
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("count=%q err=%v", got, err)
			}
		})
	}
}
