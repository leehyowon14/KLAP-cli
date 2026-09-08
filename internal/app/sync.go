package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	klapcalendar "github.com/leehyowon14/KLAP-cli/internal/calendar"
	"github.com/leehyowon14/KLAP-cli/internal/reminder"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"strconv"
	"strings"
	"time"
)

type syncStateStore interface {
	Load(scope string, owner string) (syncstate.State, bool, error)
	Save(scope string, owner string, state syncstate.State) error
}

type ReminderSyncResult struct {
	Result        reminder.SyncResult
	EligibleCount int
	Conflicts     []SyncConflict
}

type CalendarSyncResult struct {
	Result        klapcalendar.SyncResult
	EligibleCount int
	Conflicts     []SyncConflict
}

type SyncDecision string

const (
	SyncDecisionApply SyncDecision = "apply"
	SyncDecisionKeep  SyncDecision = "keep"
)

type SyncConflict struct {
	Key          string
	Scope        string
	ID           string
	Kind         string
	Title        string
	PreviousHash string
	CurrentHash  string
	Summary      string
}

type SyncConflictError struct {
	Conflicts []SyncConflict
}

func (e SyncConflictError) Error() string {
	return fmt.Sprintf("KLAS에서 갱신된 항목 %d개에 대한 확인이 필요합니다", len(e.Conflicts))
}

func (s *Service) SyncAssignmentReminders(ctx context.Context, opts AssignmentSyncOptions) (ReminderSyncResult, error) {
	rows, err := s.AssignmentList(ctx, opts.Query)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.Query.User)
	if err != nil {
		return ReminderSyncResult{}, err
	}

	currentSettings, err := s.loadSettings()
	if err != nil {
		return ReminderSyncResult{}, err
	}

	now := time.Now()
	assignments := make([]reminder.Assignment, 0, len(rows))
	for _, row := range rows {
		if !futureTime(row.Assignment.DueAt, now) {
			continue
		}
		detail, detailErr := s.AssignmentDetail(ctx, row.ID, opts.Query.User)
		if detailErr != nil {
			return ReminderSyncResult{}, detailErr
		}
		assignments = append(assignments, reminder.Assignment{
			ID:        detail.ID,
			LegacyIDs: compactNonEmpty([]string{row.LegacyID}),
			TermValue: detail.TermValue,
			Title:     detail.Detail.Title,
			Course:    detail.CourseName,
			DueAt:     detail.Detail.DueAt,
			Submitted: detail.Detail.Submitted,
			DetailURL: row.DetailURL,
			Notes:     buildReminderNotes(detail),
		})
	}

	if len(assignments) == 0 {
		return ReminderSyncResult{}, nil
	}
	prepared, err := s.prepareReminderSync("assignment", studentID, assignments, opts.Decisions)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return ReminderSyncResult{EligibleCount: len(assignments), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Assignments) == 0 {
		if err := s.saveSyncSourceState("assignment", studentID, prepared.State); err != nil {
			return ReminderSyncResult{}, err
		}
		return ReminderSyncResult{Result: reminder.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(assignments)}, nil
	}

	result, err := s.reminderSyncer.Sync(ctx, reminder.SyncRequest{
		ListName:        currentSettings.Reminder.ListName,
		UseExistingList: currentSettings.Reminder.UseExistingList,
		AlarmBeforeMin:  currentSettings.Reminder.AlarmBeforeMin,
		Assignments:     prepared.Assignments,
	})
	if err != nil {
		return ReminderSyncResult{}, err
	}
	result.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, result.SyncedIDs)
	if err := s.saveSyncSourceState("assignment", studentID, prepared.State); err != nil {
		return ReminderSyncResult{}, err
	}
	return ReminderSyncResult{
		Result:        result,
		EligibleCount: len(assignments),
	}, nil
}

