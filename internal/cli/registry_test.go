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
	const start = "<!-- cli-help:start -->\n" + "```text\n"
	const end = "```\n<!-- cli-help:end -->"
	_, section, ok := strings.Cut(string(data), start)
	if !ok {
		t.Fatal("README command section missing")
	}
	section, _, ok = strings.Cut(section, end)
	if !ok {
		t.Fatal("README command section end missing")
	}
	var out bytes.Buffer
	(Runner{Out: &out}).printHelp()
	if section != out.String() {
		t.Fatal("README command list differs from registry help")
	}
}
