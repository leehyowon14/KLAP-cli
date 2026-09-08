package app

import (
	"bytes"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	klapcalendar "github.com/leehyowon14/KLAP-cli/internal/calendar"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/reminder"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeSyncStateStore struct {
	load func(string, string) (syncstate.State, bool, error)
	save func(string, string, syncstate.State) error
}

func (s *fakeSyncStateStore) Load(scope string, owner string) (syncstate.State, bool, error) {
	return s.load(scope, owner)
}

func (s *fakeSyncStateStore) Save(scope string, owner string, state syncstate.State) error {
	return s.save(scope, owner, state)
}

func newSyncStateTestService(t *testing.T) (*Service, *cache.Store, *syncstate.Store) {
	t.Helper()
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("cache NewStoreAt() error = %v", err)
	}
	syncStateStore, err := syncstate.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("syncstate NewStoreAt() error = %v", err)
	}
	return &Service{cacheStore: cacheStore, syncStateStore: syncStateStore}, cacheStore, syncStateStore
}

func TestPrepareReminderSyncPromptsOnceForChangedSource(t *testing.T) {
	service, _, _ := newSyncStateTestService(t)
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
	service, cacheStore, syncStateStore := newSyncStateTestService(t)
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
	if err := cacheStore.Set(syncSourceCacheKey("assignment", "20260001"), time.Hour, state); err != nil {
		t.Fatalf("legacy cache Set() error = %v", err)
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
	if err := service.saveSyncSourceState("assignment", "20260001", prepared.State); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}
	migrated, ok, err := syncStateStore.Load("assignment", "20260001")
	if err != nil || !ok {
		t.Fatalf("syncStateStore.Load() = %+v, %v, %v", migrated, ok, err)
	}
	if _, exists := migrated.Items[legacy.ID]; exists {
		t.Fatalf("legacy state persisted in dedicated store: %+v", migrated.Items)
	}
	if migrated.Items[stable.ID].Hash != reminderSourceHash(stable) {
		t.Fatalf("dedicated stable state = %+v", migrated.Items)
	}
	var removedLegacy syncSourceState
	if _, ok, err := cacheStore.Get(syncSourceCacheKey("assignment", "20260001"), &removedLegacy); err != nil || ok {
		t.Fatalf("legacy cache after successful migration = %+v, %v, %v", removedLegacy, ok, err)
	}

	legacyChanged := syncSourceState{Items: map[string]syncSourceItem{legacy.ID: {Hash: "changed-after-migration"}}}
	if err := cacheStore.Set(syncSourceCacheKey("assignment", "20260001"), time.Hour, legacyChanged); err != nil {
		t.Fatalf("legacy cache update error = %v", err)
	}
	reloaded, err := service.loadSyncSourceState("assignment", "20260001")
	if err != nil || reloaded.Items[stable.ID].Hash != reminderSourceHash(stable) {
		t.Fatalf("idempotent migration reload = %+v, %v", reloaded, err)
	}
}

func TestReplaceReminderNoteIDPreservesBodyMetadataLookalike(t *testing.T) {
	notes := "본문\n--- KLAP ---\nID: body-id\n과목: body-course\n\n--- KLAP ---\n\nID: old-id\n과목: 실제과목\n[This reminder is created by KLAP.]"
	got := replaceReminderNoteID(notes, "stable-id")
	if !strings.Contains(got, "ID: body-id") || !strings.Contains(got, "과목: body-course") {
		t.Fatalf("replaceReminderNoteID() changed body:\n%s", got)
	}
	if !strings.Contains(got, "--- KLAP ---\n\nID: stable-id\n과목: 실제과목") {
		t.Fatalf("replaceReminderNoteID() did not change metadata:\n%s", got)
	}
}

func TestReplaceReminderNoteIDSupportsLegacySeparator(t *testing.T) {
	notes := "본문\nID: body-id\n=========================================\n\nID: old-id\n과목: 실제과목\n[This reminder is created by KLAP.]"
	got := replaceReminderNoteID(notes, "stable-id")
	if !strings.Contains(got, "ID: body-id") {
		t.Fatalf("replaceReminderNoteID() changed body:\n%s", got)
	}
	if !strings.Contains(got, "=========================================\n\nID: stable-id\n과목: 실제과목") {
		t.Fatalf("replaceReminderNoteID() did not change legacy metadata:\n%s", got)
	}
}

func TestPrepareReminderSyncConflictsOnMismatchedLegacyBaseline(t *testing.T) {
	service, _, _ := newSyncStateTestService(t)
	state := syncSourceState{Items: map[string]syncSourceItem{
		"lecture:2:content-123": {Hash: "another-course-hash"},
	}}
	if err := service.saveSyncSourceState("lecture", "20260001", state); err != nil {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}
	assignment := reminder.Assignment{
		ID:        "lecture:v1:stable",
		LegacyIDs: []string{"lecture:2:content-123"},
		Title:     "온라인 강의",
		Course:    "컴퓨터그래픽스",
		Notes:     "ID: lecture:v1:stable\n과목: 컴퓨터그래픽스",
	}
	prepared, err := service.prepareReminderSync("lecture", "20260001", []reminder.Assignment{assignment}, nil)
	if err != nil {
		t.Fatalf("prepareReminderSync() error = %v", err)
	}
	if len(prepared.Assignments) != 0 || len(prepared.Conflicts) != 1 {
		t.Fatalf("prepareReminderSync() = %+v", prepared)
	}
	if got := prepared.State.Items[assignment.ID].Hash; got != "another-course-hash" {
		t.Fatalf("stable conflict baseline = %q", got)
	}
	if got := prepared.State.Items["lecture:2:content-123"].Hash; got != "another-course-hash" {
		t.Fatalf("legacy baseline changed: %+v", prepared.State.Items)
	}
}

