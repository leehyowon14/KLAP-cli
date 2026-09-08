package klas

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCourseAdapterMapsPrivateWireToSharedModels(t *testing.T) {
	for _, tt := range []struct{ name, body, want string }{
		{"null terms", "null", "null"},
		{"empty terms", "[]", "[]"},
		{"null courses", `[{"label":"term","value":"2026,1","subjList":null}]`, `[{"label":"term","value":"2026,1","subjList":null}]`},
		{"empty courses", `[{"label":"term","value":"2026,1","subjList":[]}]`, `[{"label":"term","value":"2026,1","subjList":[]}]`},
		{"normalized courses", `[{"label":"term","value":"2026,1","subjList":[{"name":"course","value":"subject","unused":"private"}],"unused":"private"}]`, `[{"label":"term","value":"2026,1","subjList":[{"name":"course","value":"subject"}]}]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			terms, err := schemaResponseClient(tt.body).Courses(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(terms)
			if err != nil || string(encoded) != tt.want {
				t.Fatalf("JSON=%s want=%s error=%v", encoded, tt.want, err)
			}
			var roundTrip []Term
			if err := json.Unmarshal(encoded, &roundTrip); err != nil {
				t.Fatal(err)
			}
			encoded, err = json.Marshal(roundTrip)
			if err != nil || string(encoded) != tt.want {
				t.Fatalf("round trip=%s error=%v", encoded, err)
			}
		})
	}
}
