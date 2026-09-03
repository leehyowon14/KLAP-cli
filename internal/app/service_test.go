package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kw-klap/klap-cli/internal/account"
	"github.com/kw-klap/klap-cli/internal/cache"
	klapcalendar "github.com/kw-klap/klap-cli/internal/calendar"
	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/reminder"
	"github.com/kw-klap/klap-cli/internal/settings"
)

var errDashboardTest = errors.New("dashboard test error")

type fakeSessionStore struct {
	loadPassword func(context.Context, string) (string, error)
	loadSession  func(context.Context, string) (klas.Session, error)
	saveSession  func(context.Context, string, klas.Session) error
}

func (s *fakeSessionStore) LoadPassword(ctx context.Context, studentID string) (string, error) {
	return s.loadPassword(ctx, studentID)
}

func (s *fakeSessionStore) LoadSession(ctx context.Context, studentID string) (klas.Session, error) {
	return s.loadSession(ctx, studentID)
}

func (s *fakeSessionStore) SaveSession(ctx context.Context, studentID string, session klas.Session) error {
	return s.saveSession(ctx, studentID, session)
}

func TestNewServicePropagatesSettingsStoreFailure(t *testing.T) {
	wantErr := errors.New("settings unavailable")
	cacheFactoryCalled := false

	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) {
			return nil, wantErr
		},
		cache: func() (*cache.Store, error) {
			cacheFactoryCalled = true
			return nil, nil
		},
	})

	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "settings store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
	if cacheFactoryCalled {
		t.Fatal("cache factory should not run after settings factory failure")
	}
}

func TestCourseRefStableResourceIDIgnoresCourseOrder(t *testing.T) {
	termValue := "2026,1"
	firstOrder := []klas.Course{
		{Name: "컴퓨터그래픽스", Value: "U202613951I040013"},
		{Name: "오픈소스소프트웨어실습", Value: "U202613951I040014"},
	}
	secondOrder := []klas.Course{firstOrder[1], firstOrder[0]}

	firstRef, err := NewCourseRef(termValue, firstOrder[0])
	if err != nil {
		t.Fatalf("NewCourseRef() first error = %v", err)
	}
	secondRef, err := NewCourseRef(termValue, secondOrder[1])
	if err != nil {
		t.Fatalf("NewCourseRef() second error = %v", err)
	}
	firstID, err := stableCourseResourceID("assignment", firstRef, "7")
	if err != nil {
		t.Fatalf("stableCourseResourceID() first error = %v", err)
	}
	secondID, err := stableCourseResourceID("assignment", secondRef, "7")
	if err != nil {
		t.Fatalf("stableCourseResourceID() second error = %v", err)
	}
	if firstID != secondID {
		t.Fatalf("stable ID changed after reorder: %q != %q", firstID, secondID)
	}

	parsedRef, remoteParts, stable, err := parseStableCourseResourceID("assignment", firstID, 1)
	if err != nil || !stable {
		t.Fatalf("parseStableCourseResourceID() = %+v, %v, %v", parsedRef, stable, err)
	}
	if parsedRef != firstRef || len(remoteParts) != 1 || remoteParts[0] != "7" {
		t.Fatalf("parseStableCourseResourceID() = %+v, %v", parsedRef, remoteParts)
	}
}

func TestCourseRefStableResourceIDSeparatesTermsAndCourses(t *testing.T) {
	course := klas.Course{Name: "컴퓨터그래픽스", Value: "course:id/01"}
	firstRef, err := NewCourseRef("2026,1", course)
	if err != nil {
		t.Fatalf("NewCourseRef() first error = %v", err)
	}
	secondRef, err := NewCourseRef("2026,2", course)
	if err != nil {
		t.Fatalf("NewCourseRef() second error = %v", err)
	}
	otherCourseRef, err := NewCourseRef("2026,1", klas.Course{Name: "다른 과목", Value: "course:id/02"})
	if err != nil {
		t.Fatalf("NewCourseRef() other course error = %v", err)
	}
	firstID, _ := stableCourseResourceID("lecture", firstRef, "content:id/1")
	secondID, _ := stableCourseResourceID("lecture", secondRef, "content:id/1")
	otherCourseID, _ := stableCourseResourceID("lecture", otherCourseRef, "content:id/1")
	if firstID == secondID || firstID == otherCourseID || secondID == otherCourseID {
		t.Fatalf("stable IDs collide: %q, %q, %q", firstID, secondID, otherCourseID)
	}

	parsedRef, remoteParts, stable, err := parseStableCourseResourceID("lecture", firstID, 1)
	if err != nil || !stable || parsedRef != firstRef || len(remoteParts) != 1 || remoteParts[0] != "content:id/1" {
		t.Fatalf("parseStableCourseResourceID() = %+v, %v, %v, %v", parsedRef, remoteParts, stable, err)
	}
}

func TestCourseRefRejectsMissingAndMalformedValues(t *testing.T) {
	if _, err := NewCourseRef("", klas.Course{Value: "course"}); err == nil {
		t.Fatal("NewCourseRef() expected missing term error")
	}
	if _, err := NewCourseRef("2026,1", klas.Course{}); err == nil {
		t.Fatal("NewCourseRef() expected missing course error")
	}
	if _, _, stable, err := parseStableCourseResourceID("assignment", "assignment:v1:not-base64!:Y291cnNl:Nw", 1); err == nil || !stable {
		t.Fatalf("parseStableCourseResourceID() malformed = stable %v, error %v", stable, err)
	}
	if _, _, stable, err := parseStableCourseResourceID("assignment", "3:7", 1); err != nil || stable {
		t.Fatalf("parseStableCourseResourceID() legacy = stable %v, error %v", stable, err)
	}
}

func TestNewServicePropagatesCacheStoreFailure(t *testing.T) {
	wantErr := errors.New("cache unavailable")

	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) {
			return &settings.Store{}, nil
		},
		cache: func() (*cache.Store, error) {
			return nil, wantErr
		},
	})

	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "cache store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
}

func TestNewServiceInitializesStores(t *testing.T) {
	accountStore := &account.Store{}
	settingsStore := &settings.Store{}
	cacheStore := &cache.Store{}

	service, err := newService(accountStore, serviceStoreFactories{
		settings: func() (*settings.Store, error) { return settingsStore, nil },
		cache:    func() (*cache.Store, error) { return cacheStore, nil },
	})
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}
	if service.store != accountStore || service.sessions != accountStore || service.settingsStore != settingsStore || service.cacheStore != cacheStore {
		t.Fatalf("newService() stores = %+v", service)
	}
	if service.newKlasClient == nil || service.login == nil {
		t.Fatal("newService() authentication dependencies are nil")
	}
}

