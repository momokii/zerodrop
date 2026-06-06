package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(
	sessions *SessionStore,
	splr *spooler.Spooler,
	printerMgr *printer.PrinterManager,
	publicKeyPath, privateKeyPath, keyFingerprint string,
) *AdminHandler {
	return &AdminHandler{
		sessions:       sessions,
		spoolerMetrics: splr.GetMetrics,
		printerMgr:     printerMgr,
		publicKeyPath:  publicKeyPath,
		privateKeyPath: privateKeyPath,
		startTime:      time.Now(),
		keyFingerprint: keyFingerprint,
		keyGeneratedAt: time.Now(),
	}
}

func (h *AdminHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	session, ok := h.sessions.Login(req.Token)
	if !ok {
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "zerodrop_admin_session",
		Value:    session,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AdminHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":        "1.1.0",
		"uptime_seconds": time.Since(h.startTime).Seconds(),
		"key": map[string]interface{}{
			"fingerprint":        h.keyFingerprint,
			"generated_at":       h.keyGeneratedAt.Format(time.RFC3339),
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
		"last_print_ms":      m.LastPrintDuration.Milliseconds(),
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

func (h *AdminHandler) handleKeyDownload(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.privateKeyPath)
	if err != nil {
		http.Error(w, `{"error":"private key not found on disk"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=private_key.pem")
	w.Write(data)
}

func (h *AdminHandler) handleKeyQRDownload(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename != "private_key_qr.png" && filename != "private_key_jwk_qr.png" {
		http.Error(w, `{"error":"invalid file"}`, http.StatusBadRequest)
		return
	}
	path := filepath.Join(filepath.Dir(h.privateKeyPath), filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, `{"error":"QR file not found"}`, http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, path)
}

func (h *AdminHandler) handleKeyRotate(w http.ResponseWriter, r *http.Request) {
	publicKeyPath := h.publicKeyPath
	// Delete private key to force regeneration on next restart
	os.Remove(filepath.Join(filepath.Dir(publicKeyPath), "private_key.pem"))
	os.Remove(publicKeyPath)
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
