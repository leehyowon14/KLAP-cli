package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAuthModelFocusMaskingAndSubmission(t *testing.T) {
	m := newAuthModel()
	if !m.inputs[0].Focused() || m.inputs[1].EchoMode != textinput.EchoPassword {
		t.Fatal("initial focus or masking changed")
	}
	m.inputs[0].SetValue("20260000")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(authModel)
	if m.focus != 1 || m.inputs[0].Focused() || !m.inputs[1].Focused() {
		t.Fatal("password focus not transferred")
	}
	m.inputs[1].SetValue("secret-password")
	if strings.Contains(m.View(), "secret-password") {
		t.Fatal("password exposed")
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(authModel)
	if !m.done || cmd == nil || m.View() != "" {
		t.Fatal("submission did not finish")
	}
	got, err := authCredentialsFromModel(m)
	if err != nil || got != (AuthCredentials{StudentID: "20260000", Password: "secret-password"}) {
		t.Fatalf("credentials failed: %v", err)
	}
}

func TestAuthModelCancellationAndFocusBounds(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC} {
		updated, cmd := newAuthModel().Update(tea.KeyMsg{Type: key})
		credentials, err := authCredentialsFromModel(updated)
		if err == nil || cmd == nil || credentials != (AuthCredentials{}) {
			t.Fatal("cancel accepted credentials")
		}
	}
	m := newAuthModel()
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyShiftTab, tea.KeyDown, tea.KeyDown, tea.KeyTab} {
		updated, _ := m.Update(tea.KeyMsg{Type: key})
		m = updated.(authModel)
		if m.focus < 0 || m.focus >= len(m.inputs) {
			t.Fatal("focus out of bounds")
		}
	}
	m.err = errors.New("form error")
	if _, err := authCredentialsFromModel(m); !errors.Is(err, m.err) {
		t.Fatal("form error lost")
	}
}

func TestRunnerAuthUsesInjectedPromptAndPropagatesCancellation(t *testing.T) {
	var in, out bytes.Buffer
	want := errors.New("cancelled")
	ctx := context.WithValue(context.Background(), struct{}{}, "ctx")
	calls := 0
	r := Runner{In: &in, Out: &out, AuthPrompt: func(gotCtx context.Context, gotIn io.Reader, gotOut io.Writer) (AuthCredentials, error) {
		calls++
		if gotCtx != ctx || gotIn != &in || gotOut != &out {
			t.Fatal("prompt IO/context not injected")
		}
		return AuthCredentials{}, want
	}}
	if err := r.Run(ctx, []string{"auth"}); !errors.Is(err, want) || calls != 1 || out.Len() != 0 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestAuthFormRequiresInputAndHonorsContext(t *testing.T) {
	if _, err := runAuthForm(context.Background(), nil, io.Discard); err == nil {
		t.Fatal("missing input accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runAuthForm(ctx, strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("cancelled context accepted")
	}
}
