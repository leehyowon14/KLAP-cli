package klas

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type schemaResponseTransport string

func (body schemaResponseTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func schemaResponseClient(body string) *Client {
	return &Client{baseURL: &url.URL{Scheme: "https", Host: "klas.example"}, httpClient: &http.Client{Transport: schemaResponseTransport(body)}}
}

func TestErrorKindPreservesCauseAndMessage(t *testing.T) {
	cause := errors.New("original message")
	for _, kind := range []ErrorKind{ErrorNetwork, ErrorHTTP, ErrorSessionExpired, ErrorRemoteBusiness, ErrorSchema} {
		err := &Error{Kind: kind, Err: cause}
		if err.Error() != cause.Error() || !errors.Is(err, cause) || !IsErrorKind(fmt.Errorf("wrapped: %w", err), kind) {
			t.Fatalf("kind=%s err=%v", kind, err)
		}
	}
	if IsErrorKind(nil, ErrorSchema) || IsErrorKind(cause, ErrorNetwork) {
		t.Fatal("untyped error classified")
	}
}

func TestDecodeResponseJSONClassifiesSchemaErrors(t *testing.T) {
	for _, input := range []string{`{`, `{"count":"invalid"}`} {
		var value struct{ Count int }
		err := decodeResponseJSON([]byte(input), &value)
		if !IsErrorKind(err, ErrorSchema) {
			t.Fatalf("input=%s err=%v", input, err)
		}
		var syntax *json.SyntaxError
		var mismatch *json.UnmarshalTypeError
		if !errors.As(err, &syntax) && !errors.As(err, &mismatch) {
			t.Fatal("JSON cause lost")
		}
	}
	var value struct{ Count int }
	if err := decodeResponseJSON([]byte(`{"count":2}`), &value); err != nil || value.Count != 2 {
		t.Fatalf("value=%v err=%v", value, err)
	}
}
