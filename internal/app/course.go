package app

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"strconv"
	"strings"
)

type CourseListOptions struct {
	User    UserOption
	Refresh bool
}

type selectedCourse struct {
	Index  int
	Course klas.Course
}

type CourseRef struct {
	TermValue string
	CourseID  string
}

func NewCourseRef(termValue string, course klas.Course) (CourseRef, error) {
	ref := CourseRef{
		TermValue: strings.TrimSpace(termValue),
		CourseID:  strings.TrimSpace(course.Value),
	}
	if ref.TermValue == "" {
		return CourseRef{}, errors.New("CourseRef에 학기 값이 없습니다")
	}
	if ref.CourseID == "" {
		return CourseRef{}, errors.New("CourseRef에 과목 ID가 없습니다")
	}
	return ref, nil
}

const stableResourceIDVersion = "v1"

func stableCourseResourceID(kind string, ref CourseRef, remoteParts ...string) (string, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" || strings.Contains(kind, ":") {
		return "", errors.New("stable resource kind가 올바르지 않습니다")
	}
	if strings.TrimSpace(ref.TermValue) == "" || strings.TrimSpace(ref.CourseID) == "" {
		return "", errors.New("stable resource ID에 CourseRef가 필요합니다")
	}
	encoded := []string{
		kind,
		stableResourceIDVersion,
		encodeIDPart(ref.TermValue),
		encodeIDPart(ref.CourseID),
	}
	for _, part := range remoteParts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", errors.New("stable resource ID의 remote ID가 비어 있습니다")
		}
		encoded = append(encoded, encodeIDPart(part))
	}
	return strings.Join(encoded, ":"), nil
}

func parseStableCourseResourceID(kind string, id string, remotePartCount int) (CourseRef, []string, bool, error) {
	prefix := strings.TrimSpace(kind) + ":"
	id = strings.TrimSpace(id)
	if !strings.HasPrefix(id, prefix) {
		return CourseRef{}, nil, false, nil
	}
	parts := strings.Split(id, ":")
	if len(parts) != 4+remotePartCount || parts[1] != stableResourceIDVersion {
		return CourseRef{}, nil, true, fmt.Errorf("%s stable ID 형식이 올바르지 않습니다", kind)
	}
	termValue, err := decodeIDPart(parts[2])
	if err != nil {
		return CourseRef{}, nil, true, fmt.Errorf("stable ID 학기 파싱 실패: %w", err)
	}
	courseID, err := decodeIDPart(parts[3])
	if err != nil {
		return CourseRef{}, nil, true, fmt.Errorf("stable ID 과목 파싱 실패: %w", err)
	}
	ref := CourseRef{TermValue: strings.TrimSpace(termValue), CourseID: strings.TrimSpace(courseID)}
	if ref.TermValue == "" || ref.CourseID == "" {
		return CourseRef{}, nil, true, errors.New("stable ID의 CourseRef가 비어 있습니다")
	}
	remoteParts := make([]string, 0, remotePartCount)
	for _, encoded := range parts[4:] {
		value, err := decodeIDPart(encoded)
		if err != nil {
			return CourseRef{}, nil, true, fmt.Errorf("stable ID remote 값 파싱 실패: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return CourseRef{}, nil, true, errors.New("stable ID의 remote ID가 비어 있습니다")
		}
		remoteParts = append(remoteParts, value)
	}
	return ref, remoteParts, true, nil
}

func encodeIDPart(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.TrimSpace(value)))
}

func decodeIDPart(value string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (s *Service) CourseList(ctx context.Context, opts CourseListOptions) ([]klas.Term, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return nil, err
	}

	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return nil, err
	}

	term, _, err := s.selectedTerm(ctx, studentID, client)
	if err != nil {
		return nil, err
	}
	cacheKey := listCacheKey("course", studentID, term.Value, "")
	if !opts.Refresh {
		var cached []klas.Term
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}
	result := []klas.Term{term}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func (s *Service) courses(ctx context.Context, studentID string, client *klas.Client) ([]klas.Term, *klas.Client, error) {
	terms, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.Term, error) {
		return client.Courses(ctx)
	})
	if err != nil {
		return nil, client, err
	}
	return terms, client, nil
}

func courseResourceListCacheKey(scope string, studentID string, termValue string, courses []selectedCourse) string {
	return courseResourceListCacheKeyVersion(scope, "v1", studentID, termValue, courses)
}