func TestPrepareCalendarSyncWithoutBaselineDoesNotForceUpdate(t *testing.T) {
	service, _, _ := newSyncStateTestService(t)
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
	service, cacheStore, _ := newSyncStateTestService(t)
	if err := cacheStore.Set(syncSourceCacheKey("assignment", "20260001"), time.Minute, "invalid state"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	assignment := reminder.Assignment{ID: "assignment:1", Title: "기말 과제"}

	prepared, err := service.prepareReminderSync("assignment", "20260001", []reminder.Assignment{assignment}, nil)
	if err == nil {
		t.Fatal("prepareReminderSync() expected corrupted baseline error")
	}
	if len(prepared.Assignments) != 0 || len(prepared.Pending) != 0 {
		t.Fatalf("prepareReminderSync() returned work after baseline error: %+v", prepared)
	}
}

func TestSyncStateMigrationFailurePreservesLegacyCache(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("cache NewStoreAt() error = %v", err)
	}
	syncStateDir := t.TempDir()
	syncStateStore, err := syncstate.NewStoreAt(syncStateDir)
	if err != nil {
		t.Fatalf("syncstate NewStoreAt() error = %v", err)
	}
	legacy := syncSourceState{Items: map[string]syncSourceItem{"assignment:1": {Hash: "source-hash"}}}
	legacyKey := syncSourceCacheKey("assignment", "20260001")
	if err := cacheStore.Set(legacyKey, time.Hour, legacy); err != nil {
		t.Fatalf("legacy cache Set() error = %v", err)
	}
	corrupt := []byte(`{"version":1,"sources":`)
	if err := os.WriteFile(filepath.Join(syncStateDir, "sync-state.json"), corrupt, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	service := &Service{cacheStore: cacheStore, syncStateStore: syncStateStore}
	if _, err := service.loadSyncSourceState("assignment", "20260001"); err == nil {
		t.Fatal("loadSyncSourceState() expected dedicated store parse error")
	}
	var preserved syncSourceState
	if _, ok, err := cacheStore.Get(legacyKey, &preserved); err != nil || !ok {
		t.Fatalf("legacy cache Get() = %+v, %v, %v", preserved, ok, err)
	}
	if preserved.Items["assignment:1"].Hash != "source-hash" {
		t.Fatalf("legacy cache changed after migration failure: %+v", preserved)
	}
	body, err := os.ReadFile(filepath.Join(syncStateDir, "sync-state.json"))
	if err != nil || !bytes.Equal(body, corrupt) {
		t.Fatalf("corrupt dedicated state overwritten: %q, %v", body, err)
	}
}

func TestSyncStateSaveFailureDoesNotDeleteLegacyCache(t *testing.T) {
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("cache NewStoreAt() error = %v", err)
	}
	legacyKey := syncSourceCacheKey("assignment", "20260001")
	legacy := syncSourceState{Items: map[string]syncSourceItem{"assignment:1": {Hash: "source-hash"}}}
	if err := cacheStore.Set(legacyKey, time.Hour, legacy); err != nil {
		t.Fatalf("legacy cache Set() error = %v", err)
	}
	wantErr := errors.New("durable save failed")
	service := &Service{
		cacheStore: cacheStore,
		syncStateStore: &fakeSyncStateStore{
			load: func(string, string) (syncstate.State, bool, error) {
				return syncstate.State{}, false, nil
			},
			save: func(string, string, syncstate.State) error {
				return wantErr
			},
		},
	}
	if err := service.saveSyncSourceState("assignment", "20260001", legacy); !errors.Is(err, wantErr) {
		t.Fatalf("saveSyncSourceState() error = %v", err)
	}
	var preserved syncSourceState
	if _, ok, err := cacheStore.Get(legacyKey, &preserved); err != nil || !ok {
		t.Fatalf("legacy cache Get() = %+v, %v, %v", preserved, ok, err)
	}
	if preserved.Items["assignment:1"].Hash != "source-hash" {
		t.Fatalf("legacy cache changed after save failure: %+v", preserved)
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
	service, _, _ := newSyncStateTestService(t)
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
	service, cacheStore, _ := newSyncStateTestService(t)
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
	service, cacheStore, _ := newSyncStateTestService(t)
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
	service, _, _ := newSyncStateTestService(t)
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

func TestLectureReminderIDDoesNotDuplicateStablePrefix(t *testing.T) {
	if got := lectureReminderID("lecture:v1:stable"); got != "lecture:v1:stable" {
		t.Fatalf("lectureReminderID(stable) = %q", got)
	}
	if got := lectureReminderID("7:content-123"); got != "lecture:7:content-123" {
		t.Fatalf("lectureReminderID(legacy) = %q", got)
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
