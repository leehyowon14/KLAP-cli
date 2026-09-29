package app

import (
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"time"
)

type Lecture struct {
	FirstStartedAt   *time.Time
	FirstCompletedAt *time.Time
	ContentID        string
	PlayURL          string
	LearningSeq      string
	FileID           string
	WeekNo           string
	WeeklySeq        string
	ModuleTitle      string
	Title            string
	Progress         string
	AchievedTime     string
	RequiredTime     string
	StartAt          *time.Time
	EndAt            *time.Time
}

type LectureProgress struct {
	TotalTime string
	PTime     string
	Progress  float64
	Completed bool
}

func lectureModel(value klas.Lecture) Lecture {
	return Lecture{
		FirstStartedAt:   value.FirstStartedAt,
		FirstCompletedAt: value.FirstCompletedAt,
		ContentID:        value.ContentID,
		PlayURL:          value.PlayURL,
		LearningSeq:      value.LearningSeq,
		FileID:           value.FileID,
		WeekNo:           value.WeekNo,
		WeeklySeq:        value.WeeklySeq,
		ModuleTitle:      value.ModuleTitle,
		Title:            value.Title,
		Progress:         value.Progress,
		AchievedTime:     value.AchievedTime,
		RequiredTime:     value.RequiredTime,
		StartAt:          value.StartAt,
		EndAt:            value.EndAt,
	}
}
func lectureModels(values []klas.Lecture) []Lecture {
	if values == nil {
		return nil
	}
	result := make([]Lecture, len(values))
	for i, v := range values {
		result[i] = lectureModel(v)
	}
	return result
}
func lectureProgressModel(value klas.LectureProgress) LectureProgress {
	return LectureProgress{
		TotalTime: value.TotalTime,
		PTime:     value.PTime,
		Progress:  value.Progress,
		Completed: value.Completed,
	}
}
