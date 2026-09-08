package klas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Assignment struct {
	OrdSeq       string
	WeeklySeq    string
	WeeklySubSeq string
	Title        string
	StartAt      *time.Time
	DueAt        *time.Time
	Submitted    bool
	Raw          assignmentListItem `json:"-"`
}

type AssignmentDetail struct {
	OrdSeq         string
	Title          string
	ContentText    string
	StartAt        *time.Time
	DueAt          *time.Time
	Submitted      bool
	ReportType     string
	SubmitFileType string
	FileLimitMB    string
	SubmittedTitle string
	SubmittedText  string
	FinalScore     string
	TutorText      string
	Raw            assignmentDetailResponse `json:"-"`
}

type assignmentListItem struct {
	OrdSeq       flexibleString `json:"ordseq"`
	WeeklySeq    flexibleString `json:"weeklyseq"`
	WeeklySubSeq flexibleString `json:"weeklysubseq"`
	Title        string         `json:"title"`
	StartDate    string         `json:"startdate"`
	ExpireDate   string         `json:"expiredate"`
	SubmitYN     string         `json:"submityn"`
}

type assignmentDetailResponse struct {
	Report     assignmentReport     `json:"rpt"`
	Submission assignmentSubmission `json:"smt"`
}

type assignmentReport struct {
	OrdSeq         flexibleString `json:"ordseq"`
	Title          string         `json:"title"`
	Contents       string         `json:"contents"`
	StartDate      string         `json:"startdate"`
	ExpireDate     string         `json:"expiredate"`
	ReportType     string         `json:"reptype"`
	SubmitYN       string         `json:"submityn"`
	SubmitFileType string         `json:"submitfiletype"`
	FileLimit      flexibleString `json:"filelimit"`
}

type assignmentSubmission struct {
	Title         string      `json:"title"`
	Contents      string      `json:"contents"`
	FinalScore    interface{} `json:"finalscore"`
	TutorContents string      `json:"tutorcontents"`
}

func (c *Client) Assignments(ctx context.Context, yearHakgi string, course Course) ([]Assignment, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return nil, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/TaskStdList.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"currentPage":     0,
	})
	if err != nil {
		return nil, err
	}

	var response []assignmentListItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("과제 목록 응답 파싱 실패: %w", err)
	}

	assignments := make([]Assignment, 0, len(response))
	for _, item := range response {
		ordSeq := item.OrdSeq.String()
		if strings.TrimSpace(ordSeq) == "" {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "제목 없음"
		}
		assignments = append(assignments, Assignment{
			OrdSeq:       ordSeq,
			WeeklySeq:    item.WeeklySeq.String(),
			WeeklySubSeq: item.WeeklySubSeq.String(),
			Title:        title,
			StartAt:      parseKoreanDateTime(item.StartDate),
			DueAt:        parseKoreanDateTime(item.ExpireDate),
			Submitted:    strings.EqualFold(item.SubmitYN, "Y"),
			Raw:          item,
		})
	}
	return assignments, nil
}

func (c *Client) AssignmentDetail(ctx context.Context, yearHakgi string, course Course, ordSeq string) (AssignmentDetail, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return AssignmentDetail{}, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/TaskStdView.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"ordseq":          ordSeq,
	})
	if err != nil {
		return AssignmentDetail{}, err
	}

	var response assignmentDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return AssignmentDetail{}, fmt.Errorf("과제 상세 응답 파싱 실패: %w", err)
	}
	if strings.TrimSpace(response.Report.Title) == "" && strings.TrimSpace(response.Report.Contents) == "" {
		return AssignmentDetail{}, errors.New("과제 상세 응답에 rpt 본문이 없습니다")
	}

	return AssignmentDetail{
		OrdSeq:         firstNonEmpty(response.Report.OrdSeq.String(), ordSeq),
		Title:          firstNonEmpty(strings.TrimSpace(response.Report.Title), "제목 없음"),
		ContentText:    htmlToText(response.Report.Contents),
		StartAt:        parseKoreanDateTime(response.Report.StartDate),
		DueAt:          parseKoreanDateTime(response.Report.ExpireDate),
		Submitted:      strings.EqualFold(response.Report.SubmitYN, "Y"),
		ReportType:     reportTypeLabel(response.Report.ReportType),
		SubmitFileType: response.Report.SubmitFileType,
		FileLimitMB:    response.Report.FileLimit.String(),
		SubmittedTitle: response.Submission.Title,
		SubmittedText:  htmlToText(response.Submission.Contents),
		FinalScore:     fmt.Sprint(response.Submission.FinalScore),
		TutorText:      htmlToText(response.Submission.TutorContents),
		Raw:            response,
	}, nil
}

func reportTypeLabel(value string) string {
	switch value {
	case "1", "P":
		return "개인"
	case "2":
		return "팀"
	default:
		return value
	}
}