func (s *Service) SyncLectureReminders(ctx context.Context, opts LectureSyncOptions) (ReminderSyncResult, error) {
	rows, err := s.LectureList(ctx, opts.Query)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.Query.User)
	if err != nil {
		return ReminderSyncResult{}, err
	}

	currentSettings, err := s.loadSettings()
	if err != nil {
		return ReminderSyncResult{}, err
	}

	now := time.Now()
	assignments := make([]reminder.Assignment, 0, len(rows))
	for _, row := range rows {
		if !futureTime(row.Lecture.EndAt, now) {
			continue
		}
		assignments = append(assignments, reminder.Assignment{
			ID:        lectureReminderID(row.ID),
			LegacyIDs: compactNonEmpty([]string{lectureReminderID(row.LegacyID)}),
			TermValue: row.TermValue,
			Title:     firstNonEmpty(row.Lecture.Title, row.Lecture.ModuleTitle, "온라인 강의"),
			Course:    row.CourseName,
			DueAt:     row.Lecture.EndAt,
			Submitted: !lectureNeedsAttendance(row.Lecture, now),
			DetailURL: row.Lecture.PlayURL,
			Notes:     buildLectureReminderNotes(row),
		})
	}

	if len(assignments) == 0 {
		return ReminderSyncResult{}, nil
	}
	prepared, err := s.prepareReminderSync("lecture", studentID, assignments, opts.Decisions)
	if err != nil {
		return ReminderSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return ReminderSyncResult{EligibleCount: len(assignments), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Assignments) == 0 {
		if err := s.saveSyncSourceState("lecture", studentID, prepared.State); err != nil {
			return ReminderSyncResult{}, err
		}
		return ReminderSyncResult{Result: reminder.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(assignments)}, nil
	}

	result, err := s.reminderSyncer.Sync(ctx, reminder.SyncRequest{
		ListName:        currentSettings.Reminder.ListName,
		UseExistingList: currentSettings.Reminder.UseExistingList,
		AlarmBeforeMin:  currentSettings.Reminder.AlarmBeforeMin,
		Assignments:     prepared.Assignments,
	})
	if err != nil {
		return ReminderSyncResult{}, err
	}
	result.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, result.SyncedIDs)
	if err := s.saveSyncSourceState("lecture", studentID, prepared.State); err != nil {
		return ReminderSyncResult{}, err
	}
	return ReminderSyncResult{
		Result:        result,
		EligibleCount: len(assignments),
	}, nil
}

func (s *Service) SyncAcademicCalendar(ctx context.Context, opts AcademicSyncOptions) (CalendarSyncResult, error) {
	result, err := s.AcademicList(ctx, opts.Query)
	if err != nil {
		return CalendarSyncResult{}, err
	}

	currentSettings, err := s.loadSettings()
	if err != nil {
		return CalendarSyncResult{}, err
	}

	events := make([]klapcalendar.Event, 0, len(result.Events))
	now := time.Now()
	for _, academicEvent := range result.Events {
		startAt, endAt, ok := academicEventRange(academicEvent)
		if !ok || startAt.Before(now) {
			continue
		}
		events = append(events, klapcalendar.Event{
			ID:      academicEventID(academicEvent),
			Title:   academicEvent.Title,
			StartAt: startAt,
			EndAt:   endAt,
			AllDay:  true,
			Notes:   buildAcademicCalendarNotes(academicEvent),
			URL:     result.SourceURL,
		})
	}
	if len(events) == 0 {
		return CalendarSyncResult{}, nil
	}
	prepared, err := s.prepareCalendarSync("academic", "global", events, opts.Decisions)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return CalendarSyncResult{EligibleCount: len(events), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Events) == 0 {
		if err := s.saveSyncSourceState("academic", "global", prepared.State); err != nil {
			return CalendarSyncResult{}, err
		}
		return CalendarSyncResult{Result: klapcalendar.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(events)}, nil
	}

	syncResult, err := s.calendarSyncer.Sync(klapcalendar.SyncRequest{
		CalendarName:    currentSettings.Calendar.Name,
		UseExistingList: currentSettings.Calendar.UseExistingList,
		Events:          prepared.Events,
	})
	if err != nil {
		return CalendarSyncResult{}, err
	}
	syncResult.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, syncResult.SyncedIDs)
	if err := s.saveSyncSourceState("academic", "global", prepared.State); err != nil {
		return CalendarSyncResult{}, err
	}
	return CalendarSyncResult{Result: syncResult, EligibleCount: len(events)}, nil
}

