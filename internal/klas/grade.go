package klas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type GradeReport struct {
	Summary GradeSummary
	Terms   []GradeTerm
}

type GradeSummary struct {
	AppliedCredits        int
	MajorAppliedCredits   int
	CultureAppliedCredits int
	EtcAppliedCredits     int
	EarnedCredits         int
	MajorEarnedCredits    int
	CultureEarnedCredits  int
	EtcEarnedCredits      int
	DeletedCredits        int
	GPA                   string
	RetakeGPA             string
	Raw                   gradeSummaryItem `json:"-"`
}

type GradeTerm struct {
	Year    string
	Hakgi   string
	Label   string
	Courses []GradeCourse
	Raw     gradeTermItem `json:"-"`
}

type GradeCourse struct {
	Name          string
	CourseType    string
	Credits       int
	Grade         string
	CourseCode    string
	Department    string
	Finished      bool
	Retake        bool
	RetakeGrade   string
	GradePublic   bool
	TermCheckOpen bool
	TermFinished  bool
	Raw           gradeCourseItem `json:"-"`
}

type Rank struct {
	Year           string
	Hakgi          string
	TermValue      string
	TermLabel      string
	AppliedCredits string
	TotalScore     string
	GPA            string
	Percentile     string
	ClassRank      string
	ClassSize      string
	Warning        string
	Raw            rankItem `json:"-"`
}

type gradeSummaryItem struct {
	AppliedCredits        int     `json:"applyHakjum"`
	MajorAppliedCredits   int     `json:"majorApplyHakjum"`
	CultureAppliedCredits int     `json:"cultureApplyHakjum"`
	EtcAppliedCredits     int     `json:"etcApplyHakjum"`
	EarnedCredits         int     `json:"chidukHakjum"`
	MajorEarnedCredits    int     `json:"majorChidukHakjum"`
	CultureEarnedCredits  int     `json:"cultureChidukHakjum"`
	EtcEarnedCredits      int     `json:"etcChidukHakjum"`
	DeletedCredits        int     `json:"delHakjum"`
	GPA                   float64 `json:"hwakinScoresum"`
	RetakeGPA             float64 `json:"jaechulScoresum"`
}

type gradeTermItem struct {
	ThisYear    string            `json:"thisYear"`
	Hakgi       string            `json:"hakgi"`
	HakgiOrder  string            `json:"hakgiOrder"`
	CourseItems []gradeCourseItem `json:"sungjukList"`
}

type gradeCourseItem struct {
	Name          string `json:"gwamokKname"`
	CourseType    string `json:"codeName1"`
	Credits       int    `json:"hakjumNum"`
	Grade         string `json:"getGrade"`
	Department    string `json:"hakgwa"`
	CourseCode    string `json:"hakjungNo"`
	Finished      string `json:"finishOpt"`
	GradeOption   string `json:"sungjukOpt"`
	Retake        string `json:"retakeOpt"`
	RetakeGrade   string `json:"retakeGetGrade"`
	TermCheckOpen string `json:"termCheck"`
	TermFinished  string `json:"termFinish"`
}

type rankItem struct {
	ThisYear       string         `json:"thisYear"`
	Hakgi          string         `json:"hakgi"`
	AppliedCredits flexibleString `json:"applyHakjum"`
	TotalScore     flexibleString `json:"applySum"`
	GPA            flexibleString `json:"applyPoint"`
	ClassRank      flexibleString `json:"classOrder"`
	ClassSize      flexibleString `json:"manNum"`
	Warning        string         `json:"warningOpt"`
	Percentile     flexibleString `json:"pcnt"`
}

func (c *Client) Grades(ctx context.Context) (GradeReport, error) {
	summaryBody, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/AtnlcScreSungjukTot.do", map[string]any{})
	if err != nil {
		return GradeReport{}, err
	}
	var summaryItem gradeSummaryItem
	if err := json.Unmarshal(summaryBody, &summaryItem); err != nil {
		return GradeReport{}, fmt.Errorf("성적 요약 응답 파싱 실패: %w", err)
	}

	termsBody, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/AtnlcScreSungjukInfo.do", map[string]any{})
	if err != nil {
		return GradeReport{}, err
	}
	var termItems []gradeTermItem
	if err := json.Unmarshal(termsBody, &termItems); err != nil {
		return GradeReport{}, fmt.Errorf("성적 목록 응답 파싱 실패: %w", err)
	}

	terms := make([]GradeTerm, 0, len(termItems))
	for _, item := range termItems {
		term := buildGradeTerm(item)
		if len(term.Courses) == 0 {
			continue
		}
		terms = append(terms, term)
	}

	return GradeReport{
		Summary: buildGradeSummary(summaryItem),
		Terms:   terms,
	}, nil
}

