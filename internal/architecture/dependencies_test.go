package architecture

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const module = "github.com/leehyowon14/KLAP-cli"

type listedPackage struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

// Check direct edges, not transitive dependencies: cli -> app -> klas is valid.
// go list honours platform build constraints; CI runs this on all three OSes.
func TestDependencyBoundaries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "list", "-deps", "-json", "./...")
	command.Dir = "../.."
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, stderr.String())
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	seen := map[string]bool{}
	for {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		seen[pkg.ImportPath] = true
		for _, dependency := range pkg.Imports {
			if forbidden(pkg.ImportPath, dependency, false) {
				t.Errorf("forbidden dependency: %s -> %s", pkg.ImportPath, dependency)
			}
		}
		for _, dependency := range append(pkg.TestImports, pkg.XTestImports...) {
			if forbidden(pkg.ImportPath, dependency, true) {
				t.Errorf("forbidden test dependency: %s -> %s", pkg.ImportPath, dependency)
			}
		}
	}
	for _, name := range []string{"cli", "tui", "account", "app"} {
		if !seen[module+"/internal/"+name] {
			t.Errorf("guard did not inspect %s", name)
		}
	}
}

func within(path, root string) bool { return path == root || strings.HasPrefix(path, root+"/") }

func forbidden(source, dependency string, test bool) bool {
	internal := module + "/internal/"
	if within(source, internal+"cli") || within(source, internal+"tui") {
		for _, name := range []string{"klas", "account", "settings", "bootstrap", "platform"} {
			if within(dependency, internal+name) {
				return true
			}
		}
		if !test && dependency == "os/exec" {
			return true
		}
	}
	if within(source, internal+"cli") && within(dependency, internal+"tui") {
		return true
	}
	if within(source, internal+"account") && within(dependency, internal+"klas") {
		return true
	}
	if !test && within(source, internal+"app") {
		if dependency == "net/http" || dependency == "os/exec" {
			return true
		}
		for _, name := range []string{"bootstrap", "platform", "kwcommons", "academic", "download"} {
			if within(dependency, internal+name) {
				return true
			}
		}
	}
	return false
}

func TestBoundaryRulesRejectBackdoorsWithoutRejectingValidEdges(t *testing.T) {
	for _, test := range []struct {
		source, dependency string
		inTest, want       bool
	}{
		{"cli", "klas", false, true}, {"tui", "account", true, true},
		{"tui/child", "settings/store", false, true}, {"account", "klas", true, true},
		{"cli", "tui", false, true}, {"tui", "platform/macos", false, true},
		{"app", "download", false, true}, {"app", "download", true, false},
		{"cli", "app", false, false}, {"app", "klas", false, false},
		{"account", "domain", true, false}, {"cli", "klasextra", false, false},
	} {
		source, dependency := module+"/internal/"+test.source, module+"/internal/"+test.dependency
		if got := forbidden(source, dependency, test.inTest); got != test.want {
			t.Errorf("%s -> %s (test=%v): %v", source, dependency, test.inTest, got)
		}
	}
	for _, dependency := range []string{"net/http", "os/exec"} {
		if !forbidden(module+"/internal/app", dependency, false) || forbidden(module+"/internal/app", dependency, true) {
			t.Errorf("app IO rule: %s", dependency)
		}
	}
}
