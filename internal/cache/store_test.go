package cache

import (
	"testing"
	"time"
)

type testValue struct {
	Name string `json:"name"`
}

func TestStoreGetSetAndExpire(t *testing.T) {
	store, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}

	if err := store.Set("key", time.Minute, testValue{Name: "cached"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	var got testValue
	if _, ok, err := store.Get("key", &got); err != nil || !ok || got.Name != "cached" {
		t.Fatalf("Get() = %+v, %v, %v", got, ok, err)
	}

	if err := store.Set("expired", time.Nanosecond, testValue{Name: "old"}); err != nil {
		t.Fatalf("Set() expired error = %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, ok, err := store.Get("expired", &got); err != nil || ok {
		t.Fatalf("Get() expired = %v, %v", ok, err)
	}
}

func TestStoreStatsAndClear(t *testing.T) {
	store, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	if err := store.Set("key", time.Minute, testValue{Name: "cached"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Files != 1 || stats.Bytes <= 0 {
		t.Fatalf("Stats() = %+v", stats)
	}

	removed, err := store.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Clear() removed = %d", removed)
	}
}
