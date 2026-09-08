package app

import (
	"time"

	"github.com/leehyowon14/KLAP-cli/internal/account"
)

type User struct {
	StudentID string    `json:"studentId"`
	SavedAt   time.Time `json:"savedAt"`
	UserID    string    `json:"userId,omitempty"`
}

func userModel(user account.User) User {
	return User{StudentID: user.StudentID, SavedAt: user.SavedAt, UserID: user.UserID}
}
