package api

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/zerodrop/terminal/pkg/config"
	"github.com/zerodrop/terminal/pkg/printer"
	"github.com/zerodrop/terminal/pkg/spooler"
)

// tlsErrorFilter discards Go runtime log lines about expected TLS handshake
// failures (e.g. clients that don't trust the self-signed cert, or HTTP
// clients hitting the HTTPS port). These are noisy and non-actionable when
// using a self-signed development certificate.
type tlsErrorFilter struct {
	w io.Writer
}

func (f *tlsErrorFilter) Write(p []byte) (int, error) {
	s := string(p)
	if strings.Contains(s, "tls:") {
		return len(p), nil // discard
	}
	return f.w.Write(p)
}

type rateLimiter struct {
	visitors map[string]*visitorInfo
	mu       sync.Mutex
	limit    int
}

type visitorInfo struct {
	count       int
	windowStart time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitorInfo),
		limit:    limit,
	}
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.windowStart) > time.Hour {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, ok := rl.visitors[ip]

	if !ok || now.Sub(v.windowStart) > time.Hour {
		rl.visitors[ip] = &visitorInfo{count: 1, windowStart: now}
		return true
	}

	v.count++
	return v.count <= rl.limit
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r.RemoteAddr)
		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded. Try again later.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractIP(remoteAddr string) string {
	idx := strings.LastIndex(remoteAddr, ":")
	if idx == -1 {
		return remoteAddr
	}
	return remoteAddr[:idx]
}

// Server handles HTTP requests for the ZeroDrop API
type Server struct {
	config    *config.Config
	spooler   chan []byte
	printer   interface{}
	router    *mux.Router
	apiRouter *mux.Router // subrouter for /api/* — admin routes register here
	admin     *AdminHandler
	sessions  *SessionStore
}

// DropRequest represents the JSON payload for /drop endpoint
type DropRequest struct {
	Payload string `json:"payload"`
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, spooler chan []byte, printer interface{}) *Server {
	s := &Server{
		config:  cfg,
		spooler: spooler,
		printer: printer,
		router:  mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

// EnableAdmin wires up admin dashboard routes. Call after NewServer if
// AdminToken is configured.
func (s *Server) EnableAdmin(
	splr *spooler.Spooler,
	printerMgr *printer.PrinterManager,
	publicKeyPath, privateKeyPath, keyFingerprint string,
) {
	s.sessions = NewSessionStore(s.config.AdminToken)
	s.admin = NewAdminHandler(
		s.sessions, splr, printerMgr,
		publicKeyPath, privateKeyPath, keyFingerprint,
	)
	s.setupAdminRoutes()
	s.FinalizeRoutes()
}

// FinalizeRoutes registers the API catch-all AFTER all other routes (admin
// included) so they take priority. Must be called after all route setup is
// complete — called from EnableAdmin and from main.go when admin is disabled.
func (s *Server) FinalizeRoutes() {
	s.apiRouter.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "not found",
		})
	}))
}

// setupAdminRoutes registers all admin API routes on the /api subrouter
// (before the catch-all) so they don't get intercepted.
func (s *Server) setupAdminRoutes() {
	adminRouter := s.apiRouter.PathPrefix("/admin").Subrouter()

	// Login is public (no auth required)
	adminRouter.Handle("/login", http.HandlerFunc(s.admin.handleLogin)).Methods(http.MethodPost)

	// All other admin routes require auth
	adminAuth := adminRouter.NewRoute().Subrouter()
	adminAuth.Use(s.sessions.RequireAuth)
	adminAuth.Handle("/status", http.HandlerFunc(s.admin.handleStatus)).Methods(http.MethodGet)
	adminAuth.Handle("/metrics", http.HandlerFunc(s.admin.handleMetrics)).Methods(http.MethodGet)
	adminAuth.Handle("/printers", http.HandlerFunc(s.admin.handleListPrinters)).Methods(http.MethodGet)
	adminAuth.Handle("/printers/active", http.HandlerFunc(s.admin.handleSetActivePrinter)).Methods(http.MethodPost)
	adminAuth.Handle("/key", http.HandlerFunc(s.admin.handleKeyDownload)).Methods(http.MethodGet)
	adminAuth.Handle("/key/qr", http.HandlerFunc(s.admin.handleKeyQRDownload)).Methods(http.MethodGet)
	adminAuth.Handle("/key/rotate", http.HandlerFunc(s.admin.handleKeyRotate)).Methods(http.MethodPost)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Rate limiter for API endpoints
	rl := newRateLimiter(s.config.RateLimitRequestsPerHour)

	// Rate-limited routes (key and drop only — health is excluded so Docker
	// HEALTHCHECK doesn't get 429 after a few checks)
	apiRouter := s.router.PathPrefix("/api").Subrouter()
	s.apiRouter = apiRouter // saved so admin routes register here (before the catch-all)
	apiRouter.Handle("/key", rl.middleware(http.HandlerFunc(s.handleGetKey))).Methods(http.MethodGet)
	apiRouter.Handle("/drop", rl.middleware(http.HandlerFunc(s.handleDrop))).Methods(http.MethodPost)
	apiRouter.Handle("/health", http.HandlerFunc(s.handleHealth)).Methods(http.MethodGet)

	// Legacy routes without /api prefix
	s.router.Handle("/key", rl.middleware(http.HandlerFunc(s.handleGetKey))).Methods(http.MethodGet)
	s.router.Handle("/drop", rl.middleware(http.HandlerFunc(s.handleDrop))).Methods(http.MethodPost)
	s.router.Handle("/health", http.HandlerFunc(s.handleHealth)).Methods(http.MethodGet)

	// Serve static directory (reader.html, jsqr.min.js, etc.)
	s.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))),
	)

	// Serve reader.html with no-cache headers and a cache-busting alias
	serveReader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, "./static/reader.html")
	})
	s.router.Handle("/reader.html", serveReader)
	s.router.Handle("/reader-v2.html", serveReader) // cache-busting alias

	// Serve frontend SPA (all other routes)
	spaHandler := spaHandler{staticFilePath: "./frontend/dist", indexPath: "index.html"}
	s.router.PathPrefix("/").Handler(spaHandler)
}

