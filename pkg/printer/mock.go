package printer

import (
	"fmt"
	"log"

	"github.com/zerodrop/terminal/pkg/qr"
)

// MockPrinter writes QR code ESC/POS commands to stdout for testing and CI
type MockPrinter struct {
	available bool
}

// NewMockPrinter creates a new mock printer that writes to stdout
func NewMockPrinter() *MockPrinter {
	return &MockPrinter{
		available: true,
	}
}

// Print generates a QR code from the ciphertext and outputs ESC/POS commands to stdout
func (m *MockPrinter) Print(ciphertext []byte) error {
	if !m.available {
		return fmt.Errorf("printer is not available")
	}

	// Generate QR ESC/POS commands
	escpos, err := qr.GenerateQRESCPOS(ciphertext)
	if err != nil {
		return fmt.Errorf("failed to generate QR: %w", err)
	}

	log.Printf("[MockPrinter] Print job received (%d bytes ciphertext)", len(ciphertext))
	log.Printf("[MockPrinter] Generated QR ESC/POS output: %d bytes", len(escpos))
	if len(escpos) > 0 {
		// Log first 100 bytes as hex for debugging
		preview := escpos
		if len(preview) > 100 {
			preview = preview[:100]
		}
		log.Printf("[MockPrinter] ESC/POS header hex: %x", preview)
	}

	return nil
}

// IsAvailable returns whether the mock printer is available
func (m *MockPrinter) IsAvailable() bool {
	return m.available
}

// SetAvailable sets the availability status of the mock printer
func (m *MockPrinter) SetAvailable(available bool) {
	m.available = available
}

// HealthCheck returns health status for the mock printer
func (m *MockPrinter) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"type":      "mock",
		"available": m.available,
		"status":    "healthy",
	}
}
