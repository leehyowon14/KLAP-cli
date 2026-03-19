package academic

type course struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Label string `json:"subj"`
}

func (c course) GetLessons()
