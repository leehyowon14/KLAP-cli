package transcript

type Job struct {
	InputPath         string   `json:"inputPath"`
	OutputPath        string   `json:"outputPath"`
	Locale            string   `json:"locale,omitempty"`
	ContextualStrings []string `json:"contextualStrings,omitempty"`
}

type Request struct {
	Jobs     []Job `json:"jobs"`
	Progress bool  `json:"progress,omitempty"`
}

type Result struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	Text       string `json:"text"`
	Err        string `json:"error"`
}

type Response struct {
	Results []Result `json:"results"`
}

type Progress struct {
	InputPath  string
	OutputPath string
	Progress   float64
	Err        string
}