func TestAuthenticatedClientUsesStoredSessionWithoutSaving(t *testing.T) {
	storedSession := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "saved"}}
	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	saveCalls := 0
	loginCalls := 0
	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return storedSession, nil
			},
			loadPassword: func(context.Context, string) (string, error) {
				return "", errors.New("LoadPassword should not be called")
			},
			saveSession: func(context.Context, string, klas.Session) error {
				saveCalls++
				return nil
			},
		},
		newKlasClient: func() (*klas.Client, error) { return client, nil },
		login: func(context.Context, *klas.Client, string, string) (klas.Session, error) {
			loginCalls++
			return klas.Session{}, nil
		},
	}

	got, err := service.authenticatedClient(context.Background(), "20260001")
	if err != nil {
		t.Fatalf("authenticatedClient() error = %v", err)
	}
	if got != client {
		t.Fatalf("authenticatedClient() client = %p, want %p", got, client)
	}
	if saveCalls != 0 || loginCalls != 0 {
		t.Fatalf("authenticatedClient() save calls = %d, login calls = %d", saveCalls, loginCalls)
	}
}

func TestAuthenticatedClientPropagatesRefreshedSessionSaveFailure(t *testing.T) {
	wantErr := errors.New("keychain unavailable")
	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	refreshedSession := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "refreshed"}}
	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return klas.Session{}, errors.New("stored session missing")
			},
			loadPassword: func(_ context.Context, studentID string) (string, error) {
				if studentID != "20260001" {
					t.Fatalf("LoadPassword() studentID = %q", studentID)
				}
				return "password", nil
			},
			saveSession: func(_ context.Context, studentID string, session klas.Session) error {
				if studentID != "20260001" || session.UserID != refreshedSession.UserID {
					t.Fatalf("SaveSession() = %q, %+v", studentID, session)
				}
				return wantErr
			},
		},
		newKlasClient: func() (*klas.Client, error) { return client, nil },
		login: func(_ context.Context, gotClient *klas.Client, studentID string, password string) (klas.Session, error) {
			if gotClient != client || studentID != "20260001" || password != "password" {
				t.Fatalf("login() = %p, %q, %q", gotClient, studentID, password)
			}
			return refreshedSession, nil
		},
	}

	got, err := service.authenticatedClient(context.Background(), "20260001")
	if got != nil {
		t.Fatalf("authenticatedClient() client = %p, want nil", got)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "갱신 세션 저장 실패") {
		t.Fatalf("authenticatedClient() error = %v", err)
	}
}

func TestRefreshedClientPropagatesSessionSaveFailure(t *testing.T) {
	wantErr := errors.New("keychain unavailable")
	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	refreshedSession := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "refreshed"}}
	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return klas.Session{}, errors.New("LoadSession should not be called")
			},
			loadPassword: func(context.Context, string) (string, error) {
				return "password", nil
			},
			saveSession: func(context.Context, string, klas.Session) error {
				return wantErr
			},
		},
		newKlasClient: func() (*klas.Client, error) { return client, nil },
		login: func(context.Context, *klas.Client, string, string) (klas.Session, error) {
			return refreshedSession, nil
		},
	}

	got, refreshed, err := service.refreshedClientAfterSessionError(context.Background(), "20260001", klas.ErrSessionExpired)
	if got != nil {
		t.Fatalf("refreshedClientAfterSessionError() client = %p, want nil", got)
	}
	if !refreshed {
		t.Fatal("refreshedClientAfterSessionError() refreshed = false, want true")
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "갱신 세션 저장 실패") {
		t.Fatalf("refreshedClientAfterSessionError() error = %v", err)
	}
}

func TestPrepareReminderSyncPromptsOnceForChangedSource(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	original := reminder.Assignment{ID: "assignment:1", Title: "기말 과제", Course: "오픈소스", DueAt: &dueAt, Notes: "old"}

	first, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{original}, nil)
	if err != nil {
		t.Fatalf("prepareReminderSync() first error = %v", err)
	}
	if len(first.Conflicts) != 0 || len(first.Assignments) != 1 || first.Assignments[0].ForceUpdate {
		t.Fatalf("first prepare = %+v", first)
	}
	commitSyncSourceHashes(&first.State, first.Pending, []string{original.ID})
	if err := service.saveSyncSourceState("assignment", "20260001", first.State); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}

	changed := original
	changed.Title = "기말 대체 과제"
	second, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{changed}, nil)
	if err != nil {
		t.Fatalf("prepareReminderSync() second error = %v", err)
	}
	if len(second.Conflicts) != 1 || second.Conflicts[0].Key != "assignment:assignment:1" {
		t.Fatalf("second conflicts = %+v", second.Conflicts)
	}
	applied, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{changed}, map[string]SyncDecision{
		second.Conflicts[0].Key: SyncDecisionApply,
	})
	if err != nil {
		t.Fatalf("prepareReminderSync() apply error = %v", err)
	}
	if len(applied.Assignments) != 1 || !applied.Assignments[0].ForceUpdate {
		t.Fatalf("applied prepare = %+v", applied)
	}

	kept, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{changed}, map[string]SyncDecision{
		second.Conflicts[0].Key: SyncDecisionKeep,
	})
	if err != nil {
		t.Fatalf("prepareReminderSync() keep error = %v", err)
	}
	if len(kept.Conflicts) != 0 || len(kept.Assignments) != 0 || kept.Skipped != 1 {
		t.Fatalf("kept prepare = %+v", kept)
	}
	if err := service.saveSyncSourceState("assignment", "20260001", kept.State); err != nil {
		t.Fatalf("save keep state error = %v", err)
	}

	again, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{changed}, nil)
	if err != nil {
		t.Fatalf("prepareReminderSync() again error = %v", err)
	}
	if len(again.Conflicts) != 0 || len(again.Assignments) != 0 || again.Skipped != 1 {
		t.Fatalf("again prepare = %+v", again)
	}
}

