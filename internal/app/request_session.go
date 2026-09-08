package app

import (
	"sync"

	"github.com/leehyowon14/KLAP-cli/internal/klas"
)

// requestSession is owned by one aggregate query, never by the shared service.
// Its successful lookups are reused only within that query. Errors are not cached.
type requestSession struct {
	mu        sync.Mutex
	studentID string
	clients   map[string]*klas.Client
	terms     map[string]Term
}

func (s *Service) withRequestSession() *Service {
	if s.requestSession != nil {
		return s
	}
	query := *s
	query.requestSession = &requestSession{
		clients: make(map[string]*klas.Client),
		terms:   make(map[string]Term),
	}
	return &query
}
