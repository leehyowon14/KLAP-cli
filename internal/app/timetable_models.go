package app

import "github.com/leehyowon14/KLAP-cli/internal/klas"

type TimetableEntry struct {
	SubjectID   string
	SubjectName string
	Weekday     int
	Period      int
	Span        int
	Room        string
	Professor   string
	Online      bool
}

func timetableEntryModels(entries []klas.TimetableEntry) []TimetableEntry {
	if entries == nil {
		return nil
	}
	models := make([]TimetableEntry, len(entries))
	for index, entry := range entries {
		models[index] = TimetableEntry{
			SubjectID: entry.SubjectID, SubjectName: entry.SubjectName,
			Weekday: entry.Weekday, Period: entry.Period, Span: entry.Span,
			Room: entry.Room, Professor: entry.Professor, Online: entry.Online,
		}
	}
	return models
}
