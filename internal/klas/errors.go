package klas

import (
	"encoding/json"
	"errors"
)

// ErrorKind is a machine-readable failure category, independent of messages.
type ErrorKind string

const (
	ErrorNetwork        ErrorKind = "network"
	ErrorHTTP           ErrorKind = "http"
	ErrorSessionExpired ErrorKind = "session_expired"
	ErrorRemoteBusiness ErrorKind = "remote_business"
	ErrorSchema         ErrorKind = "schema"
)

type Error struct {
	Kind       ErrorKind
	StatusCode int
	Err        error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func IsErrorKind(err error, kind ErrorKind) bool {
	var remote *Error
	return errors.As(err, &remote) && remote.Kind == kind
}

func schemaError(err error) error { return &Error{Kind: ErrorSchema, Err: err} }

func decodeResponseJSON(data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return schemaError(err)
	}
	return nil
}
