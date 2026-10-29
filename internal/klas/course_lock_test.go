package klas

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type courseTransport struct {
	mu       sync.Mutex
	selected string
}

func (t *courseTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	body := `{}`
	if strings.HasSuffix(r.URL.Path, "LctrumHomeStdInfo.do") {
		var v map[string]string
		_ = json.NewDecoder(r.Body).Decode(&v)
		t.selected = v["selectSubj"]
	} else {
		body = `[{"subj":"` + t.selected + `","sbjt":"lecture"}]`
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}
func TestCourseTransactionsAcrossClients(t *testing.T) {
	transport := &courseTransport{}
	clients := make([]*Client, 2)
	for i := range clients {
		clients[i], _ = NewClient()
		clients[i].SetSession(Session{Cookies: map[string]string{"SESSION": "test-shared-context"}})
		clients[i].httpClient.Transport = transport
	}
	var wg sync.WaitGroup
	for i, c := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			course := Course{Value: []string{"A", "B"}[i]}
			for range 30 {
				rows, err := c.Lectures(context.Background(), "2026,2", course)
				if err != nil || len(rows) != 1 {
					t.Errorf("rows=%v err=%v", rows, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
func TestCourseLockCancellationAndRelease(t *testing.T) {
	c, _ := NewClient()
	c.SetSession(Session{Cookies: map[string]string{"SESSION": "test-cancel"}})
	unlock, err := c.lockCourse(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.lockCourse(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("%v", err)
	}
	unlock()
	unlock, err = c.lockCourse(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}
func TestLectureCourseMismatchIsNotEmptySuccess(t *testing.T) {
	for _, body := range []string{`[{"subj":"other"}]`, `[{"subj":"expected"},{"subj":"other"}]`, `[{"subj":"expected","year":"2025","hakgi":"2"}]`, `[{"subj":"expected","year":"2026","hakgi":"1"}]`} {
		rows, err := schemaResponseClient(body).Lectures(context.Background(), "2026,2", Course{Value: "expected"})
		if !IsErrorKind(err, ErrorCourseMismatch) || rows != nil {
			t.Fatalf("rows=%v err=%v", rows, err)
		}
	}
	rows, err := schemaResponseClient(`[]`).Lectures(context.Background(), "2026,2", Course{Value: "expected"})
	if err != nil || len(rows) != 0 {
		t.Fatalf("%v %v", rows, err)
	}
}
