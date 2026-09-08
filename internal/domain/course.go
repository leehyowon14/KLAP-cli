// Package domain contains shared, normalized values without I/O dependencies.
package domain

// JSON names retain the established cache representation, not a transport DTO.
type Term struct {
	Label   string   `json:"label"`
	Value   string   `json:"value"`
	Courses []Course `json:"subjList"`
}

type Course struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
