package tui

import (
	"context"
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

func TestAuthCheckShowsSetupWhenNoUsers(t *testing.T) {
	service := newTUITestService(t)
	m := model{
		ctx:     context.Background(),
		service: service,
		active:  screenAuth,
		loading: true,
		auth:    authScreenModel{authInputs: newAuthInputs()},
	}

	msg := checkAuthUsers(m.ctx, m.service)().(authCheckMsg)
	updated, cmd := m.Update(msg)
	got := updated.(model)
	if cmd == nil {
		t.Fatal("Update(authCheckMsg) should start text input blink")
	}
	if got.active != screenAuth || got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
	if len(got.auth.authInputs) != 2 {
		t.Fatalf("authInputs len = %d", len(got.auth.authInputs))
	}
}

func TestAuthEnterValidatesEmptyFieldsWithoutSubmit(t *testing.T) {
	m := model{active: screenAuth,
		auth: authScreenModel{authInputs: newAuthInputs(), authFocus: 1}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if cmd != nil {
		t.Fatal("empty auth form should not submit")
	}
	if got.err == nil || got.auth.authSubmitting {
		t.Fatalf("err=%v authSubmitting=%t", got.err, got.auth.authSubmitting)
	}
}

type authScreenServiceStub struct {
	calls        int
	ctx          context.Context
	id, password string
	err          error
}

func (s *authScreenServiceStub) Users(context.Context) ([]app.UserRow, error) { return nil, s.err }
func (s *authScreenServiceStub) Authenticate(ctx context.Context, id, password string) error {
	s.calls++
	s.ctx = ctx
	s.id = id
	s.password = password
	return s.err
}

func TestAuthChildDefersIOAndBlocksDuplicateSubmission(t *testing.T) {
	ctx := context.Background()
	service := &authScreenServiceStub{err: errors.New("authentication failed")}
	child := authScreenModel{authInputs: newAuthInputs(), authFocus: 1}
	child.authInputs[0].SetValue(" 20260000 ")
	child.authInputs[1].SetValue(" password ")
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, ctx, service)
	if !handled || action.cmd == nil || !child.authSubmitting || service.calls != 0 {
		t.Fatal("submit did not defer IO")
	}
	if duplicate, _ := child.Update(tea.KeyMsg{Type: tea.KeyEnter}, ctx, service); duplicate.cmd != nil {
		t.Fatal("duplicate submit")
	}
	child.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, ctx, service)
	if child.authInputs[1].Value() != " password " {
		t.Fatal("input changed during submit")
	}
	result := action.cmd().(authSubmitMsg)
	if service.calls != 1 || service.ctx != ctx || service.id != "20260000" || service.password != "password" {
		t.Fatal("auth command arguments changed")
	}
	action, _ = child.Update(result, ctx, service)
	if child.authSubmitting || !errors.Is(action.err, service.err) || child.authInputs[1].Value() != " password " {
		t.Fatal("failed submission state changed")
	}
	service.err = nil
	action, _ = child.Update(tea.KeyMsg{Type: tea.KeyEnter}, ctx, service)
	child.Update(action.cmd(), ctx, service)
	if child.authSubmitting || child.authInputs != nil || child.authFocus != 0 {
		t.Fatal("success did not clear credentials")
	}
}

func TestAuthChildViewMasksPasswordAndLeavesGlobalMessages(t *testing.T) {
	child := authScreenModel{authInputs: newAuthInputs()}
	child.authInputs[1].SetValue("secret-password")
	if strings.Contains(child.View(80, false, nil), "secret-password") {
		t.Fatal("password exposed")
	}
	if _, handled := child.Update(tea.WindowSizeMsg{Width: 80}, context.Background(), nil); handled {
		t.Fatal("window message consumed")
	}
}

func TestAuthChildCompletionPreservesRootPrefetch(t *testing.T) {
	m := model{ctx: context.Background(), active: screenAuth, auth: authScreenModel{authInputs: newAuthInputs(), authSubmitting: true}}
	updated, cmd := m.Update(authSubmitMsg{})
	got := updated.(model)
	if got.active != screenHome || !got.prefetchActive || cmd == nil || got.auth.authInputs != nil {
		t.Fatal("authenticated route/prefetch changed")
	}
}
