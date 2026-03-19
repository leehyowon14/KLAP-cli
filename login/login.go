package klaslogin

import (
	httpclient "KLAP/http"
	utils "KLAP/util"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// Login 은 주어진 학번과 비밀번호로 로그인을 수행합니다.
// Returns: KLAS Auth Cookies, error
func Login(id string, password string) (cookies []*http.Cookie, err error) {
	publicKey, cookies, err := getRSAKey()
	if err != nil {
		return nil, err
	}
	loginToken, err := makeLoginToken(id, password, publicKey)
	if err != nil {
		return nil, fmt.Errorf("로그인 정보 암호화에 실패하였습니다: %w", err)
	}

	payload := map[string]string{
		"loginToken":     loginToken,
		"redirectUrl":    "",
		"redirectTabUrl": "",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("로그인 요청 본문 구축에 실패하였습니다: %w", err)
	}

	req, _ := http.NewRequest("POST", "https://klas.kw.ac.kr/usr/cmn/login/LoginConfirm.do", bytes.NewBuffer(body))
	utils.SetJsonRequestHeaders(req, cookies)

	res, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("로그인 요청이 실패하였습니다: %w", err)
	}
	defer res.Body.Close()

	//TODO: 로그인 검증 로직
	/*
			{
		    "redirectUrl": "",
		    "fieldErrors": [
		        {
		            "field": "",
		            "message": "LOGIN ERROR: 개인번호 또는 비밀번호가 일치하지 않습니다.\n로그인 실패 건수 1"
		        }
		    ],
		    "responseText": "",
		    "response": {},
		    "errorCount": 1,
		    "redirect": false,
		    "loginRequired": false
			}
	*/

	return cookies, nil
}

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
	pubAny, err := x509.ParsePKIXPublicKey(decoded)
	if err != nil {
		return "", fmt.Errorf("RSA 암호화 키 파싱을 실패하였습니다: %w", err)
	}
	pubKey, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("RSA 공개키 형식이 올바르지 않습니다")
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