// handleGetKey returns the public key in PEM format
func (s *Server) handleGetKey(w http.ResponseWriter, r *http.Request) {
	// Read public key from file
	publicKeyPEM, err := os.ReadFile(s.config.PublicKeyPath)
	if err != nil {
		http.Error(w, "Failed to read public key", http.StatusInternalServerError)
		log.Printf("Error reading public key: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(publicKeyPEM)
}

// handleHealth returns server health status
// Returns HTTP 200 if the system is operational, HTTP 503 if printer is unavailable
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":  "healthy",
		"service": "zerodrop-terminal",
	}

	// Check printer availability
	if checker, ok := s.printer.(interface{ IsAvailable() bool }); ok {
		available := checker.IsAvailable()
		printerHealth := map[string]interface{}{
			"available": available,
		}

		// Add printer type if HealthCheck is available
		if hc, ok := s.printer.(interface{ HealthCheck() map[string]interface{} }); ok {
			printerInfo := hc.HealthCheck()
			for k, v := range printerInfo {
				printerHealth[k] = v
			}
		}

		health["printer"] = printerHealth

		if !available {
			health["status"] = "unhealthy"
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			json.NewEncoder(w).Encode(health)
			return
		}
	}

	// Also add printer info if available and healthy
	if hc, ok := s.printer.(interface{ HealthCheck() map[string]interface{} }); ok {
		health["printer"] = hc.HealthCheck()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// handleDrop accepts an encrypted payload and queues it for printing
func (s *Server) handleDrop(w http.ResponseWriter, r *http.Request) {
	// Parse JSON request
	var req DropRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		log.Printf("Error decoding request: %v", err)
		return
	}

	// Validate payload length (including ZD1: prefix, max 400 chars)
	if len(req.Payload) > 400 {
		http.Error(w, "Payload exceeds 400 character limit", http.StatusBadRequest)
		log.Printf("Payload too long: %d chars", len(req.Payload))
		return
	}

	// Validate ZD1: prefix (if present) and remaining base64
	var ciphertext []byte
	if strings.HasPrefix(req.Payload, "ZD1:") {
		// Strip prefix and validate the rest
		ciphertextStr := strings.TrimPrefix(req.Payload, "ZD1:")
		var err error
		ciphertext, err = base64.StdEncoding.DecodeString(ciphertextStr)
		if err != nil {
			http.Error(w, "Invalid base64 in ciphertext after ZD1: prefix", http.StatusBadRequest)
			log.Printf("Invalid base64 after ZD1: prefix: %v", err)
			return
		}
	} else {
		// No prefix, validate entire payload as base64
		var err error
		ciphertext, err = base64.StdEncoding.DecodeString(req.Payload)
		if err != nil {
			http.Error(w, "Payload must be valid base64", http.StatusBadRequest)
			log.Printf("Invalid base64 in payload: %v", err)
			return
		}
	}

	// Convert decoded ciphertext to bytes
	payloadBytes := ciphertext

	// Push to spooler (non-blocking)
	select {
	case s.spooler <- payloadBytes:
		// Successfully queued
	default:
		// Spooler is full (shouldn't happen with buffered channel)
		http.Error(w, "Server busy, please retry", http.StatusServiceUnavailable)
		log.Printf("Spooler full, rejected request")
		return
	}

	// Return 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": "Payload queued for printing",
	})
}

// Start begins listening for HTTP requests
func (s *Server) Start(addr string) error {
	log.Printf("API server starting on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// ServeTLS starts listening for HTTPS requests using the provided
// PEM-encoded certificate and key. No temp files are used — the cert
// is loaded directly into the tls.Config.
func (s *Server) ServeTLS(addr string, certPEM, keyPEM []byte) error {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
		// Suppress expected TLS noise from clients that don't trust the
		// self-signed certificate (Docker probes, other tools, etc.)
		ErrorLog: log.New(&tlsErrorFilter{w: os.Stderr}, "", log.LstdFlags),
	}

	log.Printf("HTTPS server starting on %s", addr)
	return server.ListenAndServeTLS("", "")
}

// spaHandler implements the http.Handler interface to serve a Single Page Application (SPA)
type spaHandler struct {
	staticFilePath string
	indexPath      string
}

// ServeHTTP serves static files and falls back to index.html for SPA routing.
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the absolute path to prevent directory traversal
	path, err := filepath.Abs(filepath.Join(h.staticFilePath, r.URL.Path))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the file exists
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// File doesn't exist, serve index.html for SPA routing
		// Prevent browser caching so updated frontend builds are always fetched
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, filepath.Join(h.staticFilePath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serve the static file
	http.FileServer(http.Dir(h.staticFilePath)).ServeHTTP(w, r)
}
