package syncstate

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStoreRoundTripPreservesIndependentSources(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	assignment := State{Items: map[string]Item{
		"assignment-id": {Hash: "assignment-hash", IgnoredHash: "ignored"},
	}}
	academic := State{Items: map[string]Item{
		"academic-id": {Hash: "academic-hash"},
	}}
	if err := store.Save("assignment", "20260001", assignment); err != nil {
		t.Fatalf("Save(assignment) error = %v", err)
	}
	if err := store.Save("academic", "global", academic); err != nil {
		t.Fatalf("Save(academic) error = %v", err)
	}

	gotAssignment, ok, err := store.Load("assignment", "20260001")
	if err != nil || !ok || !reflect.DeepEqual(gotAssignment, assignment) {
		t.Fatalf("Load(assignment) = %+v, %v, %v", gotAssignment, ok, err)
	}
	gotAcademic, ok, err := store.Load("academic", "global")
	if err != nil || !ok || !reflect.DeepEqual(gotAcademic, academic) {
		t.Fatalf("Load(academic) = %+v, %v, %v", gotAcademic, ok, err)
	}
	missing, ok, err := store.Load("lecture", "20260001")
	if err != nil || ok || missing.Items == nil || len(missing.Items) != 0 {
		t.Fatalf("Load(missing) = %+v, %v, %v", missing, ok, err)
	}

	body, err := os.ReadFile(filepath.Join(dir, "sync-state.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(body), `"version": 1`) {
		t.Fatalf("sync state file missing version: %s", body)
	}
	info, err := os.Stat(filepath.Join(dir, "sync-state.json"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("sync state mode = %o", info.Mode().Perm())
	}
}

func TestStoresAtSamePathSerializeUpdates(t *testing.T) {
	dir := t.TempDir()
	first, err := NewStoreAt(dir)
	if err != nil {
		t.Fatalf("NewStoreAt(first) error = %v", err)
	}
	second, err := NewStoreAt(dir)
	if err != nil {
		t.Fatalf("NewStoreAt(second) error = %v", err)
	}
	release, err := acquireStateFileLock(first.path + ".lock")
	if err != nil {
		t.Fatalf("acquireStateFileLock() error = %v", err)
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		done <- second.Save("lecture", "20260001", State{Items: map[string]Item{"lecture-id": {Hash: "lecture-hash"}}})
	}()
	<-started
	select {
	case err := <-done:
		t.Fatalf("Save() did not wait for file lock: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := release(); err != nil {
		t.Fatalf("release() error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Save() after release error = %v", err)
	}

	var waitGroup sync.WaitGroup
	errorsBySource := make(chan error, 2)
	for _, operation := range []func() error{
		func() error {
			return first.Save("assignment", "20260001", State{Items: map[string]Item{"assignment-id": {Hash: "assignment-hash"}}})
		},
		func() error {
			return second.Save("academic", "global", State{Items: map[string]Item{"academic-id": {Hash: "academic-hash"}}})
		},
	} {
		waitGroup.Add(1)
		go func(operation func() error) {
			defer waitGroup.Done()
			errorsBySource <- operation()
		}(operation)
	}
	waitGroup.Wait()
	close(errorsBySource)
	for err := range errorsBySource {
		if err != nil {
			t.Fatalf("concurrent Save() error = %v", err)
		}
	}
	for _, check := range []struct {
		scope string
		owner string
		id    string
		hash  string
	}{
		{scope: "lecture", owner: "20260001", id: "lecture-id", hash: "lecture-hash"},
		{scope: "assignment", owner: "20260001", id: "assignment-id", hash: "assignment-hash"},
		{scope: "academic", owner: "global", id: "academic-id", hash: "academic-hash"},
	} {
		state, ok, err := first.Load(check.scope, check.owner)
		if err != nil || !ok || state.Items[check.id].Hash != check.hash {
			t.Fatalf("Load(%s, %s) = %+v, %v, %v", check.scope, check.owner, state, ok, err)
		}
	}
}

func TestStoreRejectsCorruptAndUnknownSchemaWithoutOverwrite(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "corrupt", body: `{not-json`},
		{name: "unknown-version", body: `{"version":99,"sources":{}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "sync-state.json")
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			store, err := NewStoreAt(dir)
			if err != nil {
				t.Fatalf("NewStoreAt() error = %v", err)
			}
			if _, _, err := store.Load("assignment", "20260001"); err == nil {
				t.Fatal("Load() expected error")
			}
			if err := store.Save("assignment", "20260001", emptyState()); err == nil {
				t.Fatal("Save() expected error")
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if string(body) != test.body {
				t.Fatalf("corrupt state overwritten: %q", body)
			}
		})
	}
}

func TestAtomicWriteFailurePreservesPreviousState(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	original := State{Items: map[string]Item{"id": {Hash: "old"}}}
	if err := store.Save("assignment", "20260001", original); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}
	wantErr := errors.New("interrupted before rename")
	store.writer = filesystemAtomicWriter{
		replace: func(string, string) error { return wantErr },
		recover: recoverStateFile,
	}
	if err := store.Save("assignment", "20260001", State{Items: map[string]Item{"id": {Hash: "new"}}}); !errors.Is(err, wantErr) {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := NewStoreAt(dir)
	if err != nil {
		t.Fatalf("NewStoreAt() reload error = %v", err)
	}
	got, ok, err := reloaded.Load("assignment", "20260001")
	if err != nil || !ok || !reflect.DeepEqual(got, original) {
		t.Fatalf("Load() after interrupted write = %+v, %v, %v", got, ok, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("temporary file remains: %s", entry.Name())
		}
	}
}

func TestDirectorySyncFailureIsNotReportedAsDurableSave(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	wantErr := errors.New("directory fsync failed")
	store.writer = filesystemAtomicWriter{
		replace: replaceStateFile,
		recover: recoverStateFile,
		syncDir: func(string) error { return wantErr },
	}
	if err := store.Save("assignment", "20260001", State{Items: map[string]Item{"id": {Hash: "new"}}}); !errors.Is(err, wantErr) {
		t.Fatalf("Save() error = %v", err)
	}
}
