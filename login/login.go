package klaslogin

import (
	httpclient "KLAP/http"
	"encoding/json"
	"fmt"
	"net/http"
)

type rsaPublicKey struct {
	PublicKey string `json:"publicKey"`
}

func getRSAKey() (publicKey string, cookies []*http.Cookie, err error) {
	req, err := http.NewRequest("POST", "https://klas.kw.ac.kr/usr/cmn/login/LoginSecurity.do", nil)
	if err != nil {
		return "", nil, err
	}
	res, err := httpclient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	var data rsaPublicKey
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return "", nil, fmt.Errorf("로그인 정보 암호화에 필요한 키 값을 불러오지 못했습니다: %w", err)
	} else if len(data.PublicKey) == 0 {
		return "", nil, nil
	}

	// return values
	publicKey = data.PublicKey
	rawCookies := res.Cookies()
	for _, cookie := range rawCookies {
		if cookie.Name == "SESSION" || cookie.Name == "WMONID" {
			cookies = append(cookies, cookie)
		}
	}

	if len(cookies) != 2 {
		return "", nil, fmt.Errorf("로그인에 필요한 인증 정보를 받아오지 못했습니다.")
	}

	return publicKey, cookies, nil
}
