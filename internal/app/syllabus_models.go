package app

import (
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"slices"
)

type Syllabus struct {
	SubjectID      string
	CourseCode     string
	KoreanName     string
	EnglishName    string
	FullName       string
	CourseType     string
	Credits        string
	CurrentNum     string
	Professor      string
	ProfessorTitle string
	Summary        string
	Purpose        string
	Outcome        string
	Competency     string
	BookName       string
	Operation      string
	Evaluation     SyllabusEvaluation
	Schedule       []SyllabusWeek
	Times          []SyllabusTime
}

func syllabusModel(value klas.Syllabus) Syllabus {
	return Syllabus{
		SubjectID:      value.SubjectID,
		CourseCode:     value.CourseCode,
		KoreanName:     value.KoreanName,
		EnglishName:    value.EnglishName,
		FullName:       value.FullName,
		CourseType:     value.CourseType,
		Credits:        value.Credits,
		CurrentNum:     value.CurrentNum,
		Professor:      value.Professor,
		ProfessorTitle: value.ProfessorTitle,
		Summary:        value.Summary,
		Purpose:        value.Purpose,
		Outcome:        value.Outcome,
		Competency:     value.Competency,
		BookName:       value.BookName,
		Operation:      value.Operation,
		Evaluation:     syllabusEvaluationModel(value.Evaluation),
		Schedule:       syllabusWeekModels(value.Schedule),
		Times:          syllabusTimeModels(value.Times),
	}
}

type SyllabusEvaluation struct {
	Attendance int
	Learning   int
	Midterm    int
	Final      int
	Report     int
	Quiz       int
	Other      int
}

func syllabusEvaluationModel(value klas.SyllabusEvaluation) SyllabusEvaluation {
	return SyllabusEvaluation{
		Attendance: value.Attendance,
		Learning:   value.Learning,
		Midterm:    value.Midterm,
		Final:      value.Final,
		Report:     value.Report,
		Quiz:       value.Quiz,
		Other:      value.Other,
	}
}

type SyllabusWeek struct {
	Week    int
	Topic   string
	SubNote string
}

func syllabusWeekModel(value klas.SyllabusWeek) SyllabusWeek {
	return SyllabusWeek{
		Week:    value.Week,
		Topic:   value.Topic,
		SubNote: value.SubNote,
	}
}

func syllabusWeekModels(values []klas.SyllabusWeek) []SyllabusWeek {
	if values == nil {
		return nil
	}
	models := make([]SyllabusWeek, len(values))
	for index, value := range values {
		models[index] = syllabusWeekModel(value)
	}
	return models
}

type SyllabusTime struct {
	Weekday string
	Periods []int
	Room    string
}

func syllabusTimeModel(value klas.SyllabusTime) SyllabusTime {
	return SyllabusTime{
		Weekday: value.Weekday,
		Periods: slices.Clone(value.Periods),
		Room:    value.Room,
	}
}

func syllabusTimeModels(values []klas.SyllabusTime) []SyllabusTime {
	if values == nil {
		return nil
	}
	models := make([]SyllabusTime, len(values))
	for index, value := range values {
		models[index] = syllabusTimeModel(value)
	}
	return models
}
