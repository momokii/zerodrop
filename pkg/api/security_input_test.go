package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zerodrop/terminal/pkg/config"
	"github.com/zerodrop/terminal/pkg/printer"
)

// newSecurityTestServer creates a Server wired with mock config and printer
// for security testing. Returns the server and the raw spooler channel so
// tests can verify what the server actually queued.
func newSecurityTestServer(t *testing.T) (*Server, chan []byte) {
	t.Helper()
	cfg := &config.Config{
		PrinterType:              "mock",
		RateLimitRequestsPerHour: 100,
		RateLimitBurst:           10,
		PublicKeyPath:            filepath.Join(t.TempDir(), "public_key.pem"),
	}
	spoolerCh := make(chan []byte, 10)
	mockPrinter := printer.NewMockPrinter()
	server := NewServer(cfg, spoolerCh, mockPrinter)
	server.FinalizeRoutes()
	return server, spoolerCh
}

func TestSecurity_Drop_ExceedsMaxPayload_Returns400(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	// 401 chars exceeds the 400-char limit
	longPayload := "ZD1:" + strings.Repeat("A", 397) // 4 + 397 = 401
	body, _ := json.Marshal(DropRequest{Payload: longPayload})

	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("[SECURITY] POST /drop with %d-char payload: expected 400, got %d", len(longPayload), rec.Code)
	}
	t.Logf("[SECURITY] Drop max payload limit: PASS (401 chars rejected)")
}

func TestSecurity_Drop_ExactMaxPayload_Returns202(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	// Build a valid ZD1: payload that is exactly 400 chars.
	// ZD1: = 4 chars, so we need 396 chars of valid base64.
	// base64 encodes 3 bytes -> 4 chars, so 396 chars = 297 bytes.
	raw := make([]byte, 297)
	for i := range raw {
		raw[i] = byte(i % 256)
	}
	b64 := base64.StdEncoding.EncodeToString(raw)
	payload := "ZD1:" + b64
	if len(payload) != 400 {
		t.Fatalf("test setup: payload length %d (expected 400)", len(payload))
	}

	body, _ := json.Marshal(DropRequest{Payload: payload})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("[SECURITY] POST /drop with exactly 400 chars: expected 202, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Drop exact max payload accepted: PASS (400 chars -> 202)")
}

func TestSecurity_Drop_InvalidBase64_Returns400(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	body, _ := json.Marshal(DropRequest{Payload: "ZD1:this-is-not-valid-base64!!!"})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("[SECURITY] POST /drop with invalid base64: expected 400, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Drop invalid base64 rejected: PASS")
}

func TestSecurity_Drop_ZD1Prefix_ValidBase64_Returns202(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	ciphertext := base64.StdEncoding.EncodeToString([]byte("test-ciphertext-data"))
	payload := "ZD1:" + ciphertext

	body, _ := json.Marshal(DropRequest{Payload: payload})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("[SECURITY] POST /drop ZD1: + valid base64: expected 202, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Drop ZD1: + valid base64 accepted: PASS")
}

func TestSecurity_Drop_ZD1Prefix_InvalidBase64_Returns400(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	body, _ := json.Marshal(DropRequest{Payload: "ZD1:!!!notbase64!!!"})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("[SECURITY] POST /drop ZD1: + invalid base64: expected 400, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Drop ZD1: + invalid base64 rejected: PASS")
}

func TestSecurity_Drop_NoPrefix_ValidBase64_Returns202(t *testing.T) {
	server, _ := newSecurityTestServer(t)

	ciphertext := base64.StdEncoding.EncodeToString([]byte("test-ciphertext-no-prefix"))

	body, _ := json.Marshal(DropRequest{Payload: ciphertext})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("[SECURITY] POST /drop valid base64 (no prefix): expected 202, got %d", rec.Code)
	}
	t.Logf("[SECURITY] Drop valid base64 without prefix accepted: PASS")
}

func TestSecurity_Drop_PayloadDecodedCorrectly(t *testing.T) {
	server, spoolerCh := newSecurityTestServer(t)

	// The server must decode base64 and queue the raw bytes,
	// not the raw JSON string. This proves the spooler receives
	// only the decoded ciphertext.
	secret := []byte("super-secret-data-12345")
	ciphertext := base64.StdEncoding.EncodeToString(secret)
	payload := "ZD1:" + ciphertext

	body, _ := json.Marshal(DropRequest{Payload: payload})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	// Read from spooler channel and verify it's the decoded bytes
	select {
	case queued := <-spoolerCh:
		if string(queued) != string(secret) {
			t.Errorf("[SECURITY] Spooler received wrong data:\n  got:      %q\n  expected: %q", string(queued), string(secret))
		}
		// Verify the raw payload string was NOT queued
		if string(queued) == payload {
			t.Error("[SECURITY] Spooler received the raw ZD1: string instead of decoded bytes!")
		}
		t.Logf("[SECURITY] Spooler receives decoded bytes (not raw string): PASS")
	default:
		t.Fatal("spooler channel empty - payload was not queued")
	}
}