func (c *Client) Ranks(ctx context.Context) ([]Rank, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/StandStdList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var response []rankItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("석차 조회 응답 파싱 실패: %w", err)
	}

	ranks := make([]Rank, 0, len(response))
	for _, item := range response {
		rank := buildRank(item)
		if strings.TrimSpace(rank.TermValue) == "" {
			continue
		}
		ranks = append(ranks, rank)
	}
	return ranks, nil
}

func buildGradeSummary(item gradeSummaryItem) GradeSummary {
	return GradeSummary{
		AppliedCredits:        item.AppliedCredits,
		MajorAppliedCredits:   item.MajorAppliedCredits,
		CultureAppliedCredits: item.CultureAppliedCredits,
		EtcAppliedCredits:     item.EtcAppliedCredits,
		EarnedCredits:         item.EarnedCredits,
		MajorEarnedCredits:    item.MajorEarnedCredits,
		CultureEarnedCredits:  item.CultureEarnedCredits,
		EtcEarnedCredits:      item.EtcEarnedCredits,
		DeletedCredits:        item.DeletedCredits,
		GPA:                   formatGradeNumber(item.GPA),
		RetakeGPA:             formatGradeNumber(item.RetakeGPA),
		Raw:                   item,
	}
}

func buildGradeTerm(item gradeTermItem) GradeTerm {
	courses := make([]GradeCourse, 0, len(item.CourseItems))
	for _, courseItem := range item.CourseItems {
		name := strings.TrimSpace(courseItem.Name)
		if name == "" {
			continue
		}
		courses = append(courses, GradeCourse{
			Name:          name,
			CourseType:    strings.TrimSpace(courseItem.CourseType),
			Credits:       courseItem.Credits,
			Grade:         strings.TrimSpace(courseItem.Grade),
			CourseCode:    strings.TrimSpace(courseItem.CourseCode),
			Department:    strings.TrimSpace(courseItem.Department),
			Finished:      strings.EqualFold(strings.TrimSpace(courseItem.Finished), "Y"),
			Retake:        strings.EqualFold(strings.TrimSpace(courseItem.Retake), "Y"),
			RetakeGrade:   strings.TrimSpace(courseItem.RetakeGrade),
			GradePublic:   strings.TrimSpace(courseItem.Grade) != "",
			TermCheckOpen: strings.EqualFold(strings.TrimSpace(courseItem.TermCheckOpen), "Y"),
			TermFinished:  strings.EqualFold(strings.TrimSpace(courseItem.TermFinished), "Y"),
			Raw:           courseItem,
		})
	}
	return GradeTerm{
		Year:    strings.TrimSpace(item.ThisYear),
		Hakgi:   strings.TrimSpace(item.Hakgi),
		Label:   gradeTermLabel(item),
		Courses: courses,
		Raw:     item,
	}
}

func gradeTermLabel(item gradeTermItem) string {
	year := strings.TrimSpace(item.ThisYear)
	hakgi := strings.TrimSpace(item.Hakgi)
	order := strings.TrimSpace(item.HakgiOrder)
	switch hakgi {
	case "1", "2":
		if order == "" {
			order = hakgi
		}
		if year == "" {
			return order + "학기"
		}
		return year + "년도 " + order + "학기"
	case "3":
		if year == "" {
			return "여름학기"
		}
		return year + "년도 여름학기"
	case "4":
		if year == "" {
			return "겨울학기"
		}
		return year + "년도 겨울학기"
	default:
		if order != "" {
			if year == "" {
				return order
			}
			return year + "년도 " + order
		}
		return strings.TrimSpace(year + " " + hakgi)
	}
}

func formatGradeNumber(value float64) string {
	if value == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func buildRank(item rankItem) Rank {
	year := strings.TrimSpace(item.ThisYear)
	hakgi := strings.TrimSpace(item.Hakgi)
	termValue := ""
	if year != "" && hakgi != "" {
		termValue = year + "," + hakgi
	}
	return Rank{
		Year:           year,
		Hakgi:          hakgi,
		TermValue:      termValue,
		TermLabel:      rankTermLabel(year, hakgi),
		AppliedCredits: item.AppliedCredits.String(),
		TotalScore:     item.TotalScore.String(),
		GPA:            item.GPA.String(),
		Percentile:     item.Percentile.String(),
		ClassRank:      item.ClassRank.String(),
		ClassSize:      item.ClassSize.String(),
		Warning:        strings.TrimSpace(item.Warning),
		Raw:            item,
	}
}

func rankTermLabel(year string, hakgi string) string {
	if year == "" {
		return strings.TrimSpace(hakgi)
	}
	switch strings.TrimSpace(hakgi) {
	case "1":
		return year + "년도 1학기"
	case "2":
		return year + "년도 2학기"
	case "3":
		return year + "년도 여름학기"
	case "4":
		return year + "년도 겨울학기"
	default:
		return strings.TrimSpace(year + " " + hakgi)
	}
}
