package core

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

// ContentItem is a stable identity and summary, independent of read counts,
// submission status, and lecture progress. OccurredAt is nil when KLAS does not
// provide a creation time; consumers must label their detection time honestly.
type ContentItem struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	CourseName string     `json:"course_name"`
	Term       string     `json:"term"`
	URL        string     `json:"url,omitempty"`
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
}

func Content(ctx context.Context, service *Service, kind, term, course string, user UserOption) ([]ContentItem, error) {
	s, err := service.ForTerm(term)
	if err != nil {
		return nil, err
	}
	items := []ContentItem{}
	switch kind {
	case "notice.created":
		rows, err := s.NoticeList(ctx, NoticeListOptions{User: user, CourseFilter: course, Refresh: true})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			items = append(items, ContentItem{ID: r.ID, Title: r.Notice.Title, CourseName: r.CourseName, Term: r.TermValue, URL: r.DetailURL, OccurredAt: r.Notice.Registered})
		}
	case "assignment.created":
		rows, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, CourseFilter: course, Refresh: true})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			items = append(items, ContentItem{ID: r.ID, Title: r.Assignment.Title, CourseName: r.CourseName, Term: r.TermValue, URL: r.DetailURL, DueAt: r.Assignment.DueAt})
		}
	case "lecture.created":
		rows, err := s.LectureList(ctx, LectureListOptions{User: user, CourseFilter: course, Refresh: true})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			items = append(items, ContentItem{ID: r.ID, Title: r.Lecture.Title, CourseName: r.CourseName, Term: r.TermValue, DueAt: r.Lecture.EndAt})
		}
	default:
		return nil, errors.New("unsupported content kind")
	}
	return items, nil
}

// SeenContent is persisted by the host in the same transaction as its outbox.
// Call DetectContent only after a complete successful snapshot. Never remove IDs
// just because an item disappears from a later response.
type SeenContent struct {
	Initialized bool            `json:"initialized"`
	IDs         map[string]bool `json:"ids"`
}

func DetectContent(previous SeenContent, items []ContentItem) (SeenContent, []ContentItem, error) {
	next := SeenContent{Initialized: true, IDs: map[string]bool{}}
	for id := range previous.IDs {
		next.IDs[id] = true
	}
	added := []ContentItem{}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return previous, nil, errors.New("content snapshot contains an empty stable ID")
		}
		if !next.IDs[item.ID] && previous.Initialized {
			added = append(added, item)
		}
		next.IDs[item.ID] = true
	}
	sort.Slice(added, func(i, j int) bool { return added[i].ID < added[j].ID })
	return next, added, nil
}