func TestPrepareReminderSyncMigratesLegacyAssignmentBaseline(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	stableID := "assignment:v1:MjAyNiwx:Y291cnNlLTE:Nw"
	legacy := reminder.Assignment{
		ID:        "1:7",
		Title:     "기말 과제",
		Course:    "오픈소스",
		DueAt:     &dueAt,
		DetailURL: "https://example.test/assignment/7",
		Notes:     "--- KLAP ---\n\nID: 1:7\n과목: 오픈소스",
	}
	state := syncSourceState{Items: map[string]syncSourceItem{
		legacy.ID: {Hash: reminderSourceHash(legacy)},
	}}
	if err := service.saveSyncSourceState("assignment", "20260001", state); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}

	stable := legacy
	stable.ID = stableID
	stable.LegacyIDs = []string{legacy.ID}
	stable.Notes = replaceReminderNoteID(stable.Notes, stable.ID)
	prepared, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{stable}, nil)
	if err != nil {
		t.Fatalf("prepareReminderSync() error = %v", err)
	}
	if len(prepared.Conflicts) != 0 || len(prepared.Assignments) != 1 {
		t.Fatalf("prepareReminderSync() = %+v", prepared)
	}
	if prepared.Assignments[0].KnownSourceHash != reminderSourceHash(stable) {
		t.Fatalf("KnownSourceHash = %q, want migrated stable hash", prepared.Assignments[0].KnownSourceHash)
	}
	if _, exists := prepared.State.Items[legacy.ID]; exists {
		t.Fatalf("legacy state key still exists: %+v", prepared.State.Items)
	}
	if got := prepared.State.Items[stable.ID].Hash; got != reminderSourceHash(stable) {
		t.Fatalf("stable state hash = %q", got)
	}
}

func TestPrepareCalendarSyncWithoutBaselineDoesNotForceUpdate(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	event := klapcalendar.Event{
		ID:      "academic:2026:1",
		Title:   "개강",
		StartAt: time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local),
		EndAt:   time.Date(2026, 3, 2, 23, 59, 0, 0, time.Local),
	}

	prepared, err := service.prepareCalendarSync("academic", "global", []klapcalendar.Event{event}, nil)
	if err != nil {
		t.Fatalf("prepareCalendarSync() error = %v", err)
	}
	if len(prepared.Conflicts) != 0 || len(prepared.Events) != 1 {
		t.Fatalf("prepareCalendarSync() = %+v", prepared)
	}
	if prepared.Events[0].ForceUpdate || prepared.Events[0].KnownSourceHash != "" {
		t.Fatalf("event without baseline must not force update: %+v", prepared.Events[0])
	}
}

func TestPrepareSyncRejectsCorruptedBaseline(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	if err := cacheStore.Set(syncSourceCacheKey("assignment", "20260001"), time.Minute, "invalid state"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	assignment := reminder.Assignment{ID: "assignment:1", Title: "기말 과제"}

	prepared, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{assignment}, nil)
	if err == nil {
		t.Fatal("prepareReminderSync() expected corrupted baseline error")
	}
	if len(prepared.Assignments) != 0 || len(prepared.Pending) != 0 {
		t.Fatalf("prepareReminderSync() returned work after baseline error: %+v", prepared)
	}
}

func TestCommitSyncSourceHashesRecordsOnlySyncedIDs(t *testing.T) {
	state := syncSourceState{Items: map[string]syncSourceItem{}}
	pending := map[string]string{
		"created":          "created-hash",
		"existing-skipped": "skipped-hash",
	}

	commitSyncSourceHashes(&state, pending, []string{"created", "unknown"})

	if got := state.Items["created"].Hash; got != "created-hash" {
		t.Fatalf("created hash = %q, want created-hash", got)
	}
	if _, ok := state.Items["existing-skipped"]; ok {
		t.Fatal("existing item skipped without baseline must remain untracked")
	}
	if _, ok := state.Items["unknown"]; ok {
		t.Fatal("bridge ID without pending hash must be ignored")
	}
}

func TestSkippedExistingWithoutBaselineRemainsProtected(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	assignment := reminder.Assignment{ID: "assignment:1", Title: "기말 과제"}

	first, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{assignment}, nil)
	if err != nil {
		t.Fatalf("first prepareReminderSync() error = %v", err)
	}
	commitSyncSourceHashes(&first.State, first.Pending, nil)
	if err := service.saveSyncSourceState("assignment", "20260001", first.State); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}

	second, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{assignment}, nil)
	if err != nil {
		t.Fatalf("second prepareReminderSync() error = %v", err)
	}
	if len(second.Assignments) != 1 || second.Assignments[0].ForceUpdate || second.Assignments[0].KnownSourceHash != "" {
		t.Fatalf("second prepare must keep missing baseline protection: %+v", second)
	}
}

func TestClearCachePreservesSyncSourceState(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	want := syncSourceState{Items: map[string]syncSourceItem{
		"assignment:1": {Hash: "source-hash"},
	}}
	if err := service.saveSyncSourceState("assignment", "20260001", want); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}
	if err := cacheStore.Set("assignment:v1:20260001:2026,1:", time.Minute, struct {
		Name string `json:"name"`
	}{Name: "cached"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	result, err := service.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}
	if result.Removed != 1 {
		t.Fatalf("ClearCache() removed = %d, want 1", result.Removed)
	}
	got, err := service.loadSyncSourceState("assignment", "20260001")
	if err != nil {
		t.Fatalf("loadSyncSourceState() error = %v", err)
	}
	if got.Items["assignment:1"].Hash != "source-hash" {
		t.Fatalf("sync state after ClearCache() = %+v", got)
	}
}

func TestClearCacheScopePreservesSyncSourceState(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	want := syncSourceState{Items: map[string]syncSourceItem{
		"assignment:1": {Hash: "source-hash"},
	}}
	if err := service.saveSyncSourceState("assignment", "20260001", want); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}
	if err := cacheStore.Set("assignment:v1:20260001:2026,1:", time.Minute, struct {
		Name string `json:"name"`
	}{Name: "cached"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	result, err := service.ClearCacheScope("assignment")
	if err != nil {
		t.Fatalf("ClearCacheScope() error = %v", err)
	}
	if result.Removed != 1 {
		t.Fatalf("ClearCacheScope() removed = %d, want 1", result.Removed)
	}
	got, err := service.loadSyncSourceState("assignment", "20260001")
	if err != nil {
		t.Fatalf("loadSyncSourceState() error = %v", err)
	}
	if got.Items["assignment:1"].Hash != "source-hash" {
		t.Fatalf("sync state after ClearCacheScope() = %+v", got)
	}
}

func TestClearCacheScopeRejectsSyncSource(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore}
	want := syncSourceState{Items: map[string]syncSourceItem{
		"assignment:1": {Hash: "source-hash"},
	}}
	if err := service.saveSyncSourceState("assignment", "20260001", want); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}

	for _, scope := range []string{"sync-source", "SYNC-SOURCE", "Sync-Source:20260001"} {
		if _, err := service.ClearCacheScope(scope); err == nil {
			t.Fatalf("ClearCacheScope(%q) expected protected scope error", scope)
		}
	}
	got, err := service.loadSyncSourceState("assignment", "20260001")
	if err != nil {
		t.Fatalf("loadSyncSourceState() error = %v", err)
	}
	if got.Items["assignment:1"].Hash != "source-hash" {
		t.Fatalf("sync state after rejected clear = %+v", got)
	}
}

