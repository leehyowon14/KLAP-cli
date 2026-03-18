package utils

import "net/http"

// SetJsonRequestHeaders는 KLAS가 요구하는 최소한의 Request Header를 구성해줍니다.
func SetJsonRequestHeaders(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
}