func (s *Service) SyncTimetableCalendar(ctx context.Context, opts TimetableSyncOptions) (CalendarSyncResult, error) {
	currentSettings, err := s.loadSettings()
	if err != nil {
		return CalendarSyncResult{}, err
	}
	result, err := s.Timetable(ctx, opts.Query)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	studentID, err := s.selectedStudentID(ctx, opts.Query.User)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	startAt, endAt := timetableTermRange(result.Term.Value)
	if academic, academicErr := s.AcademicList(ctx, AcademicListOptions{Year: timetableTermYear(result.Term.Value), Refresh: opts.Query.Refresh}); academicErr == nil {
		startAt, endAt = timetableTermRangeFromAcademic(result.Term.Value, academic.Events, startAt, endAt)
	}

	events := make([]klapcalendar.Event, 0, len(result.Entries))
	for _, entry := range result.Entries {
		if entry.Online || strings.TrimSpace(entry.Room) == "" {
			continue
		}
		startClock, endClock, ok := timetablePeriodRange(entry.Period, entry.Span)
		if !ok {
			continue
		}
		firstDay := firstWeekdayOnOrAfter(startAt, entry.Weekday)
		eventStart := time.Date(firstDay.Year(), firstDay.Month(), firstDay.Day(), startClock.Hour(), startClock.Minute(), 0, 0, time.Local)
		eventEnd := time.Date(firstDay.Year(), firstDay.Month(), firstDay.Day(), endClock.Hour(), endClock.Minute(), 0, 0, time.Local)
		if !eventEnd.After(eventStart) {
			continue
		}
		events = append(events, klapcalendar.Event{
			ID:            timetableCalendarEventID(result.Term.Value, entry),
			Title:         entry.SubjectName,
			StartAt:       eventStart,
			EndAt:         eventEnd,
			AllDay:        false,
			Notes:         buildTimetableCalendarNotes(result.Term.Value, entry),
			Recurrence:    "weekly",
			RecurrenceEnd: &endAt,
		})
	}
	if len(events) == 0 {
		return CalendarSyncResult{}, nil
	}
	prepared, err := s.prepareCalendarSync("timetable", studentID, events, opts.Decisions)
	if err != nil {
		return CalendarSyncResult{}, err
	}
	if len(prepared.Conflicts) > 0 {
		return CalendarSyncResult{EligibleCount: len(events), Conflicts: prepared.Conflicts}, SyncConflictError{Conflicts: prepared.Conflicts}
	}
	if len(prepared.Events) == 0 {
		if err := s.saveSyncSourceState("timetable", studentID, prepared.State); err != nil {
			return CalendarSyncResult{}, err
		}
		return CalendarSyncResult{Result: klapcalendar.SyncResult{Skipped: prepared.Skipped}, EligibleCount: len(events)}, nil
	}

	syncResult, err := s.calendarSyncer.Sync(klapcalendar.SyncRequest{
		CalendarName:    currentSettings.Calendar.TimetableName,
		UseExistingList: currentSettings.Calendar.TimetableUseExistingList,
		Events:          prepared.Events,
	})
	if err != nil {
		return CalendarSyncResult{}, err
	}
	syncResult.Skipped += prepared.Skipped
	commitSyncSourceHashes(&prepared.State, prepared.Pending, syncResult.SyncedIDs)
	if err := s.saveSyncSourceState("timetable", studentID, prepared.State); err != nil {
		return CalendarSyncResult{}, err
	}
	return CalendarSyncResult{Result: syncResult, EligibleCount: len(events)}, nil
}

type syncSourceState = syncstate.State

type syncSourceItem = syncstate.Item

type preparedReminderSync struct {
	Assignments []reminder.Assignment
	State       syncSourceState
	Pending     map[string]string
	Skipped     int
	Conflicts   []SyncConflict
}

type preparedCalendarSync struct {
	Events    []klapcalendar.Event
	State     syncSourceState
	Pending   map[string]string
	Skipped   int
	Conflicts []SyncConflict
}

