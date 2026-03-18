package httpclient

import (
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout: 5 * time.Second,
}

// Do 는 외부에서 HTTP 요청을 실행할 때 사용하는 함수입니다.
func Do(req *http.Request) (*http.Response, error) {
	return client.Do(req)
}
