package klaslogin

import (
	httpclient "KLAP/http"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
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

func makeLoginToken(id string, password string, rsaPublicKey string) (loginToken string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(rsaPublicKey)
	if err != nil {
		return "", fmt.Errorf("RSA 암호화 키 디코딩을 실패하였습니다: %w", err)
	}
	pubKey, err := x509.ParsePKCS1PublicKey(decoded)
	if err != nil {
		return "", fmt.Errorf("RSA 암호화 키 파싱을 실패하였습니다: %w", err)
	}

	payload := map[string]string{
		"loginId":   id,
		"loginPwd":  password,
		"storeIdYn": "N",
	}

	body, _ := json.Marshal(payload)
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, body)
	if err != nil {
		return "", fmt.Errorf("RSA 암호화 실패: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
