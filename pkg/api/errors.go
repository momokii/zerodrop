package api

import (
	"encoding/json"
	"net/http"
)

// APIError represents a structured error response returned by all ZeroDrop API endpoints.
// Every handler should use Send() to write consistent JSON error responses instead of
// calling http.Error() or manually json.Encode() + w.WriteHeader().
type APIError struct {
	Code    int         `json:"-"`                 // HTTP status code (not serialized)
	Message string      `json:"message"`           // Human-readable error description
	Details interface{} `json:"details,omitempty"` // Optional additional context (validation errors, etc.)
}

// Send writes the APIError as a JSON response with the appropriate HTTP status code.
// Sets Content-Type to application/json.
//
// Usage:
//
//	APIError{Code: http.StatusBadRequest, Message: "Invalid payload"}.Send(w)
func (e APIError) Send(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Code)
	json.NewEncoder(w).Encode(e)
}

// Common API errors used across the application.
var (
	ErrInternalServer = APIError{Code: http.StatusInternalServerError, Message: "Internal server error"}
	ErrNotFound       = APIError{Code: http.StatusNotFound, Message: "Not found"}
	ErrUnauthorized   = APIError{Code: http.StatusUnauthorized, Message: "Unauthorized"}
	ErrForbidden      = APIError{Code: http.StatusForbidden, Message: "Forbidden"}
	ErrTooManyReqs    = APIError{Code: http.StatusTooManyRequests, Message: "Rate limit exceeded. Try again later."}
	ErrBadRequest     = APIError{Code: http.StatusBadRequest, Message: "Bad request"}
)
