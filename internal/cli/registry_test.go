package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func TestCommandRegistryDispatchAndHelpParity(t *testing.T) {
	registry := commandRegistry()
	seen := map[string]bool{}
	var help bytes.Buffer
	(Runner{Out: &help}).printHelp()
	for _, command := range registry {
		names := append([]string{command.Name}, command.Aliases...)
		for _, name := range names {
			if seen[name] {
				t.Fatalf("duplicate command %q", name)
			}
			seen[name] = true
		}
		if len(command.Usage) == 0 {
			t.Fatalf("missing help for %q", command.Name)
		}
		for _, usage := range command.Usage {
			if !strings.Contains(help.String(), usage+"\n") {
				t.Fatalf("missing usage %q", usage)
			}
			fields := strings.Fields(usage)
			if command.Name != "" && (len(fields) < 2 || fields[1] != command.Name) {
				t.Fatalf("usage does not match %q: %q", command.Name, usage)
			}
		}
		if command.Run == nil {
			continue
		} // modes are dispatched by bootstrap
		for _, name := range names {
			calls := 0
			want := errors.New("handler result")
			sentinel := &app.Service{}
			ctx := context.WithValue(context.Background(), struct{}{}, "context")
			isolated := command
			isolated.Run = func(r Runner, gotCtx context.Context, service *app.Service, args []string) error {
				calls++
				if gotCtx != ctx || r.Service != sentinel || service != sentinel || !reflect.DeepEqual(args, []string{"arg", "--flag"}) {
					t.Fatal("dispatch arguments changed")
				}
				return want
			}
			if err := dispatchCommand(ctx, Runner{Service: sentinel}, []string{name, "arg", "--flag"}, []commandSpec{isolated}); !errors.Is(err, want) || calls != 1 {
				t.Fatalf("dispatch %q: %v calls=%d", name, err, calls)
			}
		}
	}
	if err := (Runner{}).Run(context.Background(), []string{"unknown"}); err == nil || err.Error() != "unknown command: unknown" {
		t.Fatalf("unknown error=%v", err)
	}
	if err := (Runner{}).Run(context.Background(), []string{"tui"}); err == nil {
		t.Fatal("bootstrap mode was accepted as a CLI command")
	}
}

func TestHelpAliasesProduceSameOutput(t *testing.T) {
	var expected string
	for _, name := range []string{"help", "-h", "--help"} {
		var out bytes.Buffer
		if err := (Runner{Out: &out}).Run(context.Background(), []string{name}); err != nil {
			t.Fatal(err)
		}
		if expected == "" {
			expected = out.String()
		} else if out.String() != expected {
			t.Fatalf("help alias %q differs", name)
		}
	}
}

func TestReadmeCommandListMatchesRegistryHelp(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	(Runner{Out: &out}).printHelp()
	if err := validateReadmeHelp(string(data), out.String()); err != nil {
		t.Fatal(err)
	}
}

func validateReadmeHelp(readme, help string) error {
	// Git can check out Markdown as CRLF on Windows. Ignore only that
	// representation difference; command content and whitespace remain exact.
	readme = strings.ReplaceAll(readme, "\r\n", "\n")
	const start = "<!-- cli-help:start -->\n" + "```text\n"
	const end = "```\n<!-- cli-help:end -->"
	_, section, ok := strings.Cut(readme, start)
	if !ok {
		return errors.New("README command section missing")
	}
	section, _, ok = strings.Cut(section, end)
	if !ok {
		return errors.New("README command section end missing")
	}
	if section != help {
		return errors.New("README command list differs from registry help")
	}
	return nil
}

func TestReadmeHelpValidation(t *testing.T) {
	help := "KLAP CLI\n  klap help\n"
	readme := "# Intro\n<!-- cli-help:start -->\n```text\n" + help + "```\n<!-- cli-help:end -->\n"
	for _, test := range []struct {
		name, readme string
		wantError    bool
	}{
		{"LF", readme, false},
		{"CRLF", strings.ReplaceAll(readme, "\n", "\r\n"), false},
		{"mixed endings", strings.Replace(readme, "```text\n", "```text\r\n", 1), false},
		{"missing start", strings.Replace(readme, "cli-help:start", "other", 1), true},
		{"missing end", strings.Replace(readme, "cli-help:end", "other", 1), true},
		{"changed command", strings.Replace(readme, "klap help", "klap unknown", 1), true},
		{"changed indentation", strings.Replace(readme, "  klap", " klap", 1), true},
		{"empty section", strings.Replace(readme, help, "", 1), true},
		{"bare CR", strings.ReplaceAll(readme, "\n", "\r"), true},
		{"bare CR in body", strings.Replace(readme, "KLAP CLI\n", "KLAP CLI\r", 1), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateReadmeHelp(test.readme, help); (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}
