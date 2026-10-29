package app

import "errors"

// ValidateStableLectureIDs rejects positional IDs before work is persisted.
// Stable IDs bind queued actions to a term and course even if lists reorder.
func ValidateStableLectureIDs(ids []string) error {
	if len(ids) == 0 {
		return errors.New("select at least one lecture")
	}
	for _, id := range ids {
		_, _, stable, err := parseStableCourseResourceID("lecture", id, 1)
		if err != nil {
			return err
		}
		if !stable {
			return errors.New("queued attendance requires stable lecture IDs")
		}
	}
	_, err := stableLectureTermValue(ids)
	return err
}
