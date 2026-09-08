package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"strconv"
	"strings"
	"time"
)

type LectureListOptions struct {
	User          UserOption
	CourseFilter  string
	Refresh       bool
	SyncDecisions map[string]SyncDecision
}

type LectureAttendOptions struct {
	User       UserOption
	Interval   time.Duration
	OnProgress func(LectureRow, klas.LectureProgress)
}

type LectureAttendAllOptions struct {
	User         UserOption
	CourseFilter string
	Interval     time.Duration
	OnProgress   func(LectureRow, klas.LectureProgress)
}

type LectureRow struct {
	ID         string
	LegacyID   string `json:"-"`
	TermValue  string
	CourseName string
	Lecture    klas.Lecture
}

type LectureAttendResult struct {
	Lecture  LectureRow
	Progress klas.LectureProgress
}

type LectureAttendItem struct {
	Lecture  LectureRow
	Progress klas.LectureProgress
	Err      error
}

type LectureAttendAllResult struct {
	Items []LectureAttendItem
}

func (s *Service) LectureList(ctx context.Context, opts LectureListOptions) ([]LectureRow, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return nil, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return nil, err
	}

	cacheKey := courseResourceListCacheKeyVersion("lecture", "v2", studentID, term.Value, courses)
	if !opts.Refresh {
		var cached []LectureRow
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			rows, migrated, migrationErr := normalizeCachedLectureRows(cached, term)
			if migrationErr == nil && resourceIDsMatchSelected(lectureRowIDs(rows), "lecture", 1, courses, true) {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
				}
				return rows, nil
			}
		}
	}

	rows := make([]LectureRow, 0)
	for _, selectedCourse := range courses {
		lectures, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.Lecture, error) {
			return client.Lectures(ctx, term.Value, selectedCourse.Course)
		})
		if err != nil {
			return nil, err
		}
		for _, lecture := range lectures {
			row, err := newLectureRow(term.Value, selectedCourse, lecture)
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i].Lecture.StartAt
		right := rows[j].Lecture.StartAt
		if left == nil && right == nil {
			return rows[i].ID < rows[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return rows[i].Lecture.Title < rows[j].Lecture.Title
		}
		return left.Before(*right)
	})

	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
	return rows, nil
}

func (s *Service) LectureOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	resource, err := s.resolveLectureResource(ctx, studentID, id)
	if err != nil {
		return OpenURLResult{}, err
	}
	lectures, err := executeSessionRequest(ctx, s, studentID, &resource.Client, func(client *klas.Client) ([]klas.Lecture, error) {
		return client.Lectures(ctx, resource.Term.Value, resource.Course)
	})
	if err != nil {
		return OpenURLResult{}, err
	}

	for _, lecture := range lectures {
		if !lectureMatchesKey(lecture, resource.Key) {
			continue
		}
		if strings.TrimSpace(lecture.PlayURL) == "" {
			return OpenURLResult{}, fmt.Errorf("열 수 있는 강의 URL이 없습니다: %s", id)
		}
		return OpenURLResult{URL: lecture.PlayURL}, nil
	}
	return OpenURLResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
}

func (s *Service) AttendLecture(ctx context.Context, id string, opts LectureAttendOptions) (LectureAttendResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureAttendResult{}, err
	}
	resource, err := s.resolveLectureResource(ctx, studentID, id)
	if err != nil {
		return LectureAttendResult{}, err
	}
	lectures, err := executeSessionRequest(ctx, s, studentID, &resource.Client, func(client *klas.Client) ([]klas.Lecture, error) {
		return client.Lectures(ctx, resource.Term.Value, resource.Course)
	})
	if err != nil {
		return LectureAttendResult{}, err
	}

	var matched *klas.Lecture
	for index := range lectures {
		if lectureMatchesKey(lectures[index], resource.Key) {
			matched = &lectures[index]
			break
		}
	}
	if matched == nil {
		return LectureAttendResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
	}

	row := LectureRow{
		ID:         resource.ID,
		LegacyID:   resource.LegacyID,
		TermValue:  resource.Term.Value,
		CourseName: resource.Course.Name,
		Lecture:    *matched,
	}
	progress, err := attendLecture(ctx, resource.Client, row, opts.Interval, opts.OnProgress)
	if err != nil {
		return LectureAttendResult{}, err
	}
	return LectureAttendResult{Lecture: row, Progress: progress}, nil
}

