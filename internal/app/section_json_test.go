package app

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestSectionErrorJSONRetainsPartialFailure(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{{errors.New("temporary failure"), `{"Section":"과제","Err":"temporary failure"}`}, {nil, `{"Section":"과제","Err":""}`}} {
		body, e := json.Marshal(DashboardSectionError{Section: "과제", Err: tc.err})
		if e != nil || string(body) != tc.want {
			t.Fatalf("%s: %v", body, e)
		}
	}
}
