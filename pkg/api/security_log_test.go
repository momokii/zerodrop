package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurity_Log_NoPrivateKeyMaterial(t *testing.T) {
	// Capture all log output
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	handler, _ := setupAdminTest(t)

	// Login — this generates a session token internally
	body := `{"token":"test-admin-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	output := buf.String()

	// The session token value should not appear in logs
	// Extract the session token from the response
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	sessionToken := resp["session"]

	if sessionToken != "" && strings.Contains(output, sessionToken) {
		t.Errorf("[SECURITY] Session token (%s...) found in log output", sessionToken[:8])
	}
	t.Logf("[SECURITY] No session token in logs: PASS")
}

func TestSecurity_Log_NoPayloadContent(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	server, _ := newSecurityTestServer(t)

	// Submit a known payload
	secretPayload := "ZD1:VEhJUy1JUy1BLVNFX0NSRVQtVEhBVC1XT1JLUw=="
	body, _ := json.Marshal(DropRequest{Payload: secretPayload})
	req := httptest.NewRequest(http.MethodPost, "/api/drop", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	output := buf.String()
	if strings.Contains(output, secretPayload) {
		t.Errorf("[SECURITY] Encrypted payload content found in log output:\n%s", output)
	}
	if strings.Contains(output, "THIS-IS-A-SECRET-THAT-WORKS") {
		t.Errorf("[SECURITY] Decoded plaintext found in log output:\n%s", output)
	}
	t.Logf("[SECURITY] No payload content in logs: PASS")
}

func TestSecurity_Log_NoSessionToken(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	handler, _ := setupAdminTest(t)

	// Successful login
	body := `{"token":"test-admin-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	output := buf.String()
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	token := resp["session"]

	if token != "" && strings.Contains(output, token) {
		t.Errorf("[SECURITY] Session token value found in log output")
	}
	t.Logf("[SECURITY] No session token in logs: PASS")
}

func TestSecurity_Log_NoAdminToken(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	handler, _ := setupAdminTest(t)

	// Failed login with a known wrong token
	body := `{"token":"known-wrong-token-value-12345"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.handleLogin(rec, req)

	output := buf.String()
	if strings.Contains(output, "known-wrong-token-value-12345") {
		t.Errorf("[SECURITY] Attempted admin token found in log output:\n%s", output)
	}
	if strings.Contains(output, "test-admin-token") {
		t.Errorf("[SECURITY] Actual admin token found in log output:\n%s", output)
	}
	t.Logf("[SECURITY] No admin token in logs: PASS")
}
