package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const secure = true

// Session represents a user session.
type Session struct {
	ID           string         `json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	LastActive   time.Time      `json:"last_active"`
	Data         map[string]any `json:"data"`
	sync.RWMutex                // For concurrent access to session data
}

// NewSession creates a new session with a unique ID.
func NewSession() (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}
	return &Session{
		ID:         id,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Data:       map[string]any{"csrf_token": generateCSRFToken(), "username": ""},
	}, nil
}

// Get retrieves a value from the session data.
func (s *Session) Get(key string) any {
	s.RLock()
	defer s.RUnlock()
	return s.Data[key]
}

// Put sets a value in the session data.
func (s *Session) Put(key string, value any) {
	s.Lock()
	defer s.Unlock()
	s.Data[key] = value
	s.LastActive = time.Now() // Update last active time on data change
}

// Delete removes a value from the session data.
func (s *Session) Delete(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.Data, key)
	s.LastActive = time.Now() // Update last active time on data change
}

// SessionStore defines the interface for storing and retrieving sessions.
type SessionStore interface {
	Read(id string) (*Session, error)
	Write(session *Session) error
	Destroy(id string) error
	GarbageCollect(idleTimeout, absoluteTimeout time.Duration) error
}

// SessionManager manages sessions, including their lifecycle and interaction with the store.
type SessionManager struct {
	Store              SessionStore
	CookieName         string
	IdleExpiration     time.Duration
	AbsoluteExpiration time.Duration
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(
	store SessionStore,
	cookieName string,
	idleExpiration,
	absoluteExpiration time.Duration) *SessionManager {
	sm := &SessionManager{
		Store:              store,
		CookieName:         cookieName,
		IdleExpiration:     idleExpiration,
		AbsoluteExpiration: absoluteExpiration,
	}
	// Start garbage collection in a goroutine
	go sm.startGarbageCollection()
	return sm
}

// startGarbageCollection runs garbage collection periodically.
func (sm *SessionManager) startGarbageCollection() {
	ticker := time.NewTicker(sm.IdleExpiration / 2) // Run GC more frequently than idle expiration
	defer ticker.Stop()
	for range ticker.C {
		if err := sm.Store.GarbageCollect(sm.IdleExpiration, sm.AbsoluteExpiration); err != nil {
			log.Printf("Error during session garbage collection: %v", err)
		}
	}
}

// generateSessionID generates a secure, random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32) // 32 bytes for a secure ID
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// genrateCSRFToken generates a 42-character base64 string with 256 bits of randomness CSRF token
func generateCSRFToken() string {
	id := make([]byte, 32)

	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		panic("failed to generate CSRF token")
	}

	return base64.RawURLEncoding.EncodeToString(id)
}

// verifyCSRFToken extracts the CSRF token from a given session and validates
// it against the csrf_token form value or the X-CSRF-Token header.
func (m *SessionManager) verifyCSRFToken(r *http.Request, session *Session) bool {
	sToken, ok := session.Get("csrf_token").(string)
	if !ok {
		return false
	}

	token := r.FormValue("csrf_token")

	if token == "" {
		token = r.Header.Get("X-XSRF-Token")
	}

	return token == sToken
}

// sessionContextKey is a type for context keys to avoid collisions.
type sessionContextKey int

const (
	sessionKey sessionContextKey = iota
)

// GetSession retrieves the session from the request context.
func GetSession(r *http.Request) *Session {
	session, ok := r.Context().Value(sessionKey).(*Session)
	if !ok {
		panic("session not found in request context")
	}
	return session
}

// SessionMiddleware is the middleware for session management.
func (sm *SessionManager) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var session *Session
		sessionID, err := r.Cookie(sm.CookieName)

		if err == nil {
			// Cookie found, try to read session from store
			session, err = sm.Store.Read(sessionID.Value)
			if err != nil || !sm.isValid(session) {
				// Session not found or invalid, create a new one
				log.Printf("Existing session invalid or not found, creating new.")
				session, _ = NewSession() // Error handling for NewSession ignored for brevity in this example
			}
		} else {
			// No session cookie, create a new session
			session, _ = NewSession() // Error handling for NewSession ignored for brevity in this example
		}

		// Attach the session to the request context
		ctx := context.WithValue(r.Context(), sessionKey, session)
		r = r.WithContext(ctx)

		// Create a custom response writer to save the session before writing headers
		srw := &SessionResponseWriter{
			ResponseWriter:   w,
			Session:          session,
			Manager:          sm,
			HeaderWritten:    false,
			SessionDestroyed: false,
			StatusCode:       http.StatusOK, // Initialize with default 200 OK
		}

		w.Header().Add("Vary", "Cookie")
		w.Header().Add("Cache-Control", `no-cache="Set-Cookie"`)

		if r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete {
			if !sm.verifyCSRFToken(r, session) {
				http.Error(srw, "CSRF token mismatch", http.StatusForbidden)
				return
			}
		}

		// This defer ensures WriteHeader is called at the end if the handler
		// doesn't explicitly call it or write a body.
		defer func() {
			if !srw.HeaderWritten {
				srw.WriteHeader(srw.StatusCode) // Use the captured or default status
			}
		}()

		next.ServeHTTP(srw, r)
	})
}

// isValid checks if a session is still valid based on expiration times.
func (sm *SessionManager) isValid(session *Session) bool {
	if session == nil {
		return false
	}
	now := time.Now()
	if now.Sub(session.LastActive) > sm.IdleExpiration || now.Sub(session.CreatedAt) > sm.AbsoluteExpiration {
		// Session expired
		sm.Store.Destroy(session.ID) // Destroy expired session
		return false
	}
	return true
}

// Migrate updates session from unauthenticated user to authenticated user
func (sm *SessionManager) Migrate(session *Session) (*Session, error) {
	session.Lock()
	defer session.Unlock()

	newSession, _ := NewSession()
	for k, v := range session.Data {
		if k == "csrf_token" {
			continue
		}
		newSession.Put(k, v)
	}

	err := sm.Store.Destroy(session.ID)
	if err != nil {
		return session, err
	}

	return newSession, err
}

// SessionResponseWriter wraps http.ResponseWriter to handle session saving and cookie setting.
type SessionResponseWriter struct {
	http.ResponseWriter
	Session          *Session
	Manager          *SessionManager
	HeaderWritten    bool
	SessionDestroyed bool // NEW: Flag to indicate if the session has been destroyed
	StatusCode       int  // Stores the status code to be written
}

// WriteHeader captures the status code and manages header writing.
func (srw *SessionResponseWriter) WriteHeader(statusCode int) {
	if srw.HeaderWritten {
		log.Println("Warning: WriteHeader called multiple times (superfluous).")
		return // Ignore subsequent calls
	}

	srw.StatusCode = statusCode                    // Capture the status code
	srw.writeCookieIfNecessary()                   // Add the Set-Cookie header(s)
	srw.ResponseWriter.WriteHeader(srw.StatusCode) // Now, write the actual status code to the underlying writer
	srw.HeaderWritten = true
}

// Write ensures headers are written (if not already) before writing the body.
func (srw *SessionResponseWriter) Write(b []byte) (int, error) {
	if !srw.HeaderWritten {
		// If Write is called first, set default status to 200 OK.
		// Then call WriteHeader through our wrapper to ensure cookies are set.
		srw.WriteHeader(http.StatusOK)
	}
	return srw.ResponseWriter.Write(b)
}

// writeCookieIfNecessary adds the Set-Cookie header but does NOT call WriteHeader.
func (srw *SessionResponseWriter) writeCookieIfNecessary() {
	var cookie *http.Cookie
	if srw.SessionDestroyed {
		log.Println("Session destroyed, preparing clear cookie.")
		cookie = &http.Cookie{
			Name:     srw.Manager.CookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1, // Expires immediately
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		}
	} else if srw.Session != nil {
		srw.Session.LastActive = time.Now()
		if err := srw.Manager.Store.Write(srw.Session); err != nil {
			log.Printf("Error saving session %s: %v", srw.Session.ID, err)
		}
		cookie = &http.Cookie{
			Name:     srw.Manager.CookieName,
			Value:    srw.Session.ID,
			Path:     "/",
			Expires:  time.Now().Add(srw.Manager.AbsoluteExpiration),
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		}
	}

	if cookie != nil {
		// This adds the cookie header. It does NOT implicitly send headers on its own.
		http.SetCookie(srw.ResponseWriter, cookie)
	}
}
