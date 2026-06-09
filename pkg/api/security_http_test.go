package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zerodrop/terminal/pkg/printer"
	"github.com/zerodrop/terminal/pkg/spooler"
)

func TestSecurity_AdminRoutes_BehindAuth(t *testing.T) {
	server, _ := newSecurityTestServer(t)
	sessions := NewSessionStore("sec-test-token", 24*time.Hour, 5*time.Minute)
	server.sessions = sessions
	server.admin = NewAdminHandler(
		sessions,
		spooler.NewSpooler(10, printer.NewMockPrinter()),
		printer.NewPrinterManager(printer.NewMockPrinter(), printer.PrinterInfo{ID: "mock", Name: "Mock", Type: "mock"}),
		filepath.Join(t.TempDir(), "pub.pem"),
		filepath.Join(t.TempDir(), "priv.pem"),
		"test-fp",
	)
	server.setupAdminRoutes()
	server.FinalizeRoutes()

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/status"},
		{http.MethodGet, "/api/admin/metrics"},
		{http.MethodGet, "/api/admin/printers"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Errorf("[SECURITY] %s %s returned 200 without auth", ep.method, ep.path)
		}
	}
	t.Logf("[SECURITY] Admin routes behind auth middleware: PASS")
}

func TestSecurity_HealthEndpoint_NoAuth_Required(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	// Health should return 200 or 503, but never 401
	if rec.Code == http.StatusUnauthorized {
		t.Error("[SECURITY] /health endpoint should not require authentication")
	}
	t.Logf("[SECURITY] /health no auth required: PASS (status %d)", rec.Code)
}

func TestSecurity_SPABackend_DotEnvNotServed(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	// /.env should fall through to SPA handler → index.html (not serve the file)
	req := httptest.NewRequest(http.MethodGet, "/.env", nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	// Should NOT return 200 with file contents, and should NOT return the .env file
	// Either returns index.html (SPA fallback) or 404 — both are safe
	body := rec.Body.String()
	if strings.Contains(body, "ADMIN_TOKEN") || strings.Contains(body, "DATABASE_URL") {
		t.Error("[SECURITY] /.env endpoint served actual .env file contents!")
	}
	t.Logf("[SECURITY] /.env not served as raw file: PASS (status %d)", rec.Code)
}

func TestSecurity_StaticFiles_NoPathTraversal(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	// Path traversal attempts should not expose files outside static/
	traversalPaths := []string{
		"/../../../etc/passwd",
		"/static/../../../etc/passwd",
		"/api/../../../etc/passwd",
	}

	for _, path := range traversalPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		body := rec.Body.String()
		if strings.Contains(body, "root:x:") {
			t.Errorf("[SECURITY] Path traversal succeeded: %s exposed /etc/passwd", path)
		}
	}
	t.Logf("[SECURITY] Path traversal attempts blocked: PASS")
}
