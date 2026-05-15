package printer

import (
	"testing"
)

func TestDetectAvailablePrinters(t *testing.T) {
	printers := DetectAvailablePrinters()

	// Should return a slice (may be empty)
	if printers == nil {
		t.Error("DetectAvailablePrinters should never return nil")
	}

	t.Logf("Detected %d printer(s)", len(printers))
	for i, p := range printers {
		t.Logf("  Printer %d: path=%s, model=%s", i+1, p["path"], p["model"])
	}
}

func TestNewUSBPrinter_WithInvalidPath(t *testing.T) {
	// Should fail with non-existent device
	_, err := NewUSBPrinter("/dev/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent device")
	}
}

func TestNewUSBPrinter_WithEmptyPath_AutoDetect(t *testing.T) {
	// Auto-detect with empty path
	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Logf("Auto-detection failed (expected if no USB printer connected): %v", err)
		return
	}

	defer printer.Close()

	if printer.devicePath == "" {
		t.Error("Expected device path to be set")
	}

	t.Logf("Auto-detected printer at: %s", printer.devicePath)
}

func TestUSBPrinter_HealthCheck(t *testing.T) {
	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Skip("No USB printer available for testing")
		return
	}
	defer printer.Close()

	health := printer.HealthCheck()

	if health["type"] != "usb" {
		t.Errorf("Expected type 'usb', got %v", health["type"])
	}

	if health["device_path"] == nil {
		t.Error("Expected device_path in health check")
	}

	t.Logf("Health check result: %+v", health)
}

func TestUSBPrinter_IsAvailable(t *testing.T) {
	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Skip("No USB printer available for testing")
		return
	}
	defer printer.Close()

	if !printer.IsAvailable() {
		t.Error("Expected printer to be available after initialization")
	}
}

func TestUSBPrinter_SetAvailable(t *testing.T) {
	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Skip("No USB printer available for testing")
		return
	}
	defer printer.Close()

	printer.SetAvailable(false)

	if printer.IsAvailable() {
		t.Error("Expected printer to be unavailable after SetAvailable(false)")
	}

	// Test reconnect
	err = printer.Reconnect()
	if err != nil {
		t.Errorf("Reconnect failed: %v", err)
	}

	if !printer.IsAvailable() {
		t.Error("Expected printer to be available after reconnect")
	}
}

func TestIdentifyDevice_WithMockDevice(t *testing.T) {
	// Test with a non-existent device (should return error)
	_, err := identifyDevice("/dev/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent device")
	}
}

func TestKnownPrintersList(t *testing.T) {
	// Verify the known printers list is not empty
	if len(knownPrinters) == 0 {
		t.Error("Expected knownPrinters list to have entries")
	}

	t.Logf("Known printer models: %d", len(knownPrinters))
	for i, p := range knownPrinters {
		t.Logf("  %d. %s:%s - %s", i+1, p.vendorID, p.productID, p.name)
	}
}

// TestPrint_WithRealDevice is an integration test that requires actual hardware
// It should be skipped in CI and only run manually with a connected printer
func TestPrint_WithRealDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Skip("No USB printer available for integration test")
		return
	}
	defer printer.Close()

	testPayload := []byte("ZD1:SGVsbG8gWmVyb0Ryb3Ah")
	err = printer.Print(testPayload)
	if err != nil {
		t.Errorf("Print failed: %v", err)
	}

	t.Log("Print test completed successfully")
}

// TestPrint_WhenPrinterUnavailable tests error handling
func TestPrint_WhenPrinterUnavailable(t *testing.T) {
	printer := &USBPrinter{
		devicePath: "/dev/usb/lp0",
		available:  false,
	}

	err := printer.Print([]byte("test"))
	if err == nil {
		t.Error("Expected error when printer is unavailable")
	}
}

func TestGetDevicePath(t *testing.T) {
	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Skip("No USB printer available for testing")
		return
	}
	defer printer.Close()

	path := printer.GetDevicePath()
	if path == "" {
		t.Error("Expected device path to be non-empty")
	}

	t.Logf("Device path: %s", path)
}

// TestUSBPrinter_Close tests the Close method
func TestUSBPrinter_Close(t *testing.T) {
	printer, err := NewUSBPrinter("")
	if err != nil {
		t.Skip("No USB printer available for testing")
		return
	}

	printer.Close()

	if printer.IsAvailable() {
		t.Error("Expected printer to be unavailable after Close")
	}
}
