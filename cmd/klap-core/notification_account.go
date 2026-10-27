package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func notificationAccount(rows []app.UserRow) (app.UserOption, string, error) {
	for _, row := range rows {
		if row.Current && row.User.StudentID != "" {
			return app.UserOption{StudentID: row.User.StudentID}, fmt.Sprintf("%x", sha256.Sum256([]byte(row.User.StudentID))), nil
		}
	}
	return app.UserOption{}, "", fmt.Errorf("현재 로그인 계정을 확인하지 못했습니다")
}
