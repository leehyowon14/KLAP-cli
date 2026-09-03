package cache

import (
	"encoding/json"
	"os"
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

func TestStoreClearPrefix(t *testing.T) {
	store, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	if err := store.Set("assignment:v1:user:2026,1", time.Minute, testValue{Name: "assignment"}); err != nil {
		t.Fatalf("Set() assignment error = %v", err)
	}
	if err := store.Set("notice:v1:user:2026,1", time.Minute, testValue{Name: "notice"}); err != nil {
		t.Fatalf("Set() notice error = %v", err)
	}

	removed, err := store.ClearPrefix("assignment:")
	if err != nil {
		t.Fatalf("ClearPrefix() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("ClearPrefix() removed = %d", removed)
	}

	var got testValue
	if _, ok, err := store.Get("assignment:v1:user:2026,1", &got); err != nil || ok {
		t.Fatalf("Get() assignment = %v, %v", ok, err)
	}
	if _, ok, err := store.Get("notice:v1:user:2026,1", &got); err != nil || !ok {
		t.Fatalf("Get() notice = %v, %v", ok, err)
	}
}

func TestStoreClearExceptPrefixes(t *testing.T) {
	store, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	values := map[string]string{
		"assignment:v1:user:2026,1":           "assignment",
		"notice:v1:user:2026,1":               "notice",
		"sync-source:user:assignment":         "assignment state",
		"sync-source:global:academic":         "academic state",
		"sync-source-metadata:user:migration": "ordinary cache",
	}
	for key, name := range values {
		if err := store.Set(key, time.Minute, testValue{Name: name}); err != nil {
			t.Fatalf("Set(%q) error = %v", key, err)
		}
	}

	removed, err := store.ClearExceptPrefixes("sync-source:")
	if err != nil {
		t.Fatalf("ClearExceptPrefixes() error = %v", err)
	}
	if removed != 3 {
		t.Fatalf("ClearExceptPrefixes() removed = %d, want 3", removed)
	}

	for key := range values {
		var got testValue
		_, ok, err := store.Get(key, &got)
		if err != nil {
			t.Fatalf("Get(%q) error = %v", key, err)
		}
		want := key == "sync-source:user:assignment" || key == "sync-source:global:academic"
		if ok != want {
			t.Fatalf("Get(%q) hit = %v, want %v", key, ok, want)
		}
	}
}

func TestStoreClearRejectsMismatchedKey(t *testing.T) {
	tests := map[string]func(*Store) (int, error){
		"except prefixes": func(store *Store) (int, error) {
			return store.ClearExceptPrefixes("sync-source:")
		},
		"matching prefix": func(store *Store) (int, error) {
			return store.ClearPrefix("assignment:")
		},
	}
	for name, clear := range tests {
		t.Run(name, func(t *testing.T) {
			store, path := storeWithMismatchedSyncKey(t)
			if _, err := clear(store); err == nil {
				t.Fatal("clear expected mismatched key error")
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("corrupted baseline must be preserved: %v", err)
			}
		})
	}
}

func TestStoreDeleteRemovesOnlyExactKey(t *testing.T) {
	store, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	for _, key := range []string{"sync-source:user:assignment", "sync-source:user:lecture"} {
		if err := store.Set(key, time.Hour, key); err != nil {
			t.Fatalf("Set(%q) error = %v", key, err)
		}
	}
	removed, err := store.Delete("sync-source:user:assignment")
	if err != nil || !removed {
		t.Fatalf("Delete() = %v, %v", removed, err)
	}
	var value string
	if _, ok, err := store.Get("sync-source:user:assignment", &value); err != nil || ok {
		t.Fatalf("Get(deleted) = %q, %v, %v", value, ok, err)
	}
	if _, ok, err := store.Get("sync-source:user:lecture", &value); err != nil || !ok || value != "sync-source:user:lecture" {
		t.Fatalf("Get(preserved) = %q, %v, %v", value, ok, err)
	}
	removed, err = store.Delete("sync-source:user:assignment")
	if err != nil || removed {
		t.Fatalf("Delete(missing) = %v, %v", removed, err)
	}
}

func storeWithMismatchedSyncKey(t *testing.T) (*Store, string) {
	t.Helper()
	store, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	key := "sync-source:user:assignment"
	if err := store.Set(key, time.Minute, testValue{Name: "baseline"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	path := store.pathFor(key)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var cached entry
	if err := json.Unmarshal(body, &cached); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	cached.Key = "assignment:v1:user:2026,1"
	body, err = json.Marshal(cached)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return store, path
}