func (s *Service) prepareReminderSync(scope string, owner string, assignments []reminder.Assignment, decisions map[string]SyncDecision) (preparedReminderSync, error) {
	state, err := s.loadSyncSourceState(scope, owner)
	if err != nil {
		return preparedReminderSync{}, err
	}
	result := preparedReminderSync{
		State:   state,
		Pending: map[string]string{},
	}
	for _, assignment := range assignments {
		hash := reminderSourceHash(assignment)
		item, exists := state.Items[assignment.ID]
		if !exists {
			for _, legacyID := range assignment.LegacyIDs {
				legacyItem, legacyExists := state.Items[legacyID]
				if !legacyExists {
					continue
				}
				if legacyItem.Hash == legacyReminderSourceHash(assignment, legacyID) {
					legacyItem.Hash = hash
					legacyItem.IgnoredHash = ""
					delete(state.Items, legacyID)
				}
				state.Items[assignment.ID] = legacyItem
				item = legacyItem
				break
			}
		}
		assignment.KnownSourceHash = item.Hash
		key := syncConflictKey(scope, assignment.ID)
		if item.Hash != "" && item.Hash != hash {
			switch decisions[key] {
			case SyncDecisionApply:
				assignment.ForceUpdate = true
			case SyncDecisionKeep:
				item.IgnoredHash = hash
				state.Items[assignment.ID] = item
				result.Skipped++
				continue
			default:
				if item.IgnoredHash == hash {
					result.Skipped++
					continue
				}
				result.Conflicts = append(result.Conflicts, SyncConflict{
					Key:          key,
					Scope:        scope,
					ID:           assignment.ID,
					Kind:         "reminder",
					Title:        assignment.Title,
					PreviousHash: item.Hash,
					CurrentHash:  hash,
					Summary:      reminderConflictSummary(assignment),
				})
				continue
			}
		}
		result.Assignments = append(result.Assignments, assignment)
		result.Pending[assignment.ID] = hash
	}
	result.State = state
	return result, nil
}

func (s *Service) prepareCalendarSync(scope string, owner string, events []klapcalendar.Event, decisions map[string]SyncDecision) (preparedCalendarSync, error) {
	state, err := s.loadSyncSourceState(scope, owner)
	if err != nil {
		return preparedCalendarSync{}, err
	}
	result := preparedCalendarSync{
		State:   state,
		Pending: map[string]string{},
	}
	for _, event := range events {
		hash := calendarSourceHash(event)
		item := state.Items[event.ID]
		event.KnownSourceHash = item.Hash
		key := syncConflictKey(scope, event.ID)
		if item.Hash != "" && item.Hash != hash {
			switch decisions[key] {
			case SyncDecisionApply:
				event.ForceUpdate = true
			case SyncDecisionKeep:
				item.IgnoredHash = hash
				state.Items[event.ID] = item
				result.Skipped++
				continue
			default:
				if item.IgnoredHash == hash {
					result.Skipped++
					continue
				}
				result.Conflicts = append(result.Conflicts, SyncConflict{
					Key:          key,
					Scope:        scope,
					ID:           event.ID,
					Kind:         "calendar",
					Title:        event.Title,
					PreviousHash: item.Hash,
					CurrentHash:  hash,
					Summary:      calendarConflictSummary(event),
				})
				continue
			}
		}
		result.Events = append(result.Events, event)
		result.Pending[event.ID] = hash
	}
	result.State = state
	return result, nil
}

func (s *Service) loadSyncSourceState(scope string, owner string) (syncSourceState, error) {
	state := syncSourceState{Items: map[string]syncSourceItem{}}
	if s.syncStateStore == nil {
		return syncSourceState{}, errors.New("sync state store가 없습니다")
	}
	stored, ok, err := s.syncStateStore.Load(scope, owner)
	if err != nil {
		return syncSourceState{}, err
	}
	if ok {
		return stored, nil
	}
	if s.cacheStore == nil {
		return state, nil
	}
	_, ok, err = s.cacheStore.Get(syncSourceCacheKey(scope, owner), &state)
	if err != nil {
		return syncSourceState{}, fmt.Errorf("legacy sync state migration 읽기 실패: %w", err)
	}
	if !ok || state.Items == nil {
		state.Items = map[string]syncSourceItem{}
	}
	return state, nil
}

