package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionStoreLogin(t *testing.T) {
	store := NewSessionStore("test-token", 24*time.Hour)

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
	store := NewSessionStore("test-token", 24*time.Hour)

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
	store := NewSessionStore("test-token", 24*time.Hour)

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
	store := NewSessionStore("test-token", 24*time.Hour)

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
	store := NewSessionStore("a-longer-admin-token-for-testing", 24*time.Hour)

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
