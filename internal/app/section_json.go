package app

import "encoding/json"

// Preserve partial failure messages for JSON clients instead of encoding error
// interfaces as empty objects. Existing CLI callers still receive the error.
func (e DashboardSectionError) MarshalJSON() ([]byte, error) {
	message := ""
	if e.Err != nil {
		message = e.Err.Error()
	}
	return json.Marshal(struct {
		Section string
		Err     string
	}{e.Section, message})
}
