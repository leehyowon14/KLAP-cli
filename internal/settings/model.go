package settings

type Settings struct {
	Reminder   Reminder   `json:"reminder"`
	Calendar   Calendar   `json:"calendar"`
	Term       Term       `json:"term"`
	Download   Download   `json:"download"`
	Transcript Transcript `json:"transcript"`
}

type Reminder struct {
	ListName        string `json:"listName"`
	UseExistingList bool   `json:"useExistingList"`
	AlarmBeforeMin  int    `json:"alarmBeforeMin"`
}

type Calendar struct {
	Name                     string `json:"name"`
	UseExistingList          bool   `json:"useExistingList"`
	AcademicName             string `json:"academicName"`
	AcademicUseExistingList  bool   `json:"academicUseExistingList"`
	TimetableName            string `json:"timetableName"`
	TimetableUseExistingList bool   `json:"timetableUseExistingList"`
}

type Term struct {
	Value string `json:"value"`
}

type Download struct {
	Dir         string `json:"dir"`
	Concurrency int    `json:"concurrency"`
	Caffeinate  *bool  `json:"caffeinate"`
	KeepPartial bool   `json:"keepPartial"`
}

type Transcript struct {
	Concurrency int `json:"concurrency"`
}
