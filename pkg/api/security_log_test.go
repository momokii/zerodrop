package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLogs redirects log output to a buffer for the duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })
	return &buf
}

func TestSecurity_Log_NoPayloadContentOnError(t *testing.T) {
	buf := captureLogs(t)

	server, _ := newSecurityTestServer(t)

	// Submit a payload that triggers the "too long" error path
	uniqueMark := "UNIQUE-PAYLOAD-MARKER-XYZ-789"
	longPayload := uniqueMark + strings.Repeat("A", 400)
	body, _ := json.Marshal(DropRequest{Payload: longPayload})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	output := buf.String()

	// The error log path IS exercised (logs "Payload too long: N chars")
	// Verify the actual payload content is NOT in the log
	if strings.Contains(output, uniqueMark) {
		t.Errorf("[SECURITY] Payload content leaked in error log:\n%s", output)
	}
	// Verify the log DID produce output (otherwise the test is trivial)
	if strings.TrimSpace(output) == "" {
		t.Fatal("[SECURITY] No log output captured — test cannot verify absence of secrets")
	}
	t.Logf("[SECURITY] Error log contains no payload content (only length): PASS")
}

func TestSecurity_Log_NoPayloadContentOnInvalidBase64(t *testing.T) {
	buf := captureLogs(t)

	server, _ := newSecurityTestServer(t)

	// Submit invalid base64 that triggers the error log path
	invalidPayload := "ZD1:!!!not-valid-base64!!!"
	body, _ := json.Marshal(DropRequest{Payload: invalidPayload})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	output := buf.String()

	// The error log path IS exercised (logs "Invalid base64 after ZD1: prefix: ...")
	// Verify the payload itself is not in the log
	if strings.Contains(output, invalidPayload) {
		t.Errorf("[SECURITY] Invalid payload string leaked in error log:\n%s", output)
	}
	if strings.TrimSpace(output) == "" {
		t.Fatal("[SECURITY] No log output captured — test cannot verify absence of secrets")
	}
	t.Logf("[SECURITY] Error log contains no payload content (only parse error): PASS")
}

func TestSecurity_Log_FailedLogin_IsSilent(t *testing.T) {
	buf := captureLogs(t)

	handler, _ := setupAdminTest(t)

	// Failed login attempt
	wrongToken := "known-wrong-token-value-12345"
	body := `{"token":"` + wrongToken + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	output := buf.String()

	// handleLogin does NOT log on failure — this IS the security property.
	// No log output = no possible token leakage. If someone adds logging to
	// handleLogin in the future, this test will catch it and force review.
	if strings.TrimSpace(output) != "" {
		// If output appeared, verify no secrets are in it
		if strings.Contains(output, wrongToken) {
			t.Errorf("[SECURITY] Attempted (wrong) admin token found in log output:\n%s", output)
		}
		if strings.Contains(output, "test-admin-token") {
			t.Errorf("[SECURITY] Actual admin token found in log output:\n%s", output)
		}
		t.Logf("[SECURITY] Failed login produced log output (review for secret leakage): WARN")
	} else {
		t.Logf("[SECURITY] Failed login produces no log output (silent = no token leakage possible): PASS")
	}
}

func TestSecurity_Log_SuccessfulOps_ProduceNoSecretLogs(t *testing.T) {
	buf := captureLogs(t)

	server, _ := newSecurityTestServer(t)
	handler, _ := setupAdminTest(t)

	// Successful login
	body := `{"token":"test-admin-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	// Extract session token from response
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	sessionToken := resp["session"]

	// Successful drop submission
	secretPlaintext := "SENSITIVE-DROP-CONTENT-456"
	secretPayload := "ZD1:" + base64.StdEncoding.EncodeToString([]byte(secretPlaintext))
	dropBody, _ := json.Marshal(DropRequest{Payload: secretPayload})
	dropReq := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(dropBody)))
	dropReq.Header.Set("Content-Type", "application/json")
	dropRec := httptest.NewRecorder()
	server.router.ServeHTTP(dropRec, dropReq)

	output := buf.String()

	// Both success paths are silent by design (no log.Printf on happy paths).
	// This is a defense-in-depth test: if logging is ever added to these paths,
	// this test will catch any secret leakage.
	if strings.TrimSpace(output) != "" {
		// Log output appeared — verify no secrets are in it
		if sessionToken != "" && strings.Contains(output, sessionToken) {
			t.Errorf("[SECURITY] Session token found in log output after login")
		}
		if strings.Contains(output, "test-admin-token") {
			t.Errorf("[SECURITY] Admin token found in log output")
		}
		if strings.Contains(output, secretPayload) || strings.Contains(output, secretPlaintext) {
			t.Errorf("[SECURITY] Payload content found in log output")
		}
		t.Logf("[SECURITY] Success paths produced log output (reviewed — no secrets found): WARN")
	} else {
		t.Logf("[SECURITY] Success paths produce no log output (silent = no secret leakage possible): PASS")
	}
}
