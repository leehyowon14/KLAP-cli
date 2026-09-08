package app

import "github.com/leehyowon14/KLAP-cli/internal/klas"

type EvaluationTerm struct {
	Year        string
	Hakgi       string
	Value       string
	Label       string
	JudgeChasu  string
	JudgeName   string
	FromDate    string
	ToDate      string
	TermEnabled bool
}

func evaluationTermModel(value klas.EvaluationTerm) EvaluationTerm {
	return EvaluationTerm{
		Year:        value.Year,
		Hakgi:       value.Hakgi,
		Value:       value.Value,
		Label:       value.Label,
		JudgeChasu:  value.JudgeChasu,
		JudgeName:   value.JudgeName,
		FromDate:    value.FromDate,
		ToDate:      value.ToDate,
		TermEnabled: value.TermEnabled,
	}
}

type EvaluationCourse struct {
	Name            string
	Professor       string
	CourseType      string
	EngineeringType string
	Credits         string
	Evaluated       bool
	Engineering     bool
	ThisYear        string
	Hakgi           string
	OpenMajorCode   string
	OpenGrade       string
	OpenGwamokNo    string
	BunbanNo        string
}

func evaluationCourseModel(value klas.EvaluationCourse) EvaluationCourse {
	return EvaluationCourse{
		Name:            value.Name,
		Professor:       value.Professor,
		CourseType:      value.CourseType,
		EngineeringType: value.EngineeringType,
		Credits:         value.Credits,
		Evaluated:       value.Evaluated,
		Engineering:     value.Engineering,
		ThisYear:        value.ThisYear,
		Hakgi:           value.Hakgi,
		OpenMajorCode:   value.OpenMajorCode,
		OpenGrade:       value.OpenGrade,
		OpenGwamokNo:    value.OpenGwamokNo,
		BunbanNo:        value.BunbanNo,
	}
}
