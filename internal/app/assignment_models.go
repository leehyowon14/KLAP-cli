package app

import (
	"github.com/leehyowon14/KLAP-cli/internal/klas"
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
}

func assignmentModel(value klas.Assignment) Assignment {
	return Assignment{
		OrdSeq:       value.OrdSeq,
		WeeklySeq:    value.WeeklySeq,
		WeeklySubSeq: value.WeeklySubSeq,
		Title:        value.Title,
		StartAt:      value.StartAt,
		DueAt:        value.DueAt,
		Submitted:    value.Submitted,
	}
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
}

func assignmentDetailModel(value klas.AssignmentDetail) AssignmentDetail {
	return AssignmentDetail{
		OrdSeq:         value.OrdSeq,
		Title:          value.Title,
		ContentText:    value.ContentText,
		StartAt:        value.StartAt,
		DueAt:          value.DueAt,
		Submitted:      value.Submitted,
		ReportType:     value.ReportType,
		SubmitFileType: value.SubmitFileType,
		FileLimitMB:    value.FileLimitMB,
		SubmittedTitle: value.SubmittedTitle,
		SubmittedText:  value.SubmittedText,
		FinalScore:     value.FinalScore,
		TutorText:      value.TutorText,
	}
}
