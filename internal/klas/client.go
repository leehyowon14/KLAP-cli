package klas

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://klas.kw.ac.kr"

var ErrSessionExpired = errors.New("KLAS 세션이 만료되었습니다")

type Client struct {
	httpClient *http.Client
	jar        http.CookieJar
	baseURL    *url.URL
}

type Session struct {
	UserID    string            `json:"userId"`
	Cookies   map[string]string `json:"cookies"`
	CreatedAt time.Time         `json:"createdAt"`
}

type Term struct {
	Label   string   `json:"label"`
	Value   string   `json:"value"`
	Courses []Course `json:"subjList"`
}

type Course struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type commonErrorResponse struct {
	Redirect      bool         `json:"redirect"`
	RedirectURL   string       `json:"redirectUrl"`
	FieldErrors   []fieldError `json:"fieldErrors"`
	ResponseText  string       `json:"responseText"`
	LoginRequired bool         `json:"loginRequired"`
	ErrorCount    int          `json:"errorCount"`
}

type loginSecurityResponse struct {
	PublicKey string `json:"publicKey"`
}

type loginConfirmResponse struct {
	FieldErrors   []fieldError `json:"fieldErrors"`
	Response      *loginUser   `json:"response"`
	ErrorCount    int          `json:"errorCount"`
	LoginRequired bool         `json:"loginRequired"`
}

type loginUser struct {
	UserID string `json:"userId"`
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar 생성 실패: %w", err)
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
		jar:     jar,
		baseURL: parsedBaseURL,
	}, nil
}

func (c *Client) Login(ctx context.Context, studentID string, password string) (Session, error) {
	publicKey, err := c.loginSecurity(ctx)
	if err != nil {
		return Session{}, err
	}

	if !c.hasCookie("SESSION") && !c.hasCookie("WMONID") {
		return Session{}, errors.New("LoginSecurity 응답에 세션 쿠키가 없습니다")
	}

	loginToken, err := buildLoginToken(publicKey, studentID, password)
	if err != nil {
		return Session{}, err
	}

	userID, err := c.loginConfirm(ctx, loginToken)
	if err != nil {
		return Session{}, err
	}

	cookies := c.cookiesMap()
	if cookies["SESSION"] == "" && cookies["WMONID"] == "" {
		return Session{}, errors.New("LoginConfirm 이후 저장할 세션 쿠키가 없습니다")
	}

	return Session{
		UserID:    userID,
		Cookies:   cookies,
		CreatedAt: time.Now(),
	}, nil
}

func (c *Client) SetSession(session Session) {
	var cookies []*http.Cookie
	for name, value := range session.Cookies {
		if strings.TrimSpace(value) == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{Name: name, Value: value, Path: "/"})
	}
	c.jar.SetCookies(c.baseURL, cookies)
}

func (c *Client) Courses(ctx context.Context) ([]Term, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cmn/frame/YearhakgiAtnlcSbjectList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var terms []Term
	if err := json.Unmarshal(body, &terms); err != nil {
		return nil, fmt.Errorf("수업 목록 응답 파싱 실패: %w", err)
	}

	for termIndex := range terms {
		filtered := terms[termIndex].Courses[:0]
		for _, course := range terms[termIndex].Courses {
			if strings.TrimSpace(course.Name) == "" || strings.TrimSpace(course.Value) == "" {
				continue
			}
			filtered = append(filtered, course)
		}
		terms[termIndex].Courses = filtered
	}

	return terms, nil
}

func (c *Client) loginSecurity(ctx context.Context) (string, error) {
	body, err := c.do(ctx, http.MethodPost, "/usr/cmn/login/LoginSecurity.do", nil)
	if err != nil {
		return "", err
	}

	var response loginSecurityResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("LoginSecurity 응답 파싱 실패: %w", err)
	}
	if strings.TrimSpace(response.PublicKey) == "" {
		return "", errors.New("LoginSecurity 응답에 publicKey가 없습니다")
	}
	return response.PublicKey, nil
}

func (c *Client) loginConfirm(ctx context.Context, loginToken string) (string, error) {
	requestBody := map[string]string{
		"loginToken":     loginToken,
		"redirectUrl":    "",
		"redirectTabUrl": "",
	}

	body, err := c.do(ctx, http.MethodPost, "/usr/cmn/login/LoginConfirm.do", requestBody)
	if err != nil {
		return "", err
	}

	var response loginConfirmResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("LoginConfirm 응답 파싱 실패: %w", err)
	}

	if response.LoginRequired {
		return "", errors.New("로그인이 필요하다는 응답을 받았습니다")
	}
	if response.ErrorCount > 0 {
		return "", errors.New(firstFieldError(response.FieldErrors, "KLAS 로그인에 실패했습니다"))
	}
	if response.Response == nil || strings.TrimSpace(response.Response.UserID) == "" {
		return "", errors.New("LoginConfirm 성공 응답에 userId가 없습니다")
	}

	return response.Response.UserID, nil
}

func (c *Client) do(ctx context.Context, method string, path string, payload any) ([]byte, error) {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})

	var requestBody io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("요청 직렬화 실패: %w", err)
		}
		requestBody = bytes.NewReader(payloadBytes)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json;charset=utf-8")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("KLAS 요청 실패: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("KLAS HTTP 오류: %s", response.Status)
	}
	if looksLikeLoginHTML(body) {
		return nil, ErrSessionExpired
	}
	if err := checkCommonAPIError(body); err != nil {
		return nil, err
	}

	return body, nil
}

func checkCommonAPIError(body []byte) error {
	var response commonErrorResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil
	}

	if response.LoginRequired {
		return ErrSessionExpired
	}
	if response.ErrorCount > 0 {
		return errors.New(firstFieldError(response.FieldErrors, "KLAS API 오류가 발생했습니다"))
	}
	return nil
}

func buildLoginToken(publicKeyBody string, studentID string, password string) (string, error) {
	publicKey, err := parsePublicKey(publicKeyBody)
	if err != nil {
		return "", err
	}

	payload := map[string]string{
		"loginId":   studentID,
		"loginPwd":  password,
		"storeIdYn": "N",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, payloadBytes)
	if err != nil {
		return "", fmt.Errorf("로그인 토큰 암호화 실패: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func parsePublicKey(publicKeyBody string) (*rsa.PublicKey, error) {
	pemText := "-----BEGIN PUBLIC KEY-----\n" +
		strings.TrimSpace(publicKeyBody) +
		"\n-----END PUBLIC KEY-----\n"

	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, errors.New("publicKey PEM 디코딩 실패")
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("publicKey 파싱 실패: %w", err)
	}

	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("publicKey가 RSA 키가 아닙니다")
	}
	return publicKey, nil
}

func looksLikeLoginHTML(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	if !bytes.HasPrefix(trimmed, []byte("<")) {
		return false
	}

	lower := bytes.ToLower(trimmed)
	return bytes.Contains(lower, []byte("<html")) ||
		bytes.Contains(lower, []byte("loginform.do")) ||
		bytes.Contains(lower, []byte("/usr/cmn/login"))
}

func firstFieldError(errors []fieldError, fallback string) string {
	for _, fieldError := range errors {
		if strings.TrimSpace(fieldError.Message) != "" {
			return fieldError.Message
		}
	}
	return fallback
}

func (c *Client) hasCookie(name string) bool {
	for _, cookie := range c.jar.Cookies(c.baseURL) {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func (c *Client) cookiesMap() map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range c.jar.Cookies(c.baseURL) {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}
