package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
	"time"
)

type ItemDetail struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind,omitempty"`
	Status      string         `json:"status"`
	Error       string         `json:"error,omitempty"`
	Title       string         `json:"title,omitempty"`
	CourseID    string         `json:"course_id,omitempty"`
	Term        string         `json:"term,omitempty"`
	FetchedAt   time.Time      `json:"fetched_at"`
	Sections    map[string]any `json:"sections,omitempty"`
	Unsupported []string       `json:"unsupported_sections,omitempty"`
}
type ItemDetails struct {
	Items []ItemDetail `json:"items"`
}

func ValidateDetailRequest(ids, sections []string) error {
	if len(ids) == 0 || len(sections) == 0 {
		return errors.New("ids and sections must not be empty")
	}
	for _, section := range sections {
		switch section {
		case "completion", "content", "learning", "submission", "attachments":
		default:
			return errors.New("unknown detail section")
		}
	}
	return nil
}
func detailRef(id string) (string, CourseRef, error) {
	kind, _, _ := strings.Cut(id, ":")
	parts := 1
	if kind == "notice" {
		parts = 2
	} else if kind != "lecture" && kind != "assignment" {
		return kind, CourseRef{}, errors.New("unsupported resource kind")
	}
	ref, _, stable, err := parseStableCourseResourceID(kind, id, parts)
	if err != nil {
		return kind, ref, err
	}
	if !stable {
		return kind, ref, errors.New("stable ID required")
	}
	return kind, ref, nil
}

// GetItems is fresh per call. Lecture lists are reused only within this batch,
// keyed by term AND course, including failed lookups. No eligibility filter runs.
func (s *Service) GetItems(ctx context.Context, user UserOption, ids, sections []string) (ItemDetails, error) {
	if err := ValidateDetailRequest(ids, sections); err != nil {
		return ItemDetails{}, err
	}
	s = s.withRequestSession()
	result := ItemDetails{Items: []ItemDetail{}}
	seen := map[string]bool{}
	type cached struct {
		rows    []LectureRow
		err     error
		fetched time.Time
	}
	lectures := map[CourseRef]cached{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		kind, ref, err := detailRef(id)
		item := ItemDetail{ID: id, Kind: kind, CourseID: ref.CourseID, Term: ref.TermValue, Status: "ok", FetchedAt: time.Now().UTC(), Sections: map[string]any{}}
		if err != nil {
			item.Status = "error"
			item.Error = "invalid_id"
			result.Items = append(result.Items, item)
			continue
		}
		requested := map[string]bool{}
		for _, section := range sections {
			supported := (kind == "lecture" && (section == "completion" || section == "learning")) || (kind == "notice" && (section == "content" || section == "attachments")) || (kind == "assignment" && (section == "content" || section == "submission"))
			if supported {
				requested[section] = true
			} else {
				item.Unsupported = append(item.Unsupported, section)
			}
		}
		if len(requested) == 0 {
			item.Status = "unsupported"
			result.Items = append(result.Items, item)
			continue
		}
		switch kind {
		case "lecture":
			entry, ok := lectures[ref]
			if !ok {
				student, e := s.selectedStudentID(ctx, user)
				var resource resolvedLectureResource
				if e == nil {
					resource, e = s.resolveLectureResource(ctx, student, id)
				}
				if e == nil {
					var values []klas.Lecture
					values, e = executeSessionRequest(ctx, s, student, &resource.Client, func(c *klas.Client) ([]klas.Lecture, error) { return c.Lectures(ctx, ref.TermValue, resource.Course) })
					if e == nil {
						for _, v := range values {
							row, rowErr := newLectureRow(ref.TermValue, selectedCourse{Course: resource.Course}, lectureModel(v))
							if rowErr != nil {
								e = rowErr
								break
							}
							entry.rows = append(entry.rows, row)
						}
					}
				}
				entry.err = e
				entry.fetched = time.Now().UTC()
				lectures[ref] = entry
			}
			err = entry.err
			item.FetchedAt = entry.fetched
			if err == nil {
				found := false
				for _, row := range entry.rows {
					if row.ID != id {
						continue
					}
					found = true
					item.Title = row.Lecture.Title
					if requested["completion"] {
						item.Sections["completion"] = LectureCompletion(row.Lecture)
					}
					if requested["learning"] {
						item.Sections["learning"] = map[string]any{"completion": LectureCompletion(row.Lecture), "start_at": row.Lecture.StartAt, "end_at": row.Lecture.EndAt, "availability": lectureAvailability(row.Lecture, item.FetchedAt), "week": row.Lecture.WeekNo}
					}
					break
				}
				if !found {
					item.Status = "not_found"
				}
			}
		case "notice":
			var detail NoticeDetailResult
			detail, err = s.NoticeDetail(ctx, id, user)
			item.Title = detail.Detail.Title
			if err == nil {
				if requested["content"] {
					item.Sections["content"] = detail.Detail.ContentText
				}
				if requested["attachments"] {
					item.Sections["attachments"] = detail.Detail.Attachment
				}
			}
		case "assignment":
			var detail AssignmentDetailResult
			detail, err = s.AssignmentDetail(ctx, id, user)
			item.Title = detail.Detail.Title
			if err == nil {
				d := detail.Detail
				if requested["content"] {
					item.Sections["content"] = d.ContentText
				}
				if requested["submission"] {
					item.Sections["submission"] = map[string]any{"submitted": d.Submitted, "report_type": d.ReportType, "file_type": d.SubmitFileType, "file_limit_mb": d.FileLimitMB, "submitted_title": d.SubmittedTitle, "submitted_text": d.SubmittedText, "score": d.FinalScore, "feedback": d.TutorText}
				}
			}
		}
		if kind != "lecture" {
			item.FetchedAt = time.Now().UTC()
		}
		if err != nil {
			item.Status = "error"
			item.Error = queryFailure(ref.CourseID, err).Reason
			item.Sections = nil
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}
