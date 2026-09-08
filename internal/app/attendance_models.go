package app

import "github.com/leehyowon14/KLAP-cli/internal/klas"

type AttendanceCourse struct {
	CourseCode  string
	SubjectID   string
	Name        string
	Professor   string
	CourseType  string
	Credits     string
	CreditHours string
	CurrentNum  string
	Weekday     string
}

func attendanceCourseModel(value klas.AttendanceCourse) AttendanceCourse {
	return AttendanceCourse{
		CourseCode:  value.CourseCode,
		SubjectID:   value.SubjectID,
		Name:        value.Name,
		Professor:   value.Professor,
		CourseType:  value.CourseType,
		Credits:     value.Credits,
		CreditHours: value.CreditHours,
		CurrentNum:  value.CurrentNum,
		Weekday:     value.Weekday,
	}
}

type AttendanceSession struct {
	Week  string
	Slots []AttendanceSlot
}

func attendanceSessionModel(value klas.AttendanceSession) AttendanceSession {
	return AttendanceSession{
		Week:  value.Week,
		Slots: attendanceSlotModels(value.Slots),
	}
}

func attendanceSessionModels(values []klas.AttendanceSession) []AttendanceSession {
	if values == nil {
		return nil
	}
	models := make([]AttendanceSession, len(values))
	for index, value := range values {
		models[index] = attendanceSessionModel(value)
	}
	return models
}

type AttendanceSlot struct {
	Index  int
	Status string
	Mark   string
	Date   string
}

func attendanceSlotModel(value klas.AttendanceSlot) AttendanceSlot {
	return AttendanceSlot{
		Index:  value.Index,
		Status: value.Status,
		Mark:   value.Mark,
		Date:   value.Date,
	}
}

func attendanceSlotModels(values []klas.AttendanceSlot) []AttendanceSlot {
	if values == nil {
		return nil
	}
	models := make([]AttendanceSlot, len(values))
	for index, value := range values {
		models[index] = attendanceSlotModel(value)
	}
	return models
}

type CdpAttendanceReport struct {
	TotalCount string
	Rows       []CdpAttendance
}

func cdpAttendanceReportModel(value klas.CdpAttendanceReport) CdpAttendanceReport {
	return CdpAttendanceReport{
		TotalCount: value.TotalCount,
		Rows:       cdpAttendanceModels(value.Rows),
	}
}

type CdpAttendance struct {
	Date    string
	Seq     string
	Title   string
	Speaker string
}

func cdpAttendanceModel(value klas.CdpAttendance) CdpAttendance {
	return CdpAttendance{
		Date:    value.Date,
		Seq:     value.Seq,
		Title:   value.Title,
		Speaker: value.Speaker,
	}
}

func cdpAttendanceModels(values []klas.CdpAttendance) []CdpAttendance {
	if values == nil {
		return nil
	}
	models := make([]CdpAttendance, len(values))
	for index, value := range values {
		models[index] = cdpAttendanceModel(value)
	}
	return models
}
