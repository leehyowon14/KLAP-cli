package domain

import "time"

// Session is the persisted authentication snapshot shared by authentication and
// credential storage. JSON names retain the established Keyring schema.
type Session struct {
	UserID    string            `json:"userId"`
	Cookies   map[string]string `json:"cookies"`
	CreatedAt time.Time         `json:"createdAt"`
}
