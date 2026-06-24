package api

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for the request ID.
	RequestIDKey contextKey = "request_id"
)

// RequestID is middleware that assigns a unique ID to every request.
// The ID is set on the response as the X-Request-ID header and stored
// in the request context for use by downstream handlers and loggers.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID creates a 16-byte hex-encoded request ID.
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetRequestID extracts the request ID from a context (set by RequestID middleware).
// Returns empty string if no request ID is present.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// CORSMiddleware returns middleware that sets CORS headers based on the
// configured origin. When origin is empty, no CORS headers are set
// (same-origin requests only — this is the default and recommended for
// production since the SPA is served by the Go backend on the same origin).
//
// When origin is set (e.g. "http://localhost:3000" for frontend dev server),
// the middleware allows cross-origin requests from that origin with standard
// methods and credentials.
func CORSMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-Token, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders is middleware that sets security-related HTTP headers
// to harden the application against common web vulnerabilities.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=(), fullscreen=(self)")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' blob:; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"media-src 'self' blob:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// RequestLogger is middleware that logs every HTTP request with method, path,
// status code, duration, and request ID.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		duration := time.Since(start)
		reqID := GetRequestID(r.Context())
		log.Printf("[%s] %s %s %d %s",
			reqID,
			r.Method,
			r.URL.Path,
			sw.status,
			duration,
		)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code for logging.
type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.written {
		w.status = status
		w.written = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.status = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("statusWriter: ResponseWriter does not implement http.Hijacker")
}

// GzipMiddleware compresses HTTP responses with gzip when the client
// indicates support via Accept-Encoding. Only wraps responses; compressed
// content is streamed with Content-Encoding: gzip header set automatically.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

// gzipResponseWriter wraps http.ResponseWriter to write gzip-compressed data.
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

// addRetryAfter sets the Retry-After header on the response if the status
// code is 429 (Too Many Requests). The retry delay is calculated from the
// rate limit window (1 hour).
func addRetryAfter(w http.ResponseWriter, code int) {
	if code == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Hour.Seconds())))
	}
}

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
			APIError{Code: http.StatusForbidden, Message: "Key access denied. Re-authenticate at POST /api/admin/key/grant to access key material."}.Send(w)
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
			ErrUnauthorized.Send(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
