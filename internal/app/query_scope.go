package app

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
)

// ForTerm creates an independent query scope. It never changes the CLI or
// desktop's selected term, and is safe to use alongside other query scopes.
func (s *Service) ForTerm(term string) (*Service, error) {
	value, err := normalizeTermValue(term)
	if err != nil {
		return nil, err
	}
	query := *s
	query.requestSession = nil
	if value != "" {
		query.settingsStore = termSettings{base: s.settingsStore, term: value}
	}
	return query.withRequestSession(), nil
}

type termSettings struct {
	base SettingsStore
	term string
}

func (s termSettings) Load() (settings.Settings, error) {
	value, err := s.base.Load()
	value.Term.Value = s.term
	return value, err
}
func (s termSettings) Save(settings.Settings) error {
	return errors.New("query term scopes cannot change persistent settings")
}
