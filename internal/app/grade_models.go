package app

import "github.com/leehyowon14/KLAP-cli/internal/klas"

type GradeReport struct {
	Summary GradeSummary
	Terms   []GradeTerm
}

func gradeReportModel(value klas.GradeReport) GradeReport {
	return GradeReport{
		Summary: gradeSummaryModel(value.Summary),
		Terms:   gradeTermModels(value.Terms),
	}
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
}

func gradeSummaryModel(value klas.GradeSummary) GradeSummary {
	return GradeSummary{
		AppliedCredits:        value.AppliedCredits,
		MajorAppliedCredits:   value.MajorAppliedCredits,
		CultureAppliedCredits: value.CultureAppliedCredits,
		EtcAppliedCredits:     value.EtcAppliedCredits,
		EarnedCredits:         value.EarnedCredits,
		MajorEarnedCredits:    value.MajorEarnedCredits,
		CultureEarnedCredits:  value.CultureEarnedCredits,
		EtcEarnedCredits:      value.EtcEarnedCredits,
		DeletedCredits:        value.DeletedCredits,
		GPA:                   value.GPA,
		RetakeGPA:             value.RetakeGPA,
	}
}

type GradeTerm struct {
	Year    string
	Hakgi   string
	Label   string
	Courses []GradeCourse
}

func gradeTermModel(value klas.GradeTerm) GradeTerm {
	return GradeTerm{
		Year:    value.Year,
		Hakgi:   value.Hakgi,
		Label:   value.Label,
		Courses: gradeCourseModels(value.Courses),
	}
}

func gradeTermModels(values []klas.GradeTerm) []GradeTerm {
	if values == nil {
		return nil
	}
	models := make([]GradeTerm, len(values))
	for index, value := range values {
		models[index] = gradeTermModel(value)
	}
	return models
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
}

func gradeCourseModel(value klas.GradeCourse) GradeCourse {
	return GradeCourse{
		Name:          value.Name,
		CourseType:    value.CourseType,
		Credits:       value.Credits,
		Grade:         value.Grade,
		CourseCode:    value.CourseCode,
		Department:    value.Department,
		Finished:      value.Finished,
		Retake:        value.Retake,
		RetakeGrade:   value.RetakeGrade,
		GradePublic:   value.GradePublic,
		TermCheckOpen: value.TermCheckOpen,
		TermFinished:  value.TermFinished,
	}
}

func gradeCourseModels(values []klas.GradeCourse) []GradeCourse {
	if values == nil {
		return nil
	}
	models := make([]GradeCourse, len(values))
	for index, value := range values {
		models[index] = gradeCourseModel(value)
	}
	return models
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
}

func rankModel(value klas.Rank) Rank {
	return Rank{
		Year:           value.Year,
		Hakgi:          value.Hakgi,
		TermValue:      value.TermValue,
		TermLabel:      value.TermLabel,
		AppliedCredits: value.AppliedCredits,
		TotalScore:     value.TotalScore,
		GPA:            value.GPA,
		Percentile:     value.Percentile,
		ClassRank:      value.ClassRank,
		ClassSize:      value.ClassSize,
		Warning:        value.Warning,
	}
}

func rankModels(values []klas.Rank) []Rank {
	if values == nil {
		return nil
	}
	models := make([]Rank, len(values))
	for index, value := range values {
		models[index] = rankModel(value)
	}
	return models
}
