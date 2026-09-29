package account

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/domain"
	"path/filepath"
	"testing"
)

func TestKeyringServiceIsolation(t *testing.T) {
	for _, tc := range []struct{ env, want string }{{"", "klap"}, {"  ", "klap"}, {"klap-dev", "klap-dev"}} {
		t.Setenv("KLAP_KEYRING_SERVICE", tc.env)
		store := newStoreAt("unused", nil)
		if store.service != tc.want {
			t.Fatalf("service=%q want=%q", store.service, tc.want)
		}
	}
}

func TestDevelopmentSecretsDoNotReplaceRelease(t *testing.T) {
	secrets := newMemoryKeyring()
	t.Setenv("KLAP_KEYRING_SERVICE", "")
	release := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	t.Setenv("KLAP_KEYRING_SERVICE", "klap-dev")
	dev := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	ctx := context.Background()
	if err := release.Save(ctx, "student", "release-password", domain.Session{}); err != nil {
		t.Fatal(err)
	}
	if err := dev.Save(ctx, "student", "dev-password", domain.Session{}); err != nil {
		t.Fatal(err)
	}
	if err := dev.Remove(ctx, "student"); err != nil {
		t.Fatal(err)
	}
	password, err := release.LoadPassword(ctx, "student")
	if err != nil || password != "release-password" {
		t.Fatalf("release secret changed: %v", err)
	}
}