func TestSelectedCoursesByNumber(t *testing.T) {
	term := klas.Term{Courses: []klas.Course{
		{Name: "프로그래밍기초", Value: "c1"},
		{Name: "자료구조", Value: "c2"},
	}}

	selected, err := selectedCourses(term, "2")
	if err != nil {
		t.Fatalf("selectedCourses() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Index != 2 || selected[0].Course.Name != "자료구조" {
		t.Fatalf("selectedCourses() = %+v", selected)
	}
}

func TestSelectedCoursesByName(t *testing.T) {
	term := klas.Term{Courses: []klas.Course{
		{Name: "프로그래밍기초", Value: "c1"},
		{Name: "자료구조", Value: "c2"},
	}}

	selected, err := selectedCourses(term, "자료")
	if err != nil {
		t.Fatalf("selectedCourses() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Index != 2 {
		t.Fatalf("selectedCourses() = %+v", selected)
	}
}

func TestSelectTermRow(t *testing.T) {
	rows := []TermRow{
		{Index: 1, Term: klas.Term{Label: "2026년도 1학기", Value: "2026,1"}},
		{Index: 2, Term: klas.Term{Label: "2025년도 겨울학기", Value: "2025,4"}},
	}

	selected, err := selectTermRow(rows, "2")
	if err != nil {
		t.Fatalf("selectTermRow() by number error = %v", err)
	}
	if selected.Term.Value != "2025,4" {
		t.Fatalf("selectTermRow() by number = %+v", selected)
	}

	selected, err = selectTermRow(rows, "2026,1")
	if err != nil {
		t.Fatalf("selectTermRow() by value error = %v", err)
	}
	if selected.Term.Label != "2026년도 1학기" {
		t.Fatalf("selectTermRow() by value = %+v", selected)
	}

	selected, err = selectTermRow(rows, "겨울")
	if err != nil {
		t.Fatalf("selectTermRow() by label error = %v", err)
	}
	if selected.Term.Value != "2025,4" {
		t.Fatalf("selectTermRow() by label = %+v", selected)
	}
}

func TestNormalizeTermValue(t *testing.T) {
	got, err := normalizeTermValue("2026-1")
	if err != nil {
		t.Fatalf("normalizeTermValue() error = %v", err)
	}
	if got != "2026,1" {
		t.Fatalf("normalizeTermValue() = %q", got)
	}
	if _, err := normalizeTermValue("2026-5"); err == nil {
		t.Fatal("normalizeTermValue() expected error for invalid semester")
	}
}

func TestCurrentAcademicTermValue(t *testing.T) {
	tests := []struct {
		now  time.Time
		want string
	}{
		{now: time.Date(2026, time.June, 7, 0, 0, 0, 0, time.Local), want: "2026,1"},
		{now: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.Local), want: "2026,3"},
		{now: time.Date(2026, time.December, 20, 0, 0, 0, 0, time.Local), want: "2026,4"},
		{now: time.Date(2027, time.January, 10, 0, 0, 0, 0, time.Local), want: "2026,4"},
	}
	for _, tt := range tests {
		if got := currentAcademicTermValue(tt.now); got != tt.want {
			t.Fatalf("currentAcademicTermValue(%s) = %q, want %q", tt.now, got, tt.want)
		}
	}
}

func TestFutureTime(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
	past := now.Add(-time.Minute)
	same := now
	future := now.Add(time.Minute)

	if futureTime(nil, now) {
		t.Fatal("futureTime(nil) should be false")
	}
	if futureTime(&past, now) {
		t.Fatal("past deadline should not be synced")
	}
	if !futureTime(&same, now) || !futureTime(&future, now) {
		t.Fatal("current or future deadline should be synced")
	}
}

func TestDashboardCoursesBuildsCourseScopedSummary(t *testing.T) {
	due := time.Now().Add(24 * time.Hour)
	courses := []klas.Course{{Name: "컴퓨터그래픽스"}, {Name: "오픈소스소프트웨어실습"}}
	assignments := []AssignmentRow{
		{CourseName: "컴퓨터그래픽스", Assignment: klas.Assignment{Title: "과제1", DueAt: &due}},
		{CourseName: "오픈소스소프트웨어실습", Assignment: klas.Assignment{Title: "기말", DueAt: &due}},
	}
	lectures := []LectureRow{
		{CourseName: "오픈소스소프트웨어실습", Lecture: klas.Lecture{Title: "HuggingFace", ContentID: "a", Progress: "0", EndAt: &due}},
	}
	notices := []NoticeRow{
		{CourseName: "컴퓨터그래픽스", Notice: klas.Notice{Title: "공지"}},
	}
	attendance := DashboardAttendance{Rows: []DashboardAttendanceRow{{
		Course:    klas.AttendanceCourse{Name: "컴퓨터그래픽스"},
		Completed: 10,
	}}}

	got := dashboardCourses(courses, assignments, lectures, notices, attendance, DashboardEvaluation{})
	if len(got) != 2 {
		t.Fatalf("len(dashboardCourses) = %d", len(got))
	}
	if got[0].Name != "컴퓨터그래픽스" || len(got[0].Assignments) != 1 || len(got[0].Notices) != 1 || got[0].Attendance == nil {
		t.Fatalf("first dashboard course = %+v", got[0])
	}
	if got[1].Name != "오픈소스소프트웨어실습" || len(got[1].Assignments) != 1 || len(got[1].Lectures) != 1 {
		t.Fatalf("second dashboard course = %+v", got[1])
	}
}

func TestTermLabel(t *testing.T) {
	if got := termLabel("2026,3"); got != "2026년도 여름학기" {
		t.Fatalf("termLabel() = %q", got)
	}
	if got := termLabel("2026,4"); got != "2026년도 겨울학기" {
		t.Fatalf("termLabel() = %q", got)
	}
}

func TestLooksLikeSyllabusCourseCode(t *testing.T) {
	if !looksLikeSyllabusCourseCode("I040-3-3951-01") {
		t.Fatal("looksLikeSyllabusCourseCode() expected true")
	}
	if looksLikeSyllabusCourseCode("컴퓨터그래픽스") {
		t.Fatal("looksLikeSyllabusCourseCode() expected false")
	}
}

func TestCopyWithProgressReportsOffsetAndTotal(t *testing.T) {
	var dst bytes.Buffer
	var events []int64
	written, err := copyWithProgress(&dst, strings.NewReader("abcdef"), 4, 10, func(bytesWritten int64, totalBytes int64) {
		if totalBytes != 10 {
			t.Fatalf("totalBytes = %d, want 10", totalBytes)
		}
		events = append(events, bytesWritten)
	})
	if err != nil {
		t.Fatalf("copyWithProgress() error = %v", err)
	}
	if written != 6 || dst.String() != "abcdef" {
		t.Fatalf("copyWithProgress() = %d, %q", written, dst.String())
	}
	if len(events) == 0 || events[len(events)-1] != 10 {
		t.Fatalf("progress events = %v, want final 10", events)
	}
}

func TestResetConfigSettingsRestoresDefaults(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.SetConfigValue("reminder.name", "To-do"); err != nil {
		t.Fatalf("SetConfigValue(reminder.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("calendar.name", "시간표"); err != nil {
		t.Fatalf("SetConfigValue(calendar.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("timetable-calendar.name", "수업시간표"); err != nil {
		t.Fatalf("SetConfigValue(timetable-calendar.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("reminder.alarm-before-min", "60"); err != nil {
		t.Fatalf("SetConfigValue(reminder.alarm-before-min) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.concurrency", "9"); err != nil {
		t.Fatalf("SetConfigValue(download.concurrency) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.caffeinate", "false"); err != nil {
		t.Fatalf("SetConfigValue(download.caffeinate) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.keep-partial", "true"); err != nil {
		t.Fatalf("SetConfigValue(download.keep-partial) error = %v", err)
	}
	if _, err := service.SetConfigValue("transcript.concurrency", "3"); err != nil {
		t.Fatalf("SetConfigValue(transcript.concurrency) error = %v", err)
	}

	got, err := service.ResetConfigSettings()
	if err != nil {
		t.Fatalf("ResetConfigSettings() error = %v", err)
	}
	if got.Reminder.ListName != settings.DefaultReminderListName {
		t.Fatalf("Reminder.ListName = %q", got.Reminder.ListName)
	}
	if got.Reminder.AlarmBeforeMin != 24*60 {
		t.Fatalf("Reminder.AlarmBeforeMin = %d", got.Reminder.AlarmBeforeMin)
	}
	if got.Calendar.Name != settings.DefaultAcademicCalendarName {
		t.Fatalf("Calendar.Name = %q", got.Calendar.Name)
	}
	if got.Calendar.TimetableName != settings.DefaultTimetableCalendarName {
		t.Fatalf("Calendar.TimetableName = %q", got.Calendar.TimetableName)
	}
	if got.Download.Dir != settings.DefaultDownloadDir() || got.Download.Concurrency != settings.DefaultDownloadConcurrency {
		t.Fatalf("Download = %+v", got.Download)
	}
	if !got.Download.Caffeinate {
		t.Fatal("Download.Caffeinate should reset to true")
	}
	if got.Download.KeepPartial {
		t.Fatal("Download.KeepPartial should reset to false")
	}
	if got.Transcript.Concurrency != settings.DefaultTranscriptConcurrency {
		t.Fatalf("Transcript.Concurrency = %d", got.Transcript.Concurrency)
	}
	if got.Term.Value != "" {
		t.Fatalf("Term.Value = %q", got.Term.Value)
	}
}

func TestSetConfigValueRejectsInvalidTranscriptConcurrency(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.SetConfigValue("transcript.concurrency", "4"); err == nil {
		t.Fatal("SetConfigValue(transcript.concurrency) expected error")
	}
}

func TestTranscriptPath(t *testing.T) {
	if got := transcriptPath(filepath.Join("downloads", "컴퓨터그래픽스", "video", "lecture.mp4")); got != filepath.Join("downloads", "컴퓨터그래픽스", "transcription", "lecture.txt") {
		t.Fatalf("transcriptPath() = %q", got)
	}
	if got := transcriptPath(filepath.Join("downloads", "lecture")); got != filepath.Join("downloads", "lecture.txt") {
		t.Fatalf("transcriptPath() without extension = %q", got)
	}
}

func TestLectureDownloadItemNeedsTranscriptForSkippedVideo(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "컴퓨터그래픽스", "video", "lecture.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(video) error = %v", err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	item := LectureDownloadItem{Path: videoPath, Skipped: true}
	if !LectureDownloadItemNeedsTranscript(item) {
		t.Fatal("LectureDownloadItemNeedsTranscript() expected true without transcript")
	}

	outputPath := filepath.Join(root, "컴퓨터그래픽스", "transcription", "lecture.txt")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(transcript) error = %v", err)
	}
	if err := os.WriteFile(outputPath, []byte("text"), 0o644); err != nil {
		t.Fatalf("WriteFile(transcript) error = %v", err)
	}
	if LectureDownloadItemNeedsTranscript(item) {
		t.Fatal("LectureDownloadItemNeedsTranscript() expected false with existing transcript")
	}
}

func TestTranscribeDownloadedLecturesReturnsPreflightFailure(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "컴퓨터그래픽스")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile(blocker) error = %v", err)
	}
	videoPath := filepath.Join(blocker, "video", "lecture.mp4")
	row := LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "a", Title: "소개"}}

	result := (&Service{}).TranscribeDownloadedLectures(context.Background(), []LectureDownloadItem{{
		Lecture: row,
		Path:    videoPath,
	}}, LectureTranscriptOptions{})

	if len(result.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(result.Items))
	}
	if result.Items[0].Err == nil {
		t.Fatalf("Items[0].Err is nil")
	}
	if result.Items[0].InputPath != videoPath || result.Items[0].Lecture.CourseName != "컴퓨터그래픽스" {
		t.Fatalf("Items[0] = %+v", result.Items[0])
	}
}

func TestTranscriptBridgeCandidatesPreferBuiltBinary(t *testing.T) {
	got := transcriptBridgeCandidates("bridges/macos")
	want := []string{
		filepath.Join("bridges", "macos", ".build", "release", "TranscriptBridge"),
		filepath.Join("bridges", "macos", ".build", "debug", "TranscriptBridge"),
		filepath.Join("bridges", "macos", "transcribe.swift"),
	}
	if len(got) != len(want) {
		t.Fatalf("transcriptBridgeCandidates() length = %d, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("transcriptBridgeCandidates()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestLectureTranscriptContextIncludesCourseAndCodeSwitching(t *testing.T) {
	got := lectureTranscriptContext(LectureRow{
		CourseName: "컴퓨터그래픽스",
		Lecture: klas.Lecture{
			ModuleTitle: "14주차",
			Title:       "렌더링 파이프라인",
		},
	})
	for _, want := range []string{"광운대학교", "컴퓨터그래픽스", "14주차", "렌더링 파이프라인", "code switching"} {
		if !containsString(got, want) {
			t.Fatalf("lectureTranscriptContext() missing %q: %v", want, got)
		}
	}
}

func TestCleanupPartialDownloadRespectsKeepPartial(t *testing.T) {
	dir := t.TempDir()
	removePath := filepath.Join(dir, "remove.part")
	keepPath := filepath.Join(dir, "keep.part")
	if err := os.WriteFile(removePath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(remove) error = %v", err)
	}
	if err := os.WriteFile(keepPath, []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(keep) error = %v", err)
	}

	cleanupPartialDownload(removePath, false)
	if _, err := os.Stat(removePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removePath stat error = %v, want not exist", err)
	}
	cleanupPartialDownload(keepPath, true)
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("keepPath stat error = %v", err)
	}
}

func TestDownloadStatusReportsPartialFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4"), []byte("done"), 0o644); err != nil {
		t.Fatalf("WriteFile(done) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lecture.mp4.part"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("WriteFile(partial) error = %v", err)
	}

	got, err := (&Service{}).DownloadStatus(dir)
	if err != nil {
		t.Fatalf("DownloadStatus() error = %v", err)
	}
	if got.Files != 1 || got.PartialFiles != 1 || got.PartialBytes != int64(len("partial")) {
		t.Fatalf("DownloadStatus() = %+v", got)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestParseAssignmentID(t *testing.T) {
	courseIndex, ordSeq, err := ParseAssignmentID("3:7")
	if err != nil {
		t.Fatalf("ParseAssignmentID() error = %v", err)
	}
	if courseIndex != 3 || ordSeq != "7" {
		t.Fatalf("ParseAssignmentID() = %d, %q", courseIndex, ordSeq)
	}
}

func TestAssignmentResourceIDAcceptsStableAndLegacyIDs(t *testing.T) {
	ref := CourseRef{TermValue: "2026,1", CourseID: "course:id/01"}
	stableID, err := StableAssignmentID(ref, "7")
	if err != nil {
		t.Fatalf("StableAssignmentID() error = %v", err)
	}
	locator, ordSeq, err := parseAssignmentResourceID(stableID)
	if err != nil || !locator.Stable || locator.Ref != ref || ordSeq != "7" {
		t.Fatalf("parseAssignmentResourceID(stable) = %+v, %q, %v", locator, ordSeq, err)
	}
	locator, ordSeq, err = parseAssignmentResourceID("3:7")
	if err != nil || locator.Stable || locator.CourseIndex != 3 || ordSeq != "7" {
		t.Fatalf("parseAssignmentResourceID(legacy) = %+v, %q, %v", locator, ordSeq, err)
	}
}

func TestNormalizeCachedAssignmentRowsUsesCourseNameAfterReorder(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{
		{Name: "오픈소스", Value: "course-b"},
		{Name: "컴퓨터그래픽스", Value: "course-a"},
	}}
	rows, migrated, err := normalizeCachedAssignmentRows([]AssignmentRow{{
		ID:         "1:7",
		TermValue:  term.Value,
		CourseName: "컴퓨터그래픽스",
	}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedAssignmentRows() error = %v", err)
	}
	if !migrated || len(rows) != 1 || rows[0].LegacyID != "1:7" {
		t.Fatalf("normalizeCachedAssignmentRows() = %+v, migrated %v", rows, migrated)
	}
	ref, _, stable, err := parseStableCourseResourceID("assignment", rows[0].ID, 1)
	if err != nil || !stable || ref.CourseID != "course-a" {
		t.Fatalf("migrated ID = %q, ref %+v, stable %v, error %v", rows[0].ID, ref, stable, err)
	}
	persisted, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(persisted, []byte("1:7")) || bytes.Contains(persisted, []byte("LegacyID")) {
		t.Fatalf("persisted cache contains legacy ID: %s", persisted)
	}
}

func TestNormalizeCachedStableAssignmentHydratesCurrentAlias(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{
		{Name: "오픈소스", Value: "course-b"},
		{Name: "컴퓨터그래픽스", Value: "course-a"},
	}}
	id, err := StableAssignmentID(CourseRef{TermValue: term.Value, CourseID: "course-a"}, "7")
	if err != nil {
		t.Fatalf("StableAssignmentID() error = %v", err)
	}
	rows, migrated, err := normalizeCachedAssignmentRows([]AssignmentRow{{ID: id}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedAssignmentRows() error = %v", err)
	}
	if migrated || rows[0].LegacyID != "2:7" {
		t.Fatalf("normalizeCachedAssignmentRows() = %+v, migrated %v", rows, migrated)
	}
}

func TestParseNoticeID(t *testing.T) {
	courseIndex, boardNo, masterNo, err := ParseNoticeID("7:1161280:1000000")
	if err != nil {
		t.Fatalf("ParseNoticeID() error = %v", err)
	}
	if courseIndex != 7 || boardNo != "1161280" || masterNo != "1000000" {
		t.Fatalf("ParseNoticeID() = %d, %q, %q", courseIndex, boardNo, masterNo)
	}
}

func TestParseLectureID(t *testing.T) {
	courseIndex, contentID, err := ParseLectureID("7:content-123")
	if err != nil {
		t.Fatalf("ParseLectureID() error = %v", err)
	}
	if courseIndex != 7 || contentID != "content-123" {
		t.Fatalf("ParseLectureID() = %d, %q", courseIndex, contentID)
	}
}

func TestLectureRowIDMatchesLearningSeqFallback(t *testing.T) {
	lecture := klas.Lecture{LearningSeq: "39769"}
	if got := lectureRowID(1, lecture); got != "1:lrn-39769" {
		t.Fatalf("lectureRowID() = %q", got)
	}
	lecture.ContentID = "content-123"
	if got := lectureRowID(1, lecture); got != "1:content-123" {
		t.Fatalf("lectureRowID() content = %q", got)
	}
}

func TestLectureFilenameSanitizesPathComponents(t *testing.T) {
	got := lectureVideoPath("root", "오픈소스/실습", klas.Lecture{
		ModuleTitle: "1주차: 소개",
		Title:       "Git? GitHub* 시작",
	}, "https://media.example.com/video.mp4?token=1", 1)
	want := filepath.Join("root", "오픈소스_실습", "video", "1-1. Git_ GitHub_ 시작.mp4")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureFilenameUsesKlasWeekSequence(t *testing.T) {
	got := lectureVideoPath("root", "컴퓨터그래픽스", klas.Lecture{
		WeekNo:      "14",
		WeeklySeq:   "2",
		ModuleTitle: "보강",
		Title:       "기말/정리",
	}, "https://media.example.com/final.mov", 2)
	want := filepath.Join("root", "컴퓨터그래픽스", "video", "14-2. 기말_정리.mov")
	if got != want {
		t.Fatalf("lectureVideoPath() = %q, want %q", got, want)
	}
}

func TestLectureWeekOrderFallsBackToCourseOrder(t *testing.T) {
	rows := []LectureRow{
		{ID: "1:a", Lecture: klas.Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "첫번째"}},
		{ID: "1:b", Lecture: klas.Lecture{ContentID: "b", ModuleTitle: "1주차", Title: "두번째"}},
		{ID: "1:c", Lecture: klas.Lecture{ContentID: "c", ModuleTitle: "2주차", Title: "첫번째"}},
	}
	orders := lectureRowWeekOrders(rows)
	if orders["1:a"] != 1 || orders["1:b"] != 2 || orders["1:c"] != 1 {
		t.Fatalf("lectureRowWeekOrders() = %#v", orders)
	}
}

func TestAcademicEventDueAt(t *testing.T) {
	got, ok := academicEventDueAt(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.17(수)",
		Title: "기말고사",
	})
	if !ok {
		t.Fatal("academicEventDueAt() expected ok")
	}
	if got.Year() != 2026 || got.Month() != time.June || got.Day() != 17 {
		t.Fatalf("academicEventDueAt() = %s", got)
	}
}

func TestAcademicEventRangeParsesMultiDayEvent(t *testing.T) {
	startAt, endAt, ok := AcademicEventRange(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.22(월) ~ 06.26(금)",
		Title: "보강주간",
	})
	if !ok {
		t.Fatal("AcademicEventRange() expected ok")
	}
	if startAt.Year() != 2026 || startAt.Month() != time.June || startAt.Day() != 22 {
		t.Fatalf("startAt = %s", startAt)
	}
	if endAt.Year() != 2026 || endAt.Month() != time.June || endAt.Day() != 27 {
		t.Fatalf("endAt = %s", endAt)
	}
}

func TestTimetablePeriodRangeUsesKwangwoonSlots(t *testing.T) {
	start, end, ok := timetablePeriodRange(1, 1)
	if !ok || start.Hour() != 9 || start.Minute() != 0 || end.Hour() != 10 || end.Minute() != 15 {
		t.Fatalf("period 1 = %02d:%02d-%02d:%02d ok=%t", start.Hour(), start.Minute(), end.Hour(), end.Minute(), ok)
	}
	start, end, ok = timetablePeriodRange(8, 1)
	if !ok || start.Hour() != 18 || start.Minute() != 50 || end.Hour() != 19 || end.Minute() != 35 {
		t.Fatalf("period 8 = %02d:%02d-%02d:%02d ok=%t", start.Hour(), start.Minute(), end.Hour(), end.Minute(), ok)
	}
	start, end, ok = timetablePeriodRange(5, 4)
	if !ok || start.Hour() != 15 || start.Minute() != 0 || end.Hour() != 18 || end.Minute() != 50 {
		t.Fatalf("period 5 span 4 = %02d:%02d-%02d:%02d ok=%t", start.Hour(), start.Minute(), end.Hour(), end.Minute(), ok)
	}
}

func TestParseConfigBool(t *testing.T) {
	got, err := parseConfigBool("yes")
	if err != nil {
		t.Fatalf("parseConfigBool() error = %v", err)
	}
	if !got {
		t.Fatal("parseConfigBool() expected true")
	}
	got, err = parseConfigBool("off")
	if err != nil {
		t.Fatalf("parseConfigBool() off error = %v", err)
	}
	if got {
		t.Fatal("parseConfigBool() expected false")
	}
	if _, err := parseConfigBool("maybe"); err == nil {
		t.Fatal("parseConfigBool() expected error")
	}
}

func TestLectureNeedsAttendance(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	if !lectureNeedsAttendance(klas.Lecture{ContentID: "content", Progress: "20", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected true")
	}
	if lectureNeedsAttendance(klas.Lecture{ContentID: "content", Progress: "100", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false for completed lecture")
	}
	if lectureNeedsAttendance(klas.Lecture{ContentID: "", Progress: "20", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false without content id")
	}
	if !lectureNeedsAttendance(klas.Lecture{LearningSeq: "15", AchievedTime: "0", RequiredTime: "10", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected true for incomplete learning activity")
	}
	if lectureNeedsAttendance(klas.Lecture{LearningSeq: "15", AchievedTime: "10", RequiredTime: "10", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false for completed learning activity")
	}
}

func TestDashboardAssignmentsKeepsUpcomingUnsubmitted(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	rows := []AssignmentRow{
		{ID: "1:past", Assignment: klas.Assignment{Title: "지난 과제", DueAt: &past}},
		{ID: "1:done", Assignment: klas.Assignment{Title: "제출 과제", DueAt: &future, Submitted: true}},
		{ID: "1:todo", Assignment: klas.Assignment{Title: "할 과제", DueAt: &future}},
	}

	got := dashboardAssignments(rows, 5)
	if len(got) != 1 || got[0].ID != "1:todo" {
		t.Fatalf("dashboardAssignments() = %+v", got)
	}
}

func TestDashboardAttendanceCountsMarks(t *testing.T) {
	got := dashboardAttendance([]AttendanceRow{
		{
			Sessions: []klas.AttendanceSession{
				{Slots: []klas.AttendanceSlot{
					{Mark: "O"},
					{Mark: "X"},
					{Mark: "L"},
					{Mark: "R"},
					{Mark: "A"},
					{Mark: "??"},
				}},
			},
		},
		{Err: errDashboardTest},
	})

	if got.TotalCourses != 2 || got.Completed != 1 || got.Absent != 1 || got.Late != 1 || got.LeaveEarly != 1 || got.Excused != 1 || got.Unknown != 1 || got.DetailErrors != 1 {
		t.Fatalf("dashboardAttendance() = %+v", got)
	}
	if len(got.Rows) != 2 || got.Rows[0].Completed != 1 || got.Rows[0].Unknown != 1 || got.Rows[1].Err == nil {
		t.Fatalf("dashboardAttendance() rows = %+v", got.Rows)
	}
}

func TestDashboardNoticesSortsByRecentDate(t *testing.T) {
	oldDate := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	newDate := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	rows := []NoticeRow{
		{ID: "old", Notice: klas.Notice{Registered: &oldDate, Top: true}},
		{ID: "new", Notice: klas.Notice{Registered: &newDate}},
	}

	got := dashboardNotices(rows, 2)
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("dashboardNotices() = %+v", got)
	}
}

func TestBuildReminderNotesIncludesBodyAndMarker(t *testing.T) {
	dueAt := time.Date(2026, 5, 5, 23, 59, 0, 0, time.FixedZone("KST", 9*60*60))
	notes := buildReminderNotes(AssignmentDetailResult{
		ID:         "3:1",
		TermValue:  "2026,1",
		CourseName: "컴퓨터그래픽스",
		Detail: klas.AssignmentDetail{
			Title:          "과제1",
			ContentText:    "과제 skeleton code 구성 가이드",
			DueAt:          &dueAt,
			Submitted:      true,
			ReportType:     "개인",
			SubmitFileType: "zip",
			FileLimitMB:    "10",
		},
	})

	for _, want := range []string{
		"과제 skeleton code 구성 가이드",
		"--- KLAP ---",
		"ID: 3:1",
		"과목: 컴퓨터그래픽스",
		"제목: 과제1",
		"마감: 2026-05-05 23:59",
		"상태: 제출",
		"#2026-1 #컴퓨터그래픽스",
		"[This reminder is created by KLAP.]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes does not contain %q:\n%s", want, notes)
		}
	}
}

func TestReminderHashtags(t *testing.T) {
	got := reminderHashtags("2026,1", "오픈소스 소프트웨어 실습")
	want := "#2026-1 #오픈소스소프트웨어실습"
	if got != want {
		t.Fatalf("reminderHashtags() = %q, want %q", got, want)
	}
}

func TestBuildLectureReminderNotesIncludesMarker(t *testing.T) {
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	notes := buildLectureReminderNotes(LectureRow{
		ID:         "7:lecture",
		TermValue:  "2026,1",
		CourseName: "오픈소스소프트웨어실습",
		Lecture: klas.Lecture{
			Title:        "HuggingFace",
			ModuleTitle:  "14주차",
			EndAt:        &dueAt,
			AchievedTime: "0",
			RequiredTime: "10",
		},
	})

	for _, want := range []string{
		"ID: lecture:7:lecture",
		"과목: 오픈소스소프트웨어실습",
		"제목: HuggingFace",
		"#2026-1 #오픈소스소프트웨어실습",
		"[This reminder is created by KLAP.]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("buildLectureReminderNotes() missing %q: %q", want, notes)
		}
	}
}

func TestBuildAcademicCalendarNotesIncludesMarker(t *testing.T) {
	event := AcademicEvent{Year: "2026", Month: "3월", Date: "3(화)", Title: "개강", Note: "비고"}
	notes := buildAcademicCalendarNotes(event)
	for _, want := range []string{
		"ID: academic:2026:개강:비고",
		"학년도: 2026",
		"비고: 비고",
		"[This calendar event is created by KLAP.]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("buildAcademicCalendarNotes() missing %q: %q", want, notes)
		}
	}
}

func TestAcademicEventIDStableWhenDateChanges(t *testing.T) {
	before := AcademicEvent{Year: "2026", Month: "3월", Date: "3(화)", Title: "개강", Note: "비고"}
	after := AcademicEvent{Year: "2026", Month: "3월", Date: "2(월)", Title: "개강", Note: "비고"}
	if academicEventID(before) != academicEventID(after) {
		t.Fatalf("academicEventID should ignore date changes: %q != %q", academicEventID(before), academicEventID(after))
	}
}

func TestTermHashtagSeasonSemesters(t *testing.T) {
	if got := termHashtag("2026,3"); got != "#2026-여름학기" {
		t.Fatalf("termHashtag() summer = %q", got)
	}
	if got := termHashtag("2026,4"); got != "#2026-겨울학기" {
		t.Fatalf("termHashtag() winter = %q", got)
	}
}

func TestParseAcademicEvents(t *testing.T) {
	body := []byte(`
		<!--
		<h3>2025 학사일정</h3>
		<table><tbody><tr><td>3월</td><td>1(토)</td><td>오래된 일정</td><td></td></tr></tbody></table>
		-->
		<h3>2026 학사일정</h3>
		<table><tbody>
			<tr><td rowspan="2">3월</td><td>3(화)</td><td>2026학년도 1학기 개강(학기개시일)</td><td>3월2일(월) - 대체공휴일</td></tr>
			<tr><td>28(토)</td><td>수업일수 4분의 1</td><td>15주 기준</td></tr>
			<tr><td>8월</td><td>3(월)~31(월)</td><td>2학기 복학신청</td></tr>
		</tbody></table>
		<h3>2027 학사일정</h3>
		<table><tbody>
			<tr><td>3월</td><td>2(화)</td><td>2027학년도 1학기 개강</td><td></td></tr>
		</tbody></table>
	`)

	events, err := parseAcademicEvents(body, "2026")
	if err != nil {
		t.Fatalf("parseAcademicEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("parseAcademicEvents() len = %d, want 3: %+v", len(events), events)
	}
	if events[0].Month != "3월" || events[0].Date != "3(화)" || events[0].Title != "2026학년도 1학기 개강(학기개시일)" {
		t.Fatalf("parseAcademicEvents()[0] = %+v", events[0])
	}
	if events[1].Month != "3월" || events[1].Date != "28(토)" || events[1].Note != "15주 기준" {
		t.Fatalf("parseAcademicEvents()[1] = %+v", events[1])
	}
	if events[2].Month != "8월" || events[2].Date != "3(월)~31(월)" || events[2].Title != "2학기 복학신청" {
		t.Fatalf("parseAcademicEvents()[2] = %+v", events[2])
	}
}