func (s *Service) saveSyncSourceState(scope string, owner string, state syncSourceState) error {
	if s.syncStateStore == nil {
		return errors.New("sync state store가 없습니다")
	}
	if state.Items == nil {
		state.Items = map[string]syncSourceItem{}
	}
	if err := s.syncStateStore.Save(scope, owner, state); err != nil {
		return err
	}
	if s.cacheStore != nil {
		if _, err := s.cacheStore.Delete(syncSourceCacheKey(scope, owner)); err != nil {
			return fmt.Errorf("migrated legacy sync state 정리 실패: %w", err)
		}
	}
	return nil
}

func syncSourceCacheKey(scope string, owner string) string {
	return "sync-source:" + strings.TrimSpace(owner) + ":" + strings.TrimSpace(scope)
}

func syncConflictKey(scope string, id string) string {
	return strings.TrimSpace(scope) + ":" + strings.TrimSpace(id)
}

func commitSyncSourceHashes(state *syncSourceState, pending map[string]string, syncedIDs []string) {
	if state.Items == nil {
		state.Items = map[string]syncSourceItem{}
	}
	for _, id := range syncedIDs {
		if hash, ok := pending[id]; ok {
			state.Items[id] = syncSourceItem{Hash: hash}
		}
	}
}

func reminderSourceHash(assignment reminder.Assignment) string {
	dueAt := ""
	if assignment.DueAt != nil {
		dueAt = assignment.DueAt.UTC().Format(time.RFC3339Nano)
	}
	return hashSyncParts(
		assignment.ID,
		assignment.Title,
		assignment.Course,
		dueAt,
		strconv.FormatBool(assignment.Submitted),
		assignment.DetailURL,
		assignment.Notes,
	)
}

func legacyReminderSourceHash(assignment reminder.Assignment, legacyID string) string {
	assignment.ID = strings.TrimSpace(legacyID)
	assignment.Notes = replaceReminderNoteID(assignment.Notes, assignment.ID)
	return reminderSourceHash(assignment)
}

