package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zerodrop/terminal/pkg/config"
	"github.com/zerodrop/terminal/pkg/printer"
)

func newRateLimitedServer(t *testing.T, limit int) (*Server, chan []byte) {
	t.Helper()
	cfg := &config.Config{
		PrinterType:              "mock",
		RateLimitRequestsPerHour: limit,
		RateLimitBurst:           limit,
		PublicKeyPath:            filepath.Join(t.TempDir(), "public_key.pem"),
	}
	spoolerCh := make(chan []byte, 10)
	mockPrinter := printer.NewMockPrinter()
	server := NewServer(cfg, spoolerCh, mockPrinter)
	server.FinalizeRoutes()
	return server, spoolerCh
}

func TestSecurity_RateLimit_ExceedsLimit_Returns429(t *testing.T) {
	const limit = 5
	server, _ := newRateLimitedServer(t, limit)

	// Drain any spooler messages
	dropBody := func() string {
		b, _ := json.Marshal(DropRequest{Payload: "ZD1:" + strings.Repeat("A", 4)})
		return string(b)
	}

	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(dropBody()))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("[SECURITY] request %d/%d was rate-limited too early", i+1, limit)
		}
	}

	// 6th request should be rate-limited
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(dropBody()))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("[SECURITY] 6th request from same IP: expected 429, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Rate limiting enforces limit (%d req/hr): PASS", limit)
}

func TestSecurity_RateLimit_HealthEndpoint_Exempt(t *testing.T) {
	const limit = 3
	server, _ := newRateLimitedServer(t, limit)

	// Exhaust rate limit on /drop
	dropBody := func() string {
		b, _ := json.Marshal(DropRequest{Payload: "ZD1:" + strings.Repeat("A", 4)})
		return string(b)
	}
	for i := 0; i < limit+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(dropBody()))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
	}

	// /health should still work even after rate limit is exhausted
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Errorf("[SECURITY] /health request %d was rate-limited — health endpoint must be exempt", i+1)
		}
	}
	t.Logf("[SECURITY] /health exempt from rate limiting: PASS")
}

func TestSecurity_RateLimit_DifferentIPs_Independent(t *testing.T) {
	const limit = 3
	server, _ := newRateLimitedServer(t, limit)

	dropBody := func() string {
		b, _ := json.Marshal(DropRequest{Payload: "ZD1:" + strings.Repeat("B", 4)})
		return string(b)
	}

	// Exhaust rate limit for IP 1
	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(dropBody()))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
	}

	// Verify IP 1 is rate-limited
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(dropBody()))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("[SECURITY] IP 1 should be rate-limited, got %d", rec.Code)
	}

	// IP 2 should NOT be rate-limited
	req2 := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(dropBody()))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "192.168.1.2:1234"
	rec2 := httptest.NewRecorder()
	server.router.ServeHTTP(rec2, req2)
	if rec2.Code == http.StatusTooManyRequests {
		t.Errorf("[SECURITY] IP 2 should NOT be rate-limited, got 429")
	}
	t.Logf("[SECURITY] Rate limits per-IP independence: PASS")
}

func TestSecurity_RateLimit_KeyEndpoint_Counted(t *testing.T) {
	const limit = 3
	server, _ := newRateLimitedServer(t, limit)

	// Create a dummy public key file so /key doesn't 500
	tmpDir := filepath.Dir(server.config.PublicKeyPath)
	pubKeyPath := filepath.Join(tmpDir, "public_key.pem")
	server.config.PublicKeyPath = pubKeyPath
	_ = writeTestPublicKey(t, pubKeyPath)

	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/key", nil)
		req.RemoteAddr = "172.16.0.1:1234"
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
	}

	// (limit+1)th request should be rate-limited
	req := httptest.NewRequest(http.MethodGet, "/api/key", nil)
	req.RemoteAddr = "172.16.0.1:1234"
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("[SECURITY] /key endpoint should be rate-limited: expected 429, got %d", rec.Code)
	}
	t.Logf("[SECURITY] /key endpoint is rate-limited: PASS")
}

func TestSecurity_AdminLogin_ExceedsLimit_Returns429(t *testing.T) {
	handler, _ := setupAdminTest(t)

	// The admin login rate limiter allows 10 attempts per 15 min.
	// Send 10 invalid login attempts, then verify the 11th is blocked.
	for i := 0; i < 10; i++ {
		body := `{"token":"wrong"}`
		req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.10.10.10:1234"
		rec := httptest.NewRecorder()
		handler.handleLogin(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("[SECURITY] attempt %d/10 was rate-limited too early", i+1)
		}
	}

	// 11th attempt should be blocked
	body := `{"token":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.10.10.10:1234"
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("[SECURITY] 11th admin login attempt: expected 429, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Admin login rate limit (10/15min): PASS")
}

func TestSecurity_AdminLogin_DifferentIP_Independent(t *testing.T) {
	handler, _ := setupAdminTest(t)

	// Exhaust login attempts for IP 1
	for i := 0; i < 10; i++ {
		body := `{"token":"wrong"}`
		req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.10.10.10:1234"
		rec := httptest.NewRecorder()
		handler.handleLogin(rec, req)
	}

	// Verify IP 1 is blocked
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"token":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.10.10.10:1234"
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1 should be blocked, got %d", rec.Code)
	}

	// IP 2 should NOT be blocked — attempt a valid login
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"token":"test-admin-token"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "10.10.10.20:1234"
	rec2 := httptest.NewRecorder()
	handler.handleLogin(rec2, req2)
	if rec2.Code == http.StatusTooManyRequests {
		t.Errorf("[SECURITY] IP 2 should NOT be rate-limited, got 429")
	}
	t.Logf("[SECURITY] Admin login rate limit per-IP: PASS")
}

// writeTestPublicKey writes a minimal valid SPKI PEM for testing /key endpoint.
func writeTestPublicKey(t *testing.T, path string) error {
	t.Helper()
	return writeFile(path, []byte("-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VuAyEA\n-----END PUBLIC KEY-----\n"), 0644)
}

func writeFile(path string, data []byte, perm uint32) error {
	return os.WriteFile(path, data, fs.FileMode(perm))
}