func (s *Service) AttendAllLectures(ctx context.Context, opts LectureAttendAllOptions) (LectureAttendAllResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureAttendAllResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return LectureAttendAllResult{}, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return LectureAttendAllResult{}, err
	}

	result := LectureAttendAllResult{}
	now := time.Now()
	for _, selectedCourse := range courses {
		lectures, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.Lecture, error) {
			return client.Lectures(ctx, term.Value, selectedCourse.Course)
		})
		if err != nil {
			return LectureAttendAllResult{}, err
		}

		for _, lecture := range lectures {
			if !lectureNeedsAttendance(lecture, now) {
				continue
			}
			row, rowErr := newLectureRow(term.Value, selectedCourse, lecture)
			if rowErr != nil {
				return LectureAttendAllResult{}, rowErr
			}
			progress, err := attendLecture(ctx, client, row, opts.Interval, opts.OnProgress)
			result.Items = append(result.Items, LectureAttendItem{
				Lecture:  row,
				Progress: progress,
				Err:      err,
			})
		}
	}
	return result, nil
}

func lectureRowIDs(rows []LectureRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func lectureDueStatus(lecture klas.Lecture) string {
	if lectureIsLearningActivity(lecture) {
		return formatProgressValue(lecture.AchievedTime, lecture.RequiredTime)
	}
	return emptyStatusFallback(lecture.Progress, "진도 확인 필요")
}

func limitLectures(rows []LectureRow, limit int) []LectureRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func LectureID(courseIndex int, lectureKey string) string {
	lectureKey = strings.TrimSpace(lectureKey)
	if lectureKey == "" {
		return "-"
	}
	return fmt.Sprintf("%d:%s", courseIndex, lectureKey)
}

func StableLectureID(ref CourseRef, lectureKey string) (string, error) {
	return stableCourseResourceID("lecture", ref, lectureKey)
}

func parseLectureResourceID(id string) (courseResourceLocator, string, error) {
	ref, remoteParts, stable, err := parseStableCourseResourceID("lecture", id, 1)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	if stable {
		return courseResourceLocator{Ref: ref, Stable: true}, remoteParts[0], nil
	}
	courseIndex, lectureKey, err := ParseLectureID(id)
	if err != nil {
		return courseResourceLocator{}, "", err
	}
	return courseResourceLocator{CourseIndex: courseIndex}, lectureKey, nil
}

func stableLectureTermValue(ids []string) (string, error) {
	termValue := ""
	for _, id := range ids {
		ref, _, stable, err := parseStableCourseResourceID("lecture", id, 1)
		if err != nil {
			return "", err
		}
		if !stable {
			continue
		}
		if termValue != "" && termValue != ref.TermValue {
			return "", errors.New("서로 다른 학기의 강의를 한 번에 처리할 수 없습니다")
		}
		termValue = ref.TermValue
	}
	return termValue, nil
}

type resolvedLectureResource struct {
	Client   *klas.Client
	Term     klas.Term
	Course   klas.Course
	Key      string
	ID       string
	LegacyID string
}

func (s *Service) resolveLectureResource(ctx context.Context, studentID string, id string) (resolvedLectureResource, error) {
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	locator, lectureKey, err := parseLectureResourceID(id)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	var term klas.Term
	if locator.Stable {
		term, client, err = s.termForSyllabus(ctx, studentID, client, locator.Ref.TermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return resolvedLectureResource{}, err
	}
	course, err := resolveResourceCourse(term, locator)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	ref, err := NewCourseRef(term.Value, course)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	stableID, err := StableLectureID(ref, lectureKey)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	courseIndex, err := courseIndexByID(term, course.Value)
	if err != nil {
		return resolvedLectureResource{}, err
	}
	return resolvedLectureResource{
		Client:   client,
		Term:     term,
		Course:   course,
		Key:      lectureKey,
		ID:       stableID,
		LegacyID: LectureID(courseIndex, lectureKey),
	}, nil
}

func newLectureRow(termValue string, selected selectedCourse, lecture klas.Lecture) (LectureRow, error) {
	ref, err := NewCourseRef(termValue, selected.Course)
	if err != nil {
		return LectureRow{}, err
	}
	lectureKey := lectureResourceKey(lecture)
	id, err := StableLectureID(ref, lectureKey)
	if err != nil {
		return LectureRow{}, err
	}
	legacyID := lectureRowID(selected.Index, lecture)
	if legacyID == "-" {
		legacyID = ""
	}
	return LectureRow{
		ID:         id,
		LegacyID:   legacyID,
		TermValue:  strings.TrimSpace(termValue),
		CourseName: selected.Course.Name,
		Lecture:    lecture,
	}, nil
}

func normalizeCachedLectureRows(rows []LectureRow, term klas.Term) ([]LectureRow, bool, error) {
	migrated := false
	for index := range rows {
		locator, lectureKey, parseErr := parseLectureResourceID(rows[index].ID)
		legacyID := ""
		if parseErr != nil && strings.TrimSpace(rows[index].ID) != "-" {
			return nil, false, parseErr
		}
		var course klas.Course
		var err error
		if parseErr == nil && locator.Stable {
			course, err = resolveResourceCourse(term, locator)
			if err != nil {
				return nil, false, err
			}
		} else {
			course, err = resolveLegacyCachedCourse(term, locator.CourseIndex, rows[index].CourseName)
			if err != nil {
				return nil, false, err
			}
			legacyID = strings.TrimSpace(rows[index].ID)
			if legacyID == "-" {
				legacyID = ""
			}
			lectureKey = lectureResourceKey(rows[index].Lecture)
			ref, err := NewCourseRef(term.Value, course)
			if err != nil {
				return nil, false, err
			}
			rows[index].ID, err = StableLectureID(ref, lectureKey)
			if err != nil {
				return nil, false, err
			}
			rows[index].TermValue = term.Value
			migrated = true
		}
		if legacyID == "" {
			courseIndex, err := courseIndexByID(term, course.Value)
			if err != nil {
				return nil, false, err
			}
			legacyID = LectureID(courseIndex, lectureKey)
		}
		rows[index].LegacyID = legacyID
	}
	return rows, migrated, nil
}

func ParseLectureID(id string) (int, string, error) {
	parts := strings.SplitN(strings.TrimSpace(id), ":", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("강의ID는 course list 번호와 강의 키를 조합한 <과목번호>:<contentID|lrn-lrnSn> 형식이어야 합니다")
	}

	courseIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("과목 번호 파싱 실패: %w", err)
	}
	lectureKey := strings.TrimSpace(parts[1])
	if lectureKey == "" {
		return 0, "", errors.New("강의ID에 강의 키가 없습니다")
	}
	return courseIndex, lectureKey, nil
}

func attendLecture(ctx context.Context, client *klas.Client, row LectureRow, interval time.Duration, onProgress func(LectureRow, klas.LectureProgress)) (klas.LectureProgress, error) {
	if lectureIsLearningActivity(row.Lecture) {
		status := lectureLearningStatus(row.Lecture, time.Now())
		if status != "Y" {
			return klas.LectureProgress{}, errors.New("학습기간이 아니어서 학습활동 시간이 반영되지 않습니다")
		}
		progress, err := client.SaveLectureLearningStatus(ctx, row.Lecture, status)
		if err != nil {
			return progress, err
		}
		if onProgress != nil {
			onProgress(row, progress)
		}
		return progress, nil
	}
	return attendLectureLoop(ctx, client, row, interval, onProgress)
}

func attendLectureLoop(ctx context.Context, client *klas.Client, row LectureRow, interval time.Duration, onProgress func(LectureRow, klas.LectureProgress)) (klas.LectureProgress, error) {
	if interval <= 0 {
		interval = 60 * time.Second
	}

	lecKey, err := client.LectureKey(ctx, row.Lecture)
	if err != nil {
		return klas.LectureProgress{}, err
	}

	var lastProgress klas.LectureProgress
	for {
		if err := client.CheckLectureView(ctx, row.Lecture, lecKey); err != nil {
			return lastProgress, err
		}
		progress, err := client.UpdateLectureProgress(ctx, row.Lecture, lecKey)
		if err != nil {
			return lastProgress, err
		}
		lastProgress = progress
		if onProgress != nil {
			onProgress(row, progress)
		}
		if progress.Completed {
			return progress, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastProgress, ctx.Err()
		case <-timer.C:
		}
	}
}

func lectureNeedsAttendance(lecture klas.Lecture, now time.Time) bool {
	if lectureIsLearningActivity(lecture) {
		if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
			return false
		}
		if lecture.EndAt != nil && now.After(*lecture.EndAt) {
			return false
		}
		progress, progressErr := strconv.ParseFloat(strings.TrimSpace(lecture.AchievedTime), 64)
		required, requiredErr := strconv.ParseFloat(strings.TrimSpace(lecture.RequiredTime), 64)
		if progressErr == nil && requiredErr == nil && required > 0 && progress >= required {
			return false
		}
		return strings.TrimSpace(lecture.LearningSeq) != ""
	}

	progress, err := strconv.ParseFloat(strings.TrimSpace(lecture.Progress), 64)
	if err == nil && progress >= 100 {
		return false
	}
	if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
		return false
	}
	if lecture.EndAt != nil && now.After(*lecture.EndAt) {
		return false
	}
	return strings.TrimSpace(lecture.ContentID) != ""
}

