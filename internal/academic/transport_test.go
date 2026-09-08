package academic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

type failingBody struct {
	err    error
	closed bool
}

func (b *failingBody) Read([]byte) (int, error) { return 0, b.err }
func (b *failingBody) Close() error             { b.closed = true; return nil }

func TestFetchAcademicHTTPContract(t *testing.T) {
	client := NewClient(doerFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != academicScheduleURL || r.Method != "GET" {
			t.Fatalf("request=%s %s", r.Method, r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("<h3>2026 학사일정</h3><tbody><tr><td>3월</td><td>2(월)</td><td>개강 &amp; 안내</td></tr></tbody>"))}, nil
	}))
	got, err := client.FetchAcademic(context.Background(), "2026")
	if err != nil || got.Year != "2026" || got.SourceURL != academicScheduleURL || len(got.Events) != 1 || got.Events[0].Title != "개강 & 안내" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestFetchAcademicFailureAndBodyClose(t *testing.T) {
	sentinel := errors.New("read failed")
	for _, status := range []int{200, 503} {
		body := &failingBody{err: sentinel}
		client := NewClient(doerFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: body}, nil
		}))
		_, err := client.FetchAcademic(context.Background(), "2026")
		if err == nil || !body.closed {
			t.Fatalf("status=%d closed=%v err=%v", status, body.closed, err)
		}
		if status == 200 && !errors.Is(err, sentinel) {
			t.Fatalf("read error=%v", err)
		}
		if status == 503 && !strings.Contains(err.Error(), "HTTP 503") {
			t.Fatalf("HTTP error=%v", err)
		}
	}
	client := NewClient(doerFunc(func(*http.Request) (*http.Response, error) { return nil, sentinel }))
	if _, err := client.FetchAcademic(context.Background(), "2026"); !errors.Is(err, sentinel) {
		t.Fatalf("network error=%v", err)
	}
}

func TestFetchAcademicCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	done := make(chan error, 1)
	client := NewClient(doerFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))
	go func() { _, err := client.FetchAcademic(ctx, "2026"); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error=%v", err)
	}
}

func TestParserMissingYearOrEvents(t *testing.T) {
	for _, body := range []string{"", "<h3>2025 학사일정</h3>", "<h3>2026 학사일정</h3><tbody></tbody>", "<!--<h3>2026 학사일정</h3>-->"} {
		if _, err := parseAcademicEvents([]byte(body), "2026"); err == nil {
			t.Fatalf("accepted %q", body)
		}
	}
}
