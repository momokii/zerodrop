package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zerodrop/terminal/pkg/printer"
	"github.com/zerodrop/terminal/pkg/spooler"
)

func setupAdminTest(t *testing.T) (*AdminHandler, string) {
	t.Helper()
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "private_key.pem")
	os.WriteFile(privKeyPath, []byte("test-key-data"), 0600)

	mockPrinter := printer.NewMockPrinter()
	splr := spooler.NewSpooler(10, mockPrinter)
	pm := printer.NewPrinterManager(mockPrinter, printer.PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})

	sessions := NewSessionStore("test-admin-token", 24*time.Hour)
	handler := NewAdminHandler(
		sessions, splr, pm,
		filepath.Join(tmpDir, "public_key.pem"),
		privKeyPath,
		"abc123fingerprint",
	)
	return handler, tmpDir
}

func adminCookie(t *testing.T, sessions *SessionStore) *http.Cookie {
	t.Helper()
	session, _ := sessions.Login("test-admin-token")
	return &http.Cookie{
		Name:  "zerodrop_admin_session",
		Value: session,
	}
}

func TestAdminLogin_Success(t *testing.T) {
	handler, _ := setupAdminTest(t)

	body := `{"token":"test-admin-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Should set cookie
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "zerodrop_admin_session" {
			found = true
			if !c.HttpOnly {
				t.Error("cookie should be HttpOnly")
			}
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func TestAdminLogin_InvalidToken(t *testing.T) {
	handler, _ := setupAdminTest(t)

	body := `{"token":"wrong-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminStatus(t *testing.T) {
	handler, _ := setupAdminTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
	rec := httptest.NewRecorder()

	handler.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["version"] != "1.1.0" {
		t.Errorf("expected version 1.1.0, got %v", resp["version"])
	}
}

func TestAdminMetrics(t *testing.T) {
	handler, _ := setupAdminTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics", nil)
	rec := httptest.NewRecorder()

	handler.handleMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if _, ok := resp["queue_depth"]; !ok {
		t.Error("expected queue_depth in response")
	}
}

func TestAdminListPrinters(t *testing.T) {
	handler, _ := setupAdminTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/printers", nil)
	rec := httptest.NewRecorder()

	handler.handleListPrinters(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	printers, ok := resp["printers"].([]interface{})
	if !ok || len(printers) == 0 {
		t.Error("expected at least one printer (mock)")
	}
}

func TestAdminKeyDownload(t *testing.T) {
	handler, _ := setupAdminTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/key", nil)
	rec := httptest.NewRecorder()

	handler.handleKeyDownload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if rec.Body.String() != "test-key-data" {
		t.Errorf("expected private key data, got %s", rec.Body.String())
	}
}
