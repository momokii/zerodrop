package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// SessionStore manages admin authentication sessions and key access grants.
type SessionStore struct {
	mu          sync.RWMutex
	sessions    map[string]time.Time // token -> expiry
	keyGrants   map[string]time.Time
	adminKey    string
	maxAge      time.Duration
	keyGrantTTL time.Duration
}

// NewSessionStore creates a new session store with the given admin token
// and session lifetime. maxAge controls how long sessions remain valid.
// keyGrantTTL controls how long a key access grant lasts before re-auth.
func NewSessionStore(adminToken string, maxAge, keyGrantTTL time.Duration) *SessionStore {
	return &SessionStore{
		sessions:    make(map[string]time.Time),
		keyGrants:   make(map[string]time.Time),
		adminKey:    adminToken,
		maxAge:      maxAge,
		keyGrantTTL: keyGrantTTL,
	}
}

// Logout invalidates a session token.
func (s *SessionStore) Logout(sessionToken string) {
	s.mu.Lock()
	delete(s.sessions, sessionToken)
	s.mu.Unlock()
}

func (s *SessionStore) MaxAge() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxAge
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

// GrantKeyAccess marks a session as having key access for the configured TTL.
func (s *SessionStore) GrantKeyAccess(sessionToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyGrants[sessionToken] = time.Now().Add(s.keyGrantTTL)
}

// HasKeyGrant checks if a session has a valid, non-expired key access grant.
func (s *SessionStore) HasKeyGrant(sessionToken string) bool {
	s.mu.RLock()
	expiry, ok := s.keyGrants[sessionToken]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		s.mu.Lock()
		delete(s.keyGrants, sessionToken)
		s.mu.Unlock()
		return false
	}
	return true
}

// RequireKeyGrant is middleware that checks for a valid key access grant.
// The session token is extracted from the same sources as RequireAuth.
// Returns 403 if no valid grant exists.
func (s *SessionStore) RequireKeyGrant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := extractSession(r)
		if session == "" || !s.HasKeyGrant(session) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "key_access_denied",
				"message": "Re-authenticate at POST /api/admin/key/grant to access key material.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractSession extracts the session token from an HTTP request.
// Tried in order: X-Session-Token header, Authorization: Bearer, cookie.
func extractSession(r *http.Request) string {
	session := r.Header.Get("X-Session-Token")
	if session == "" {
		if ah := r.Header.Get("Authorization"); len(ah) > 7 && ah[:7] == "Bearer " {
			session = ah[7:]
		}
	}
	if session == "" {
		if c, err := r.Cookie("zerodrop_admin_session"); err == nil {
			session = c.Value
		}
	}
	return session
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RequireAuth is middleware that checks for a valid admin session.
// It accepts the session token from three sources (tried in order):
//   1. X-Session-Token header (used by SPA — reliable in fetch API)
//   2. Authorization: Bearer <token> header (standard HTTP auth)
//   3. zerodrop_admin_session cookie (backward compatibility)
func (s *SessionStore) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := extractSession(r)
		if session == "" || !s.Valid(session) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
