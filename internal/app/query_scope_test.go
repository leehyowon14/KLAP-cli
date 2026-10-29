package app

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"sync"
	"testing"
)

type scopedSettingsFake struct {
	value settings.Settings
	err   error
	saves int
}

func (s *scopedSettingsFake) Load() (settings.Settings, error) { return s.value, s.err }
func (s *scopedSettingsFake) Save(v settings.Settings) error   { s.saves++; s.value = v; return nil }

func TestTermScopesAreIndependent(t *testing.T) {
	store := &scopedSettingsFake{value: settings.Default()}
	base := &Service{settingsStore: store}
	var wg sync.WaitGroup
	for _, term := range []string{"2026-1", "2026,2", "2025-4"} {
		wg.Add(1)
		go func(term string) {
			defer wg.Done()
			s, err := base.ForTerm(term)
			if err != nil {
				t.Error(err)
				return
			}
			value, err := s.settingsStore.Load()
			if err != nil {
				t.Error(err)
			}
			want, _ := normalizeTermValue(term)
			if value.Term.Value != want {
				t.Errorf("got %s want %s", value.Term.Value, want)
			}
			if s.requestSession == base.requestSession {
				t.Error("shared request state")
			}
			if s.settingsStore.Save(value) == nil {
				t.Error("query scope allowed persistent writes")
			}
		}(term)
	}
	wg.Wait()
	if store.saves != 0 || store.value.Term.Value != "" {
		t.Fatal("base settings changed")
	}
	for _, bad := range []string{"2026-5", "x", "2026-01"} {
		if _, err := base.ForTerm(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	store.err = errors.New("unavailable")
	s, _ := base.ForTerm("2026-2")
	if _, err := s.settingsStore.Load(); !errors.Is(err, store.err) {
		t.Fatal(err)
	}
}
