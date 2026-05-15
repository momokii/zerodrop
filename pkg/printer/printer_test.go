package printer

import (
	"testing"
)

func TestMockPrinter_Print(t *testing.T) {
	printer := NewMockPrinter()

	ciphertext := []byte("ZD1:test123")

	err := printer.Print(ciphertext)
	if err != nil {
		t.Fatalf("Print failed: %v", err)
	}
}

func TestMockPrinter_IsAvailable(t *testing.T) {
	printer := NewMockPrinter()

	if !printer.IsAvailable() {
		t.Error("New mock printer should be available")
	}

	printer.SetAvailable(false)
	if printer.IsAvailable() {
		t.Error("Mock printer should not be available after SetAvailable(false)")
	}
}
