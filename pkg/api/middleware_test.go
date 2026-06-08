package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionStoreLogin(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)

	// Correct token
	session, ok := store.Login("test-token")
	if !ok {
		t.Fatal("expected login to succeed with correct token")
	}
	if session == "" {
		t.Fatal("expected non-empty session token")
	}

	// Wrong token
	_, ok = store.Login("wrong-token")
	if ok {
		t.Fatal("expected login to fail with wrong token")
	}
}

func TestSessionStoreValid(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)

	// No session
	if store.Valid("nonexistent") {
		t.Fatal("expected invalid for nonexistent session")
	}

	// Valid session
	session, _ := store.Login("test-token")
	if !store.Valid(session) {
		t.Fatal("expected valid for freshly created session")
	}

	// Expired session
	store.mu.Lock()
	store.sessions[session] = time.Now().Add(-time.Hour)
	store.mu.Unlock()
	if store.Valid(session) {
		t.Fatal("expected invalid for expired session")
	}
}

func TestSessionStoreCleanup(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)

	session1, _ := store.Login("test-token")
	store.mu.Lock()
	store.sessions[session1] = time.Now().Add(-time.Hour) // expired
	store.mu.Unlock()

	session2, _ := store.Login("test-token") // valid

	store.Cleanup()

	if store.Valid(session1) {
		t.Fatal("expected expired session to be cleaned up")
	}
	if !store.Valid(session2) {
		t.Fatal("expected valid session to remain after cleanup")
	}
}

func TestRequireAuth(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := store.RequireAuth(handler)

	// No cookie
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cookie, got %d", rec.Code)
	}

	// Valid cookie
	session, _ := store.Login("test-token")
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "zerodrop_admin_session",
		Value: session,
	})
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid cookie, got %d", rec.Code)
	}
}

func TestConstantTimeComparison(t *testing.T) {
	store := NewSessionStore("a-longer-admin-token-for-testing", 24*time.Hour, 5*time.Minute)

	// Various wrong tokens should all fail
	wrongTokens := []string{"", "wrong", "a-longer-admin-token-for-testin", "a-longer-admin-token-for-testing "}
	for _, tok := range wrongTokens {
		if _, ok := store.Login(tok); ok {
			t.Fatalf("expected login to fail for token %q", tok)
		}
	}

	// Exact match should succeed
	if _, ok := store.Login("a-longer-admin-token-for-testing"); !ok {
		t.Fatal("expected login to succeed with exact token")
	}
}

func TestGrantKeyAccess(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)
	session, _ := store.Login("test-token")

	if store.HasKeyGrant(session) {
		t.Fatal("expected no grant before GrantKeyAccess")
	}

	store.GrantKeyAccess(session)
	if !store.HasKeyGrant(session) {
		t.Fatal("expected grant after GrantKeyAccess")
	}
}

func TestHasKeyGrantExpired(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)
	session, _ := store.Login("test-token")

	store.GrantKeyAccess(session)
	store.mu.Lock()
	store.keyGrants[session] = time.Now().Add(-time.Minute)
	store.mu.Unlock()

	if store.HasKeyGrant(session) {
		t.Fatal("expected expired grant to be invalid")
	}
}

func TestRequireKeyGrant(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := store.RequireKeyGrant(handler)
	session, _ := store.Login("test-token")

	// Without grant
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Session-Token", session)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without grant, got %d", rec.Code)
	}

	// With grant
	store.GrantKeyAccess(session)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Session-Token", session)
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with grant, got %d", rec.Code)
	}
}
