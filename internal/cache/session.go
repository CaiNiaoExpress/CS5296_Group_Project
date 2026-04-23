package cache

import (
	"sync"
	"time"
)

type SessionContext struct {
	LastIntent string
	LastReply  string
	UpdatedAt  time.Time
}

type SessionStore struct {
	mu    sync.RWMutex
	store map[string]SessionContext
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		store: make(map[string]SessionContext),
	}
}

func (s *SessionStore) Get(sessionID string) (SessionContext, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ctx, ok := s.store[sessionID]
	return ctx, ok
}

func (s *SessionStore) Set(sessionID string, ctx SessionContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[sessionID] = ctx
}
