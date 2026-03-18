package klasuser

import (
	"fmt"
	"net/http"
)

// User 는 KLAS User의 정보를 저장하기 위한 타입입니다.
type User struct {
	id              string
	password        string
	cookies         []*http.Cookie
	isIDValid       bool
	isPasswordValid bool
}

// ===Constructor===

// NewUser 는 새로운 KLAS User를 선언합니다.
func NewUser() *User {
	return &User{}
}

// ===Setter===

// SetID 는 사용자의 학번(ID)를 설정합니다.
func (u *User) SetID(id string) error {
	if u.isIDValid {
		return fmt.Errorf("유저의 ID가 이미 검증되었습니다.")
	} else if len(id) == 0 {
		return fmt.Errorf("잘못된 입력값입니다.")
	}
	u.id = id
	return nil
}

// SetPassword 는 사용자의 KLAS 비밀번호를 설정합니다.
func (u *User) SetPassword(password string) error {
	if u.isPasswordValid {
		return fmt.Errorf("유저의 비밀번호가 이미 검증되었습니다.")
	} else if len(password) == 0 {
		return fmt.Errorf("잘못된 입력값입니다.")
	}
	u.password = password
	return nil
}

// ===Getter===

// GetID returns User's ID(hakbun)
func (u User) GetID() string {
	return u.id
}

// GetCookies returns User's Auth Cookies
func (u User) GetCookies() []*http.Cookie {
	return u.cookies
}

// ===Method===