func lectureIsLearningActivity(lecture klas.Lecture) bool {
	return strings.TrimSpace(lecture.ContentID) == "" && strings.TrimSpace(lecture.LearningSeq) != ""
}

func lectureLearningStatus(lecture klas.Lecture, now time.Time) string {
	if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
		return "N"
	}
	if lecture.EndAt != nil && now.After(*lecture.EndAt) {
		return "N"
	}
	return "Y"
}

func lectureAttendKey(lecture klas.Lecture) string {
	if contentID := strings.TrimSpace(lecture.ContentID); contentID != "" {
		return contentID
	}
	if learningSeq := strings.TrimSpace(lecture.LearningSeq); learningSeq != "" {
		return "lrn-" + learningSeq
	}
	return ""
}

func lectureResourceKey(lecture klas.Lecture) string {
	if key := lectureAttendKey(lecture); key != "" {
		return key
	}
	if fileID := strings.TrimSpace(lecture.FileID); fileID != "" {
		return "file-" + fileID
	}
	if weekNo := strings.TrimSpace(lecture.WeekNo); weekNo != "" || strings.TrimSpace(lecture.WeeklySeq) != "" {
		return "week-" + weekNo + "-" + strings.TrimSpace(lecture.WeeklySeq)
	}
	startAt := ""
	if lecture.StartAt != nil {
		startAt = lecture.StartAt.UTC().Format(time.RFC3339Nano)
	}
	endAt := ""
	if lecture.EndAt != nil {
		endAt = lecture.EndAt.UTC().Format(time.RFC3339Nano)
	}
	return "meta-" + hashSyncParts(lecture.ModuleTitle, lecture.Title, startAt, endAt)[:24]
}

func lectureRowID(courseIndex int, lecture klas.Lecture) string {
	return LectureID(courseIndex, lectureAttendKey(lecture))
}

func lectureMatchesKey(lecture klas.Lecture, key string) bool {
	key = strings.TrimSpace(key)
	return key != "" && lectureResourceKey(lecture) == key
}
