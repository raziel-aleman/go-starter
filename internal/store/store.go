package store

import (
	"log"
	"net/http"
	"sync"
	"time"

	sm "github.com/raziel-aleman/go-starter/internal/session"
)

// InMemorySessionStore is a simple in-memory implementation of SessionStore.
// NOT suitable for production due to lack of persistence and scalability.
type InMemorySessionStore struct {
	sessions map[string]*sm.Session
	sync.RWMutex
}

// NewInMemorySessionStore creates a new InMemorySessionStore.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*sm.Session),
	}
}

// Read retrieves a session from the store.
func (s *InMemorySessionStore) Read(id string) (*sm.Session, error) {
	s.RLock()
	defer s.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, http.ErrNoCookie // Or a custom error for session not found
	}
	return session, nil
}

// Write saves a session to the store.
func (s *InMemorySessionStore) Write(session *sm.Session) error {
	s.Lock()
	defer s.Unlock()
	s.sessions[session.ID] = session
	return nil
}

// Destroy removes a session from the store.
func (s *InMemorySessionStore) Destroy(id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.sessions, id)
	return nil
}

// GarbageCollect removes expired sessions.
func (s *InMemorySessionStore) GarbageCollect(idleTimeout, absoluteTimeout time.Duration) error {
	s.Lock()
	defer s.Unlock()
	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.LastActive) > idleTimeout || now.Sub(session.CreatedAt) > absoluteTimeout {
			delete(s.sessions, id)
			log.Printf("Garbage collected session: %s", id)
		}
	}
	return nil
}