func replaceReminderNoteID(notes string, id string) string {
	lines := strings.Split(notes, "\n")
	metadataStart, metadataEnd, ok := reminderMetadataBounds(lines)
	if !ok {
		return notes
	}
	for index := metadataStart; index < metadataEnd; index++ {
		line := lines[index]
		if strings.HasPrefix(line, "ID: ") {
			lines[index] = "ID: " + strings.TrimSpace(id)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func reminderMetadataBounds(lines []string) (int, int, bool) {
	footer := -1
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.TrimSpace(lines[index]) == "[This reminder is created by KLAP.]" {
			footer = index
			break
		}
	}
	if footer < 0 {
		return 0, 0, false
	}
	for index := footer - 1; index >= 0; index-- {
		marker := strings.TrimSpace(lines[index])
		if marker == "--- KLAP ---" || marker == "=========================================" {
			return index + 1, footer, true
		}
	}
	return 0, 0, false
}

func calendarSourceHash(event klapcalendar.Event) string {
	recurrenceEnd := ""
	if event.RecurrenceEnd != nil {
		recurrenceEnd = event.RecurrenceEnd.UTC().Format(time.RFC3339Nano)
	}
	return hashSyncParts(
		event.ID,
		event.Title,
		event.StartAt.UTC().Format(time.RFC3339Nano),
		event.EndAt.UTC().Format(time.RFC3339Nano),
		strconv.FormatBool(event.AllDay),
		event.Notes,
		event.URL,
		event.Recurrence,
		recurrenceEnd,
	)
}

func hashSyncParts(parts ...string) string {
	hasher := sha256.New()
	for _, part := range parts {
		hasher.Write([]byte(part))
		hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func reminderConflictSummary(assignment reminder.Assignment) string {
	if assignment.DueAt == nil {
		return assignment.Title
	}
	return fmt.Sprintf("%s · %s", assignment.Title, assignment.DueAt.Format("2006-01-02 15:04"))
}

func calendarConflictSummary(event klapcalendar.Event) string {
	if event.AllDay {
		return fmt.Sprintf("%s · %s ~ %s", event.Title, event.StartAt.Format("2006-01-02"), event.EndAt.Format("2006-01-02"))
	}
	return fmt.Sprintf("%s · %s ~ %s", event.Title, event.StartAt.Format("2006-01-02 15:04"), event.EndAt.Format("15:04"))
}

func buildReminderNotes(result AssignmentDetailResult) string {
	detail := result.Detail
	var builder strings.Builder

	if strings.TrimSpace(detail.ContentText) != "" {
		builder.WriteString(strings.TrimSpace(detail.ContentText))
		builder.WriteString("\n\n")
	}

	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(result.ID)
	builder.WriteString("\n")
	builder.WriteString("과목: ")
	builder.WriteString(result.CourseName)
	builder.WriteString("\n")
	builder.WriteString("제목: ")
	builder.WriteString(detail.Title)
	builder.WriteString("\n")
	builder.WriteString("마감: ")
	if detail.DueAt == nil {
		builder.WriteString("마감 확인 필요")
	} else {
		builder.WriteString(detail.DueAt.Format("2006-01-02 15:04"))
	}
	builder.WriteString("\n")
	builder.WriteString("상태: ")
	if detail.Submitted {
		builder.WriteString("제출")
	} else {
		builder.WriteString("미제출")
	}
	builder.WriteString("\n")
	if detail.ReportType != "" {
		builder.WriteString("제출 방식: ")
		builder.WriteString(detail.ReportType)
		builder.WriteString("\n")
	}
	if detail.SubmitFileType != "" {
		builder.WriteString("파일 형식: ")
		builder.WriteString(detail.SubmitFileType)
		builder.WriteString("\n")
	}
	if detail.FileLimitMB != "" {
		builder.WriteString("파일 제한: ")
		builder.WriteString(detail.FileLimitMB)
		builder.WriteString("MB\n")
	}
	if hashtags := reminderHashtags(result.TermValue, result.CourseName); hashtags != "" {
		builder.WriteString(hashtags)
		builder.WriteString("\n")
	}
	builder.WriteString("[This reminder is created by KLAP.]")

	return builder.String()
}

func buildLectureReminderNotes(row LectureRow) string {
	var builder strings.Builder
	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(lectureReminderID(row.ID))
	builder.WriteString("\n")
	builder.WriteString("과목: ")
	builder.WriteString(row.CourseName)
	builder.WriteString("\n")
	builder.WriteString("제목: ")
	builder.WriteString(firstNonEmpty(row.Lecture.Title, row.Lecture.ModuleTitle, "온라인 강의"))
	builder.WriteString("\n")
	builder.WriteString("마감: ")
	if row.Lecture.EndAt == nil {
		builder.WriteString("마감 확인 필요")
	} else {
		builder.WriteString(row.Lecture.EndAt.Format("2006-01-02 15:04"))
	}
	builder.WriteString("\n")
	builder.WriteString("상태: ")
	builder.WriteString(lectureDueStatus(row.Lecture))
	builder.WriteString("\n")
	if row.Lecture.ModuleTitle != "" {
		builder.WriteString("주차/모듈: ")
		builder.WriteString(row.Lecture.ModuleTitle)
		builder.WriteString("\n")
	}
	if hashtags := reminderHashtags(row.TermValue, row.CourseName); hashtags != "" {
		builder.WriteString(hashtags)
		builder.WriteString("\n")
	}
	builder.WriteString("[This reminder is created by KLAP.]")
	return builder.String()
}

func buildAcademicCalendarNotes(event AcademicEvent) string {
	var builder strings.Builder
	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(academicEventID(event))
	builder.WriteString("\n")
	builder.WriteString("학년도: ")
	builder.WriteString(event.Year)
	builder.WriteString("\n")
	builder.WriteString("날짜: ")
	builder.WriteString(event.Month)
	builder.WriteString(" ")
	builder.WriteString(event.Date)
	builder.WriteString("\n")
	if strings.TrimSpace(event.Note) != "" {
		builder.WriteString("비고: ")
		builder.WriteString(event.Note)
		builder.WriteString("\n")
	}
	builder.WriteString("[This calendar event is created by KLAP.]")
	return builder.String()
}

func academicEventID(event AcademicEvent) string {
	parts := []string{event.Year, event.Title, event.Note}
	for index, part := range parts {
		parts[index] = strings.TrimSpace(part)
	}
	return "academic:" + strings.Join(compactNonEmpty(parts), ":")
}

func buildTimetableCalendarNotes(termValue string, entry TimetableEntry) string {
	var builder strings.Builder
	builder.WriteString("--- KLAP ---\n\n")
	builder.WriteString("ID: ")
	builder.WriteString(timetableCalendarEventID(termValue, entry))
	builder.WriteString("\n")
	builder.WriteString("학기: ")
	builder.WriteString(termLabel(termValue))
	builder.WriteString("\n")
	builder.WriteString("과목: ")
	builder.WriteString(entry.SubjectName)
	builder.WriteString("\n")
	builder.WriteString("교시: ")
	builder.WriteString(fmt.Sprintf("%s %d교시", RoomWeekdayLabel(entry.Weekday), entry.Period))
	if entry.Span > 1 {
		builder.WriteString(fmt.Sprintf(" (%d교시 연강)", entry.Span))
	}
	builder.WriteString("\n")
	if strings.TrimSpace(entry.Room) != "" {
		builder.WriteString("강의실: ")
		builder.WriteString(entry.Room)
		builder.WriteString("\n")
	}
	if strings.TrimSpace(entry.Professor) != "" {
		builder.WriteString("교수: ")
		builder.WriteString(entry.Professor)
		builder.WriteString("\n")
	}
	builder.WriteString("[This calendar event is created by KLAP.]")
	return builder.String()
}

func timetableCalendarEventID(termValue string, entry TimetableEntry) string {
	parts := []string{
		"timetable",
		strings.TrimSpace(termValue),
		strings.TrimSpace(entry.SubjectID),
		strconv.Itoa(entry.Weekday),
		strconv.Itoa(entry.Period),
		strconv.Itoa(entry.Span),
		strings.TrimSpace(entry.Room),
	}
	return strings.Join(compactNonEmpty(parts), ":")
}

func reminderHashtags(termValue string, courseName string) string {
	tags := make([]string, 0, 2)
	if tag := termHashtag(termValue); tag != "" {
		tags = append(tags, tag)
	}
	if tag := courseHashtag(courseName); tag != "" {
		tags = append(tags, tag)
	}
	return strings.Join(tags, " ")
}

func termHashtag(termValue string) string {
	termValue = strings.TrimSpace(termValue)
	if termValue == "" {
		return ""
	}
	parts := strings.FieldsFunc(termValue, func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(parts) >= 2 {
		year := strings.TrimSpace(parts[0])
		switch strings.TrimSpace(parts[1]) {
		case "3":
			return "#" + year + "-여름학기"
		case "4":
			return "#" + year + "-겨울학기"
		}
	}
	return "#" + strings.ReplaceAll(termValue, ",", "-")
}

func courseHashtag(courseName string) string {
	courseName = strings.TrimSpace(courseName)
	if courseName == "" {
		return ""
	}
	return "#" + strings.Join(strings.Fields(courseName), "")
}

func lectureReminderID(rowID string) string {
	rowID = strings.TrimSpace(rowID)
	if rowID == "" {
		return ""
	}
	if strings.HasPrefix(rowID, "lecture:") {
		return rowID
	}
	return "lecture:" + rowID
}

type ReminderSyncer interface {
	Sync(context.Context, reminder.SyncRequest) (reminder.SyncResult, error)
}

type CalendarSyncer interface {
	Sync(klapcalendar.SyncRequest) (klapcalendar.SyncResult, error)
}
