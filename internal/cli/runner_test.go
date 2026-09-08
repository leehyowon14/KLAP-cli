package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunnerHelpUsesInjectedOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	r := Runner{Out: &out, ErrOut: &errOut}
	if err := r.Run(context.Background(), []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "KLAP CLI\n") || !strings.Contains(out.String(), "assignment") || errOut.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

func TestRunnerRejectsMissingCommand(t *testing.T) {
	if err := (Runner{}).Run(context.Background(), nil); err == nil {
		t.Fatal("expected missing command error")
	}
}
