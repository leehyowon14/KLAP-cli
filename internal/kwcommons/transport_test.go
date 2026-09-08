package kwcommons

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestResolveRequestContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/viewer/ssplayer/uniplayer_support/content.php" || r.URL.Query().Get("content_id") != "a & b" || r.UserAgent() != "KLAP-CLI/0.1" || r.Header.Get("Cookie") != "" {
			t.Errorf("request = %s %s %v", r.Method, r.URL, r.Header)
		}
		_, _ = io.WriteString(w, "<content><desktop><media_uri>/video.mp4</media_uri></desktop></content>")
	}))
	defer server.Close()
	client := NewClient(doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || r.URL.Host != "kwcommons.kw.ac.kr" {
			t.Fatalf("host = %s", r.URL)
		}
		redirected := r.Clone(r.Context())
		redirected.URL.Scheme = "http"
		redirected.URL.Host = strings.TrimPrefix(server.URL, "http://")
		return server.Client().Do(redirected)
	}))
	got, err := client.ResolveLectureMediaURL(context.Background(), " a & b ")
	if err != nil || got != "https://kwcommons.kw.ac.kr/video.mp4" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestResolveFailureContract(t *testing.T) {
	sentinel := errors.New("transport unavailable")
	client := NewClient(doerFunc(func(*http.Request) (*http.Response, error) { return nil, sentinel }))
	if _, err := client.ResolveLectureMediaURL(context.Background(), " "); err == nil || errors.Is(err, sentinel) {
		t.Fatalf("empty ID: %v", err)
	}
	if _, err := client.ResolveLectureMediaURL(context.Background(), "id"); !errors.Is(err, sentinel) {
		t.Fatalf("network: %v", err)
	}
	for _, tc := range []struct {
		status     int
		body, want string
	}{
		{503, "", "KWCommons HTTP 오류"},
		{200, "broken", "동영상 URL을 찾을 수 없습니다"},
	} {
		client := NewClient(doerFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		}))
		if _, err := client.ResolveLectureMediaURL(context.Background(), "id"); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("status %d: %v", tc.status, err)
		}
	}
}

func TestResolveCancellation(t *testing.T) {
	started := make(chan struct{})
	client := NewClient(doerFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.ResolveLectureMediaURL(ctx, "id"); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
