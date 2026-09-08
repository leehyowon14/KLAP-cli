package download

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadResumeAndFullRestart(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "range ignored", true: "resume"}[partial], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Range") != "bytes=3-" || r.UserAgent() != "KLAP-CLI/0.1" {
					t.Errorf("headers=%v", r.Header)
				}
				if partial {
					w.Header().Set("Content-Range", "bytes 3-5/6")
					w.WriteHeader(http.StatusPartialContent)
					_, _ = io.WriteString(w, "def")
				} else {
					_, _ = io.WriteString(w, "abcdef")
				}
			}))
			defer server.Close()
			path := filepath.Join(t.TempDir(), "video.mp4")
			if err := os.WriteFile(path+".part", []byte("abc"), 0600); err != nil {
				t.Fatal(err)
			}
			var progress []int64
			n, err := NewClient(server.Client()).DownloadFile(context.Background(), server.URL, path, true, func(written, total int64) { progress = append(progress, written) })
			body, readErr := os.ReadFile(path)
			if err != nil || readErr != nil || n != 6 || string(body) != "abcdef" || len(progress) == 0 || progress[len(progress)-1] != 6 {
				t.Fatalf("bytes=%d body=%q err=%v read=%v progress=%v", n, body, err, readErr, progress)
			}
			if _, err := os.Stat(path + ".part"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("partial remains: %v", err)
			}
		})
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

func TestDownloadReadFailurePartialPolicy(t *testing.T) {
	sentinel := errors.New("read failed")
	for _, keep := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "video.mp4")
		client := NewClient(doerFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, ContentLength: -1, Body: io.NopCloser(io.MultiReader(strings.NewReader("abc"), errorReader{sentinel}))}, nil
		}))
		n, err := client.DownloadFile(context.Background(), "https://example.test/video", path, keep, nil)
		if n != 3 || !errors.Is(err, sentinel) {
			t.Fatalf("bytes=%d err=%v", n, err)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("final exists: %v", err)
		}
		body, err := os.ReadFile(path + ".part")
		if keep && (err != nil || string(body) != "abc") {
			t.Fatalf("keep partial=%q err=%v", body, err)
		}
		if !keep && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("remove partial err=%v", err)
		}
	}
}

func TestDownloadExistingFinalAndHTTPFailurePreserveFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	client := NewClient(doerFunc(func(*http.Request) (*http.Response, error) { t.Fatal("existing file triggered HTTP"); return nil, nil }))
	if _, err := client.DownloadFile(context.Background(), "https://example.test", path, false, nil); err == nil {
		t.Fatal("existing accepted")
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "existing" {
		t.Fatalf("existing=%q err=%v", body, err)
	}
	path = filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(path+".part", []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	client = NewClient(doerFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Status: "503 unavailable", Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	if _, err := client.DownloadFile(context.Background(), "https://example.test", path, false, nil); err == nil {
		t.Fatal("HTTP failure accepted")
	}
	body, err = os.ReadFile(path + ".part")
	if err != nil || string(body) != "abc" {
		t.Fatalf("existing partial=%q err=%v", body, err)
	}
}

func TestDownloadCanceledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(t.TempDir(), "video.mp4")
	client := NewClient(doerFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() }))
	if _, err := client.DownloadFile(ctx, "https://example.test", path, true, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel=%v", err)
	}
	if _, err := os.Stat(path + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial=%v", err)
	}
}
