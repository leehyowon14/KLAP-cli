package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type NoticeListOptions struct {
	User         UserOption
	CourseFilter string
	Refresh      bool
}

type NoticeRow struct {
	ID         string
	TermValue  string
	CourseName string
	DetailURL  string
	Notice     klas.Notice
}

type NoticeDetailResult struct {
	ID         string
	TermValue  string
	CourseName string
	DetailURL  string
	Detail     klas.NoticeDetail
}

func (s *Service) NoticeList(ctx context.Context, opts NoticeListOptions) ([]NoticeRow, error) {
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

	cacheKey := courseResourceListCacheKeyVersion("notice", "v2", studentID, term.Value, courses)
	if !opts.Refresh {
		var cached []NoticeRow
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			rows, migrated, migrationErr := normalizeCachedNoticeRows(cached, term)
			if migrationErr == nil && resourceIDsMatchSelected(noticeRowIDs(rows), "notice", 2, courses, true) {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
				}
				return rows, nil
			}
		}
	}

	rows := make([]NoticeRow, 0)
	for _, selectedCourse := range courses {
		notices, err := client.Notices(ctx, term.Value, selectedCourse.Course)
		if err != nil {
			refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
			if refreshErr != nil {
				return nil, refreshErr
			}
			if refreshed {
				client = refreshedClient
				notices, err = client.Notices(ctx, term.Value, selectedCourse.Course)
			}
		}
		if err != nil {
			return nil, err
		}
		for _, notice := range notices {
			ref, err := NewCourseRef(term.Value, selectedCourse.Course)
			if err != nil {
				return nil, err
			}
			id, err := StableNoticeID(ref, notice.BoardNo, notice.MasterNo)
			if err != nil {
				return nil, err
			}
			rows = append(rows, NoticeRow{
				ID:         id,
				TermValue:  term.Value,
				CourseName: selectedCourse.Course.Name,
				DetailURL:  noticeDetailURL(term.Value, selectedCourse.Course, notice.BoardNo, notice.MasterNo),
				Notice:     notice,
			})
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Notice.Top != rows[j].Notice.Top {
			return rows[i].Notice.Top
		}
		left := rows[i].Notice.Registered
		right := rows[j].Notice.Registered
		if left == nil && right == nil {
			return rows[i].ID < rows[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return right.Before(*left)
	})

	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), rows)
	return rows, nil
}

func (s *Service) NoticeDetail(ctx context.Context, id string, user UserOption) (NoticeDetailResult, error) {
	studentID, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return NoticeDetailResult{}, err
	}

	locator, boardNo, masterNo, err := parseNoticeResourceID(id)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	var term klas.Term
	if locator.Stable {
		term, client, err = s.termForSyllabus(ctx, studentID, client, locator.Ref.TermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return NoticeDetailResult{}, err
	}
	course, err := resolveResourceCourse(term, locator)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	ref, err := NewCourseRef(term.Value, course)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	canonicalID, err := StableNoticeID(ref, boardNo, masterNo)
	if err != nil {
		return NoticeDetailResult{}, err
	}
	detail, err := client.NoticeDetail(ctx, term.Value, course, boardNo, masterNo)
	if err != nil {
		refreshedClient, refreshed, refreshErr := s.refreshedClientAfterSessionError(ctx, studentID, err)
		if refreshErr != nil {
			return NoticeDetailResult{}, refreshErr
		}
		if refreshed {
			client = refreshedClient
			detail, err = client.NoticeDetail(ctx, term.Value, course, boardNo, masterNo)
		}
	}
	if err != nil {
		return NoticeDetailResult{}, err
	}

	return NoticeDetailResult{
		ID:         canonicalID,
		TermValue:  term.Value,
		CourseName: course.Name,
		DetailURL:  noticeDetailURL(term.Value, course, boardNo, masterNo),
		Detail:     detail,
	}, nil
}

func (s *Service) NoticeOpenURL(ctx context.Context, id string, user UserOption) (OpenURLResult, error) {
	detail, err := s.NoticeDetail(ctx, id, user)
	if err != nil {
		return OpenURLResult{}, err
	}
	return OpenURLResult{URL: detail.DetailURL}, nil
}

func noticeDetailURL(yearHakgi string, course klas.Course, boardNo string, masterNo string) string {
	values := url.Values{}
	values.Set("selectYearhakgi", yearHakgi)
	values.Set("selectSubj", course.Value)
	values.Set("boardNo", strings.TrimSpace(boardNo))
	values.Set("masterNo", strings.TrimSpace(masterNo))
	return "https://klas.kw.ac.kr/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardViewStdPage.do?" + values.Encode()
}

func noticeRowIDs(rows []NoticeRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func limitNotices(rows []NoticeRow, limit int) []NoticeRow {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	return rows[:limit]
}

func NoticeID(courseIndex int, boardNo string, masterNo string) string {
	return fmt.Sprintf("%d:%s:%s", courseIndex, strings.TrimSpace(boardNo), strings.TrimSpace(masterNo))
}

func StableNoticeID(ref CourseRef, boardNo string, masterNo string) (string, error) {
	return stableCourseResourceID("notice", ref, boardNo, masterNo)
}

func parseNoticeResourceID(id string) (courseResourceLocator, string, string, error) {
	ref, remoteParts, stable, err := parseStableCourseResourceID("notice", id, 2)
	if err != nil {
		return courseResourceLocator{}, "", "", err
	}
	if stable {
		return courseResourceLocator{Ref: ref, Stable: true}, remoteParts[0], remoteParts[1], nil
	}
	courseIndex, boardNo, masterNo, err := ParseNoticeID(id)
	if err != nil {
		return courseResourceLocator{}, "", "", err
	}
	return courseResourceLocator{CourseIndex: courseIndex}, boardNo, masterNo, nil
}

func normalizeCachedNoticeRows(rows []NoticeRow, term klas.Term) ([]NoticeRow, bool, error) {
	migrated := false
	for index := range rows {
		locator, boardNo, masterNo, err := parseNoticeResourceID(rows[index].ID)
		if err != nil {
			return nil, false, err
		}
		if locator.Stable {
			if _, err := resolveResourceCourse(term, locator); err != nil {
				return nil, false, err
			}
			continue
		}
		course, err := resolveLegacyCachedCourse(term, locator.CourseIndex, rows[index].CourseName)
		if err != nil {
			return nil, false, err
		}
		ref, err := NewCourseRef(term.Value, course)
		if err != nil {
			return nil, false, err
		}
		stableID, err := StableNoticeID(ref, boardNo, masterNo)
		if err != nil {
			return nil, false, err
		}
		rows[index].ID = stableID
		rows[index].TermValue = term.Value
		migrated = true
	}
	return rows, migrated, nil
}

func ParseNoticeID(id string) (int, string, string, error) {
	parts := strings.Split(strings.TrimSpace(id), ":")
	if len(parts) != 3 {
		return 0, "", "", errors.New("공지ID는 course list 번호, boardNo, masterNo를 조합한 <과목번호>:<boardNo>:<masterNo> 형식이어야 합니다")
	}

	courseIndex, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("과목 번호 파싱 실패: %w", err)
	}
	boardNo := strings.TrimSpace(parts[1])
	if boardNo == "" {
		return 0, "", "", errors.New("공지ID에 boardNo가 없습니다")
	}
	masterNo := strings.TrimSpace(parts[2])
	if masterNo == "" {
		return 0, "", "", errors.New("공지ID에 masterNo가 없습니다")
	}
	return courseIndex, boardNo, masterNo, nil
}
