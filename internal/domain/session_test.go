package domain

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestSessionPersistedSchema(t *testing.T) {
	for _, fixture := range []string{
		`{"userId":"user","cookies":{"SESSION":"cookie","WMONID":"monitor"},"createdAt":"2026-09-01T01:02:03Z"}`,
		`{"userId":"","cookies":null,"createdAt":"0001-01-01T00:00:00Z"}`,
		`{"userId":"","cookies":{},"createdAt":"0001-01-01T00:00:00Z"}`,
	} {
		var session Session
		if err := json.Unmarshal([]byte(fixture), &session); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(session)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != fixture {
			t.Fatalf("schema changed: %s", encoded)
		}
		var restored Session
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(session, restored) {
			t.Fatalf("roundtrip: %#v", restored)
		}
	}
	var legacy Session
	if err := json.Unmarshal([]byte(`{"userId":"legacy","cookies":{"SESSION":"saved"}}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.UserID != "legacy" || legacy.Cookies["SESSION"] != "saved" || legacy.CreatedAt != (time.Time{}) {
		t.Fatalf("legacy: %#v", legacy)
	}
}