func courseResourceListCacheKeyVersion(scope string, version string, studentID string, termValue string, courses []selectedCourse) string {
	courseIDs := make([]string, 0, len(courses))
	for _, course := range courses {
		courseIDs = append(courseIDs, strings.TrimSpace(course.Course.Value))
	}
	sort.Strings(courseIDs)
	parts := []string{
		strings.TrimSpace(scope),
		strings.TrimSpace(version),
		strings.TrimSpace(studentID),
		strings.TrimSpace(termValue),
		"courses-" + hashSyncParts(courseIDs...)[:24],
	}
	return strings.Join(parts, ":")
}

func resourceIDsMatchSelected(ids []string, kind string, remotePartCount int, courses []selectedCourse, allowEmpty bool) bool {
	if len(ids) == 0 {
		return allowEmpty
	}
	selected := make(map[string]struct{}, len(courses))
	for _, course := range courses {
		selected[strings.TrimSpace(course.Course.Value)] = struct{}{}
	}
	for _, id := range ids {
		ref, _, stable, err := parseStableCourseResourceID(kind, id, remotePartCount)
		if err != nil || !stable {
			return false
		}
		if _, ok := selected[ref.CourseID]; !ok {
			return false
		}
	}
	return true
}

func selectedCourses(term klas.Term, filter string) ([]selectedCourse, error) {
	if strings.TrimSpace(filter) == "" {
		selected := make([]selectedCourse, 0, len(term.Courses))
		for index, course := range term.Courses {
			selected = append(selected, selectedCourse{Index: index + 1, Course: course})
		}
		return selected, nil
	}

	if number, err := strconv.Atoi(filter); err == nil {
		if number < 1 || number > len(term.Courses) {
			return nil, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", number)
		}
		return []selectedCourse{{Index: number, Course: term.Courses[number-1]}}, nil
	}

	normalizedFilter := strings.ToLower(strings.TrimSpace(filter))
	var containsMatches []selectedCourse
	for index, course := range term.Courses {
		normalizedName := strings.ToLower(strings.TrimSpace(course.Name))
		if normalizedName == normalizedFilter {
			return []selectedCourse{{Index: index + 1, Course: course}}, nil
		}
		if strings.Contains(normalizedName, normalizedFilter) {
			containsMatches = append(containsMatches, selectedCourse{Index: index + 1, Course: course})
		}
	}
	if len(containsMatches) == 1 {
		return containsMatches, nil
	}
	if len(containsMatches) > 1 {
		return nil, fmt.Errorf("과목명이 여러 개와 일치합니다. course list 번호를 사용하세요: %s", filter)
	}
	return nil, fmt.Errorf("과목을 찾을 수 없습니다: %s", filter)
}

type courseResourceLocator struct {
	Ref         CourseRef
	CourseIndex int
	Stable      bool
}

func resolveResourceCourse(term klas.Term, locator courseResourceLocator) (klas.Course, error) {
	if locator.Stable {
		if strings.TrimSpace(term.Value) != strings.TrimSpace(locator.Ref.TermValue) {
			return klas.Course{}, fmt.Errorf("stable ID 학기와 조회 학기가 다릅니다: %s != %s", locator.Ref.TermValue, term.Value)
		}
		for _, course := range term.Courses {
			if strings.TrimSpace(course.Value) == locator.Ref.CourseID {
				return course, nil
			}
		}
		return klas.Course{}, fmt.Errorf("학기 %s에서 과목을 찾을 수 없습니다: %s", term.Value, locator.Ref.CourseID)
	}
	if locator.CourseIndex < 1 || locator.CourseIndex > len(term.Courses) {
		return klas.Course{}, fmt.Errorf("과목 번호가 범위를 벗어났습니다: %d", locator.CourseIndex)
	}
	return term.Courses[locator.CourseIndex-1], nil
}

func resolveLegacyCachedCourse(term klas.Term, courseIndex int, courseName string) (klas.Course, error) {
	normalizedName := strings.TrimSpace(courseName)
	if normalizedName == "" {
		return klas.Course{}, fmt.Errorf("legacy cache 과목명 없이 순번을 안전하게 이전할 수 없습니다: %d", courseIndex)
	}
	var match *klas.Course
	for index := range term.Courses {
		if !strings.EqualFold(strings.TrimSpace(term.Courses[index].Name), normalizedName) {
			continue
		}
		if match != nil {
			return klas.Course{}, fmt.Errorf("legacy cache 과목명이 여러 과목과 일치합니다: %s", courseName)
		}
		match = &term.Courses[index]
	}
	if match != nil {
		return *match, nil
	}
	return klas.Course{}, fmt.Errorf("legacy cache 과목을 찾을 수 없습니다: %s", courseName)
}

func courseIndexByID(term klas.Term, courseID string) (int, error) {
	for index, course := range term.Courses {
		if strings.TrimSpace(course.Value) == strings.TrimSpace(courseID) {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("학기 %s에서 과목을 찾을 수 없습니다: %s", term.Value, courseID)
}
