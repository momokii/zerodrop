package printer

import (
	"fmt"
	"log"
)

// MockPrinter writes ESC/POS commands to stdout for testing and CI
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

	log.Printf("[MockPrinter] Print job received")
	log.Printf("[MockPrinter] Ciphertext: %s", string(ciphertext))
	log.Printf("[MockPrinter] QR Payload (for display): %s", string(ciphertext))
	log.Printf("[MockPrinter] Simulating QR code generation and ESC/POS rasterization...")
	log.Printf("[MockPrinter] Paper cut command sent")

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
