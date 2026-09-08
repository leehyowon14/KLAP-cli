package klas

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestTransportClassifiesRemoteErrors(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		kind       ErrorKind
	}{
		{"http", `{}`, 503, ErrorHTTP},
		{"expired", `{"loginRequired":true}`, 200, ErrorSessionExpired},
		{"business", `{"errorCount":1,"fieldErrors":[{"message":"denied"}]}`, 200, ErrorRemoteBusiness},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			base, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := &Client{httpClient: server.Client(), baseURL: base}
			for _, form := range []bool{false, true} {
				var err error
				if form {
					_, err = client.doForm(context.Background(), "/test", nil, false)
				} else {
					_, err = client.do(context.Background(), http.MethodPost, "/test", nil)
				}
				if !IsErrorKind(err, tt.kind) {
					t.Fatalf("form=%t error=%v", form, err)
				}
				var remote *Error
				if !errors.As(err, &remote) {
					t.Fatal("typed error missing")
				}
				if tt.kind == ErrorHTTP && remote.StatusCode != tt.status {
					t.Fatal("HTTP status lost")
				}
				if tt.kind == ErrorSessionExpired && !errors.Is(err, ErrSessionExpired) {
					t.Fatal("session sentinel lost")
				}
			}
		})
	}
}

type failingTransport struct {
	err  error
	read bool
}

func (f failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if f.read {
		return &http.Response{StatusCode: 200, Body: failingBody{err: f.err}, Header: make(http.Header)}, nil
	}
	return nil, f.err
}

type failingBody struct{ err error }

func (f failingBody) Read([]byte) (int, error) { return 0, f.err }
func (f failingBody) Close() error             { return nil }

func TestTransportPreservesNetworkAndReadCauses(t *testing.T) {
	base, err := url.Parse("https://klas.example")
	if err != nil {
		t.Fatal(err)
	}
	for _, cause := range []error{errors.New("network unavailable"), context.Canceled} {
		for _, read := range []bool{false, true} {
			client := &Client{baseURL: base, httpClient: &http.Client{Transport: failingTransport{err: cause, read: read}}}
			for _, form := range []bool{false, true} {
				var err error
				if form {
					_, err = client.doForm(context.Background(), "/test", nil, false)
				} else {
					_, err = client.do(context.Background(), http.MethodGet, "/test", nil)
				}
				if !IsErrorKind(err, ErrorNetwork) || !errors.Is(err, cause) {
					t.Fatalf("form=%t read=%t error=%v", form, read, err)
				}
			}
		}
	}
}
