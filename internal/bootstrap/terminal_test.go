package bootstrap

import (
	"bytes"
	"os"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/cli"
)

func TestTerminalCapabilitiesEnvironmentPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"redirect", nil, false},
		{"force", map[string]string{"KLAP_FORCE_HYPERLINKS": "1"}, true},
		{"dumb overrides force", map[string]string{"TERM": "dumb", "KLAP_FORCE_HYPERLINKS": "1"}, false},
		{"disabled overrides force", map[string]string{"KLAP_NO_HYPERLINKS": "1", "KLAP_FORCE_HYPERLINKS": "1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := detectTerminalCapabilities(&bytes.Buffer{}, func(key string) string { return tc.env[key] })
			if got != (cli.TerminalCapabilities{Hyperlinks: tc.want}) {
				t.Fatalf("capabilities=%+v", got)
			}
		})
	}
}

func TestTerminalCapabilitiesCharacterDeviceAndClosedFile(t *testing.T) {
	file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	got := detectTerminalCapabilities(file, func(string) string { return "" })
	if !got.Hyperlinks || !got.InPlaceProgress {
		t.Fatalf("character device changed: %+v", got)
	}
	got = detectTerminalCapabilities(file, func(key string) string {
		if key == "TERM" {
			return "dumb"
		}
		return ""
	})
	if got != (cli.TerminalCapabilities{}) {
		t.Fatalf("dumb=%+v", got)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	got = detectTerminalCapabilities(file, func(string) string { return "" })
	if got != (cli.TerminalCapabilities{}) {
		t.Fatalf("closed=%+v", got)
	}
}
