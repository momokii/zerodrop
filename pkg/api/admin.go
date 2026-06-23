package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zerodrop/terminal/pkg/printer"
	"github.com/zerodrop/terminal/pkg/spooler"
)

// AdminHandler handles admin API requests.
type AdminHandler struct {
	sessions       *SessionStore
	spoolerMetrics func() spooler.Metrics
	printerMgr     *printer.PrinterManager
	publicKeyPath  string
	privateKeyPath string
	startTime      time.Time
	keyFingerprint string
	keyGeneratedAt time.Time
	loginLimiter   *loginRateLimiter
}

// loginRateLimiter prevents brute-force attacks on the admin login endpoint.
type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempts
	limit    int
	window   time.Duration
}

type loginAttempts struct {
	count    int
	windowStart time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	rl := &loginRateLimiter{
		attempts: make(map[string]*loginAttempts),
		limit:    10,
		window:   15 * time.Minute,
	}
	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.mu.Lock()
			now := time.Now()
			for ip, a := range rl.attempts {
				if now.Sub(a.windowStart) > rl.window {
					delete(rl.attempts, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *loginRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	a, ok := rl.attempts[ip]
	if !ok || now.Sub(a.windowStart) > rl.window {
		rl.attempts[ip] = &loginAttempts{count: 1, windowStart: now}
		return true
	}
	a.count++
	return a.count <= rl.limit
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(
	sessions *SessionStore,
	splr *spooler.Spooler,
	printerMgr *printer.PrinterManager,
	publicKeyPath, privateKeyPath, keyFingerprint string,
) *AdminHandler {
	// Prefer the public key file's mtime as the key generation time.
	// Falls back to now() if the file doesn't exist yet.
	generatedAt := time.Now()
	if fi, err := os.Stat(publicKeyPath); err == nil {
		generatedAt = fi.ModTime()
	}

	return &AdminHandler{
		sessions:        sessions,
		spoolerMetrics:  splr.GetMetrics,
		printerMgr:      printerMgr,
		publicKeyPath:   publicKeyPath,
		privateKeyPath:  privateKeyPath,
		startTime:       time.Now(),
		keyFingerprint:  keyFingerprint,
		keyGeneratedAt:  generatedAt,
		loginLimiter:    newLoginRateLimiter(),
	}
}

func (h *AdminHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := extractIP(r.RemoteAddr)
	if !h.loginLimiter.allow(ip) {
		addRetryAfter(w, http.StatusTooManyRequests)
		APIError{Code: http.StatusTooManyRequests, Message: "Too many login attempts"}.Send(w)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrBadRequest.Send(w)
		return
	}

	session, ok := h.sessions.Login(req.Token)
	if !ok {
		ErrUnauthorized.Send(w)
		return
	}

	// Set cookie for backward compatibility (some clients may use it)
	http.SetCookie(w, &http.Cookie{
		Name:     "zerodrop_admin_session",
		Value:    session,
		Path:     "/",
		MaxAge:   int(h.sessions.MaxAge().Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	// Also return the session token in a response header and body so the SPA
	// can use header-based auth (X-Session-Token), which is more reliable
	// than cookies in fetch API.
	w.Header().Set("X-Session-Token", session)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"session": session,
	})
}

func (h *AdminHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	session := extractSession(r)
	if session != "" {
		h.sessions.Logout(session)
	}
	// Clear the cookie regardless
	http.SetCookie(w, &http.Cookie{
		Name:     "zerodrop_admin_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AdminHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Use the public key file's modification time as the key generation timestamp.
	// For newly generated keys this is ~now; for reused keys it reflects the
	// original generation time.
	generatedAt := h.keyGeneratedAt
	if fi, err := os.Stat(h.publicKeyPath); err == nil {
		generatedAt = fi.ModTime()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":               "1.1.0",
		"uptime_seconds":        time.Since(h.startTime).Seconds(),
		"key_grant_ttl_seconds": h.sessions.keyGrantTTL.Seconds(),
		"key": map[string]interface{}{
			"fingerprint":        h.keyFingerprint,
			"generated_at":       generatedAt.Format(time.RFC3339),
			"private_key_on_disk": fileExists(h.privateKeyPath),
		},
	})
}

func (h *AdminHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m := h.spoolerMetrics()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_depth":        m.CurrentDepth,
		"max_queue_depth":    m.MaxDepth,
		"total_processed":    m.TotalProcessed,
		"total_failed":       m.TotalFailed,
		"last_print_time":    m.LastPrintTime.Format(time.RFC3339),
		"last_print_ms":      elapsedSince(m.LastPrintTime, m.TotalProcessed),
		"spooler_start_time": m.StartTime.Format(time.RFC3339),
	})
}

func (h *AdminHandler) handleListPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	printers := h.printerMgr.Detect()
	active := h.printerMgr.GetActiveInfo()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"printers":       printers,
		"active_printer": active.ID,
	})
}

func (h *AdminHandler) handleSetActivePrinter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrBadRequest.Send(w)
		return
	}

	if err := h.printerMgr.SetActive(req.ID); err != nil {
		APIError{Code: http.StatusBadRequest, Message: err.Error()}.Send(w)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if err := h.printerMgr.SetActive(req.ID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AdminHandler) handleKeyGrant(w http.ResponseWriter, r *http.Request) {
	session := extractSession(r)
	if session == "" || !h.sessions.Valid(session) {
		ErrUnauthorized.Send(w)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrBadRequest.Send(w)
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(h.sessions.adminKey)) != 1 {
		ErrUnauthorized.Send(w)
		return
	}

	h.sessions.GrantKeyAccess(session)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"ttl_seconds": h.sessions.keyGrantTTL.Seconds(),
	})
}

func (h *AdminHandler) handleKeyDownload(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.privateKeyPath)
	if err != nil {
		APIError{Code: http.StatusNotFound, Message: "Private key not found on disk"}.Send(w)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=private_key.pem")
	w.Write(data)
}

func (h *AdminHandler) handleKeyQRDownload(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename != "private_key_qr.png" && filename != "private_key_jwk_qr.png" {
		APIError{Code: http.StatusBadRequest, Message: "Invalid file"}.Send(w)
		return
	}
	path := filepath.Join(filepath.Dir(h.privateKeyPath), filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		APIError{Code: http.StatusNotFound, Message: "QR file not found"}.Send(w)
		return
	}
	http.ServeFile(w, r, path)
}

func (h *AdminHandler) handleKeyRotate(w http.ResponseWriter, r *http.Request) {
	var errs []string
	if err := os.Remove(h.privateKeyPath); err != nil {
		errs = append(errs, fmt.Sprintf("private key: %v", err))
	}
	if err := os.Remove(h.publicKeyPath); err != nil {
		errs = append(errs, fmt.Sprintf("public key: %v", err))
	}

	if len(errs) > 0 {
		log.Printf("Admin key rotation failed: %s", strings.Join(errs, "; "))
		APIError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("Key rotation failed: %s", strings.Join(errs, "; "))}.Send(w)
		return
	}

	h.keyGeneratedAt = time.Now()
	log.Println("Admin triggered key rotation — keys deleted, restart required")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Keys deleted. Restart the server to generate a new key pair.",
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// elapsedSince returns the number of milliseconds elapsed since the given time.
// Returns 0 if no events have occurred (zero-value time).
func elapsedSince(t time.Time, total int64) int64 {
	if total == 0 || t.IsZero() {
		return 0
	}
	return time.Since(t).Milliseconds()
}
