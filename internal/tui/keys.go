package tui

func keyMatches(value string, keys ...string) bool {
	for _, key := range keys {
		if value == key {
			return true
		}
	}
	return false
}
