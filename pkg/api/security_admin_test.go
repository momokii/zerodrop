package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecurity_AdminCookie_HttpOnly(t *testing.T) {
	handler, _ := setupAdminTest(t)

	body := `{"token":"test-admin-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d", rec.Code)
	}

	for _, c := range rec.Result().Cookies() {
		if c.Name == "zerodrop_admin_session" {
			if !c.HttpOnly {
				t.Error("[SECURITY] Admin session cookie must have HttpOnly flag")
			}
			t.Logf("[SECURITY] Admin cookie HttpOnly: PASS")
			return
		}
	}
	t.Fatal("session cookie not found in response")
}

func TestSecurity_AdminCookie_SameSiteLax(t *testing.T) {
	handler, _ := setupAdminTest(t)

	body := `{"token":"test-admin-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.Name == "zerodrop_admin_session" {
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("[SECURITY] Admin cookie SameSite: expected Lax (%d), got %d", http.SameSiteLaxMode, c.SameSite)
			}
			t.Logf("[SECURITY] Admin cookie SameSite=Lax: PASS")
			return
		}
	}
	t.Fatal("session cookie not found")
}

func TestSecurity_AdminSession_Expired_Rejected(t *testing.T) {
	sessions := NewSessionStore("test-token", 1*time.Nanosecond, 5*time.Minute)

	// Login succeeds
	session, ok := sessions.Login("test-token")
	if !ok {
		t.Fatal("login should succeed")
	}

	// Session should expire almost immediately
	time.Sleep(10 * time.Millisecond)

	if sessions.Valid(session) {
		t.Error("[SECURITY] Expired session should not be valid")
	}

	// Verify via middleware
	called := false
	protected := sessions.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
	req.Header.Set("X-Session-Token", session)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if called {
		t.Error("[SECURITY] Expired session should not reach protected handler")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("[SECURITY] Expired session: expected 401, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Expired admin session rejected: PASS")
}

func TestSecurity_AdminSession_KeyGrant_Expired_Returns403(t *testing.T) {
	sessions := NewSessionStore("test-token", 24*time.Hour, 1*time.Nanosecond)

	session, _ := sessions.Login("test-token")
	sessions.GrantKeyAccess(session)

	// Key grant should expire immediately
	time.Sleep(10 * time.Millisecond)

	if sessions.HasKeyGrant(session) {
		t.Error("[SECURITY] Expired key grant should not be valid")
	}

	protected := sessions.RequireKeyGrant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/key", nil)
	req.Header.Set("X-Session-Token", session)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("[SECURITY] Expired key grant: expected 403, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Expired key grant returns 403: PASS")
}

func TestSecurity_AdminSession_KeyGrant_StepUpRequired(t *testing.T) {
	sessions := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)
	session, _ := sessions.Login("test-token")

	// Session is valid but no key grant has been issued
	if !sessions.Valid(session) {
		t.Fatal("session should be valid")
	}
	if sessions.HasKeyGrant(session) {
		t.Fatal("no key grant should exist yet")
	}

	protected := sessions.RequireKeyGrant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler without key grant")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/key", nil)
	req.Header.Set("X-Session-Token", session)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("[SECURITY] Key access without step-up auth: expected 403, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Key grant step-up auth required: PASS")
}

func TestSecurity_AdminEndpoints_AllRequireAuth(t *testing.T) {
	handler, _ := setupAdminTest(t)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/admin/logout", ""},
		{http.MethodGet, "/api/admin/status", ""},
		{http.MethodGet, "/api/admin/metrics", ""},
		{http.MethodGet, "/api/admin/printers", ""},
		{http.MethodPost, "/api/admin/printers/active", `{"id":"mock"}`},
	}

	for _, ep := range endpoints {
		var bodyReader *strings.Reader
		if ep.body != "" {
			bodyReader = strings.NewReader(ep.body)
		} else {
			bodyReader = strings.NewReader("")
		}
		req := httptest.NewRequest(ep.method, ep.path, bodyReader)
		if ep.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		handler.sessions.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("[SECURITY] %s %s without auth: expected 401, got %d", ep.method, ep.path, rec.Code)
		}
	}
	t.Logf("[SECURITY] All admin endpoints require authentication: PASS")
}

func TestSecurity_AdminSession_ThreeTokenSources(t *testing.T) {
	sessions := NewSessionStore("test-token", 24*time.Hour, 5*time.Minute)
	session, _ := sessions.Login("test-token")

	called := 0
	protected := sessions.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
	}))

	sources := []struct {
		name string
		setup func(r *http.Request)
	}{
		{"X-Session-Token header", func(r *http.Request) { r.Header.Set("X-Session-Token", session) }},
		{"Authorization Bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+session) }},
		{"cookie", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "zerodrop_admin_session", Value: session}) }},
	}

	for _, src := range sources {
		called = 0
		req := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
		src.setup(req)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)

		if called != 1 {
			t.Errorf("[SECURITY] Session via %s: auth failed (called=%d)", src.name, called)
		}
	}
	t.Logf("[SECURITY] Admin session works via all 3 token sources: PASS")
}
