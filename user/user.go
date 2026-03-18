package klasuser

import (
	"KLAP/academic"
	klaslogin "KLAP/login"
	"fmt"
	"net/http"
)

// User 는 KLAS User의 정보를 저장하기 위한 타입입니다.
type User struct {
	id         string
	password   string
	cookies    []*http.Cookie
	isLoggedIn bool
	semesters  academic.SemesterSet
}

// ===Constructor===

// NewUser 는 새로운 KLAS User를 선언합니다.
func NewUser() *User {
	return &User{}
}

// ===Setter===

// SetID 는 사용자의 학번(ID)를 설정합니다.
func (u *User) SetID(id string) error {
	if u.isLoggedIn {
		return fmt.Errorf("유저의 ID가 이미 검증되었습니다.")
	} else if len(id) == 0 {
		return fmt.Errorf("잘못된 입력값입니다.")
	}
	u.id = id
	return nil
}

// SetPassword 는 사용자의 KLAS 비밀번호를 설정합니다.
func (u *User) SetPassword(password string) error {
	if u.isLoggedIn {
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

func (u *User) Login() error {
	defer func(u *User) {
		if len(u.cookies) == 0 {
			u.isLoggedIn = false
		}
	}(u)
	u.isLoggedIn = true
	cookies, err := klaslogin.Login(u.id, u.password)
	if err != nil {
		return fmt.Errorf("로그인에 실패하였습니다. %w", err)
	}
	u.cookies = cookies

	return nil
}

func (u *User) LoadSemesters() error {
	if len(u.cookies) == 0 {
		return fmt.Errorf("과목 및 학기 정보를 불러오는 데 필요한 인증 정보가 없습니다. 로그인을 먼저 수행해주세요.")
	}
	semesters, err := academic.Load(u.cookies)
	if err != nil {
		return err
	}
	u.semesters = *semesters

	return nil
}
