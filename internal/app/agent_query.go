package app

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

// AgentQuery describes meaning, never a number of results.
type AgentQuery struct {
	asOf           time.Time
	Intent         string       `json:"intent"`
	CourseIDs      []string     `json:"course_ids,omitempty"`
	IncludeExpired *bool        `json:"include_expired,omitempty"`
	Period         *QueryPeriod `json:"period,omitempty"`
}
type QueryPeriod struct {
	Name  string `json:"name"`
	From  string `json:"from,omitempty"`
	Until string `json:"until,omitempty"`
}
type QueryWindow struct {
	From  time.Time `json:"from"`
	Until time.Time `json:"until"`
}

func (p QueryPeriod) Resolve(now time.Time) (QueryWindow, error) {
	loc := time.FixedZone("Asia/Seoul", 9*60*60)
	now = now.In(loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if p.Name != "custom" && (p.From != "" || p.Until != "") {
		return QueryWindow{}, errors.New("from/until require custom period")
	}
	var w QueryWindow
	switch p.Name {
	case "today":
		w = QueryWindow{day, day.AddDate(0, 0, 1)}
	case "this_week", "next_week":
		start := day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
		if p.Name == "next_week" {
			start = start.AddDate(0, 0, 7)
		}
		w = QueryWindow{start, start.AddDate(0, 0, 7)}
	case "next_7_days":
		w = QueryWindow{now, now.AddDate(0, 0, 7)}
	case "custom":
		parse := func(v string) (time.Time, error) {
			if len(v) == 10 {
				return time.ParseInLocation("2006-01-02", v, loc)
			}
			return time.Parse(time.RFC3339, v)
		}
		var err error
		w.From, err = parse(p.From)
		if err != nil {
			return w, err
		}
		w.Until, err = parse(p.Until)
		if err != nil {
			return w, err
		}
	default:
		return w, errors.New("unknown period")
	}
	if !w.Until.After(w.From) {
		return w, errors.New("until must be after from")
	}
	return w, nil
}
func (w QueryWindow) Contains(t time.Time) bool { return !t.Before(w.From) && t.Before(w.Until) }

type Completion struct {
	Status   string   `json:"status"`
	Progress *float64 `json:"progress,omitempty"`
	Achieved *float64 `json:"achieved,omitempty"`
	Required *float64 `json:"required,omitempty"`
}

func validNumber(raw string) (float64, bool) {
	v, e := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return v, e == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

// LectureCompletion deliberately ignores availability, playback support and job state.
func LectureCompletion(l Lecture) Completion {
	c := Completion{Status: "unknown"}
	if l.ContentID == "" && l.LearningSeq != "" {
		achieved, a := validNumber(l.AchievedTime)
		required, r := validNumber(l.RequiredTime)
		if !a || !r || required <= 0 {
			return c
		}
		c.Achieved = &achieved
		c.Required = &required
		c.Status = "incomplete"
		if achieved >= required {
			c.Status = "completed"
		}
		return c
	}
	progress, ok := validNumber(l.Progress)
	if !ok || progress > 100 {
		return c
	}
	c.Progress = &progress
	c.Status = "incomplete"
	if progress == 100 {
		c.Status = "completed"
	}
	return c
}

type QueryFailure struct {
	CourseID string `json:"course_id"`
	Reason   string `json:"reason"`
}
type QueryCoverage struct {
	Status       string         `json:"status"`
	Requested    int            `json:"requested_courses"`
	Succeeded    int            `json:"succeeded_courses"`
	Failures     []QueryFailure `json:"failures,omitempty"`
	FetchedFrom  time.Time      `json:"fetched_from"`
	FetchedUntil time.Time      `json:"fetched_until"`
}
type QuerySelection struct {
	Status       string `json:"status"`
	MatchedTotal *int   `json:"matched_total"`
	Observed     int    `json:"observed_matches,omitempty"`
}
type AgentItem struct {
	ID             string      `json:"id"`
	Kind           string      `json:"kind"`
	Title          string      `json:"title"`
	CourseID       string      `json:"course_id"`
	StartAt        *time.Time  `json:"start_at,omitempty"`
	EndAt          *time.Time  `json:"end_at,omitempty"`
	RegisteredAt   *time.Time  `json:"registered_at,omitempty"`
	Completion     *Completion `json:"completion,omitempty"`
	Attendable     bool        `json:"attendable,omitempty"`
	Availability   string      `json:"availability,omitempty"`
	Week           string      `json:"week,omitempty"`
	Submitted      string      `json:"submitted,omitempty"`
	Pinned         bool        `json:"pinned,omitempty"`
	HasAttachments bool        `json:"has_attachments,omitempty"`
}
type AppliedQuery struct {
	Intent         string       `json:"intent"`
	Term           string       `json:"term"`
	AsOf           time.Time    `json:"as_of"`
	CourseIDs      []string     `json:"course_ids,omitempty"`
	IncludeExpired *bool        `json:"include_expired,omitempty"`
	Period         *QueryWindow `json:"period,omitempty"`
}
type AgentResult struct {
	Sources        map[string]QueryCoverage `json:"sources,omitempty"`
	SearchedFields []string                 `json:"searched_fields,omitempty"`
	Query          AppliedQuery             `json:"query"`
	Coverage       QueryCoverage            `json:"coverage"`
	Selection      QuerySelection           `json:"selection"`
	Courses        map[string]string        `json:"courses"`
	Items          []AgentItem              `json:"items"`
}

func (r *AgentResult) finish(uncertain bool) {
	r.Coverage.FetchedUntil = time.Now().UTC()
	r.Coverage.Status = "complete"
	if len(r.Coverage.Failures) > 0 {
		r.Coverage.Status = "partial"
		uncertain = true
	}
	r.Selection.Status = "verified"
	n := len(r.Items)
	r.Selection.MatchedTotal = &n
	if uncertain {
		r.Selection.Status = "uncertain"
		r.Selection.MatchedTotal = nil
		r.Selection.Observed = n
	}
}

func (q AgentQuery) referenceTime() time.Time {
	if !q.asOf.IsZero() {
		return q.asOf
	}
	return time.Now().UTC()
}
