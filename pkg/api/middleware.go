package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// SessionStore manages admin authentication sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time // token -> expiry
	adminKey string
	maxAge   time.Duration
}

// NewSessionStore creates a new session store with the given admin token.
func NewSessionStore(adminToken string) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]time.Time),
		adminKey: adminToken,
		maxAge:   24 * time.Hour,
	}
}

// Login validates the admin token and creates a session.
func (s *SessionStore) Login(providedToken string) (string, bool) {
	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(s.adminKey)) != 1 {
		return "", false
	}
	sessionToken := generateToken()
	s.mu.Lock()
	s.sessions[sessionToken] = time.Now().Add(s.maxAge)
	s.mu.Unlock()
	return sessionToken, true
}

// Valid checks if a session token is valid and not expired.
func (s *SessionStore) Valid(sessionToken string) bool {
	s.mu.RLock()
	expiry, ok := s.sessions[sessionToken]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		s.mu.Lock()
		delete(s.sessions, sessionToken)
		s.mu.Unlock()
		return false
	}
	return true
}

// Cleanup removes expired sessions.
func (s *SessionStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, expiry := range s.sessions {
		if now.After(expiry) {
			delete(s.sessions, token)
		}
	}
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RequireAuth is middleware that checks for a valid admin session.
func (s *SessionStore) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("zerodrop_admin_session")
		if err != nil || !s.Valid(cookie.Value) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
