package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestAuthCheckShowsSetupWhenNoUsers(t *testing.T) {
	service := newTUITestService(t)
	m := model{
		ctx:        context.Background(),
		service:    service,
		active:     screenAuth,
		loading:    true,
		authInputs: newAuthInputs(),
	}

	msg := m.checkAuthUsers()().(authCheckMsg)
	updated, cmd := m.Update(msg)
	got := updated.(model)
	if cmd == nil {
		t.Fatal("Update(authCheckMsg) should start text input blink")
	}
	if got.active != screenAuth || got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
	if len(got.authInputs) != 2 {
		t.Fatalf("authInputs len = %d", len(got.authInputs))
	}
}

func TestAuthEnterValidatesEmptyFieldsWithoutSubmit(t *testing.T) {
	m := model{active: screenAuth, authInputs: newAuthInputs(), authFocus: 1}
	updated, cmd := m.updateAuth(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if cmd != nil {
		t.Fatal("empty auth form should not submit")
	}
	if got.err == nil || got.authSubmitting {
		t.Fatalf("err=%v authSubmitting=%t", got.err, got.authSubmitting)
	}
}
