package klas

import (
	"context"
	"testing"
)

func TestAuthSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{").loginSecurity(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{").loginConfirm(context.Background(), "token"); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}

func TestAuthRequiredResponseSchemaErrors(t *testing.T) {
	if _, err := schemaResponseClient("{}").loginSecurity(context.Background()); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := schemaResponseClient("{}").loginConfirm(context.Background(), "token"); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
	if _, err := parsePublicKey("invalid"); !IsErrorKind(err, ErrorSchema) {
		t.Fatal(err)
	}
}
