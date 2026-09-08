package cli

import (
	"errors"
)

func userFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "--user" {
			continue
		}
		if i+1 >= len(args) {
			return ""
		}
		return args[i+1]
	}
	return ""
}

func courseFilter(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--course" {
			continue
		}
		if i+1 >= len(args) {
			return "", errors.New("--course에는 과목명 또는 course list 번호가 필요합니다")
		}
		return args[i+1], nil
	}
	return "", nil
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}
