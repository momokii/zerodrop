package api

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/zerodrop/terminal/pkg/config"
)

// Server handles HTTP requests for the ZeroDrop API
type Server struct {
	config  *config.Config
	spooler chan []byte
	printer interface{}
	router  *mux.Router
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

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// API routes
	apiRouter := s.router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/key", s.handleGetKey).Methods(http.MethodGet)
	apiRouter.HandleFunc("/drop", s.handleDrop).Methods(http.MethodPost)
	apiRouter.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// Legacy API routes (without /api prefix for backward compatibility)
	s.router.HandleFunc("/key", s.handleGetKey).Methods(http.MethodGet)
	s.router.HandleFunc("/drop", s.handleDrop).Methods(http.MethodPost)
	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// Serve static directory (reader.html, jsqr.min.js, etc.)
	s.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))),
	)

	// Also serve reader.html at root level for convenience
	s.router.Handle("/reader.html", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/reader.html")
	}))

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

// spaHandler implements the http.Handler interface to serve a Single Page Application (SPA)
type spaHandler struct {
	staticFilePath string
	indexPath      string
}

// ServeHTTP serves static files and falls back to index.html for SPA routing
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
		http.ServeFile(w, r, filepath.Join(h.staticFilePath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serve the static file
	http.FileServer(http.Dir(h.staticFilePath)).ServeHTTP(w, r)
}
