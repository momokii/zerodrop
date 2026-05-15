package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.PrinterType != "mock" {
		t.Errorf("Expected PrinterType to be 'mock', got '%s'", config.PrinterType)
	}

	if config.PrinterDevice != "/dev/usb/lp0" {
		t.Errorf("Expected PrinterDevice to be '/dev/usb/lp0', got '%s'", config.PrinterDevice)
	}

	if config.RateLimitRequestsPerHour != 5 {
		t.Errorf("Expected RateLimitRequestsPerHour to be 5, got %d", config.RateLimitRequestsPerHour)
	}

	if config.RateLimitBurst != 1 {
		t.Errorf("Expected RateLimitBurst to be 1, got %d", config.RateLimitBurst)
	}

	if config.LogEnabled != false {
		t.Errorf("Expected LogEnabled to be false, got %v", config.LogEnabled)
	}
}

func TestLoadFromEnv_MissingRequired(t *testing.T) {
	// Clear all env vars
	os.Unsetenv("PRINTER_TYPE")
	os.Unsetenv("PRINTER_DEVICE")

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("Expected error when PRINTER_TYPE is missing")
	}
}

func TestLoadFromEnv_InvalidPrinterType(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "invalid")
	defer os.Unsetenv("PRINTER_TYPE")

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("Expected error for invalid PRINTER_TYPE")
	}
}

func TestLoadFromEnv_MockPrinter(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "mock")
	defer os.Unsetenv("PRINTER_TYPE")

	config, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if config.PrinterType != "mock" {
		t.Errorf("Expected PrinterType to be 'mock', got '%s'", config.PrinterType)
	}
}

func TestLoadFromEnv_USBPrinter_MissingDevice(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "usb")
	os.Unsetenv("PRINTER_DEVICE")
	defer os.Unsetenv("PRINTER_TYPE")

	config, err := LoadFromEnv()
	if err != nil {
		t.Errorf("Expected no error for USB printer with auto-detection, got: %v", err)
	}

	if config.PrinterDevice != "" {
		t.Errorf("Expected PrinterDevice to be empty for auto-detection, got '%s'", config.PrinterDevice)
	}
}

func TestLoadFromEnv_USBPrinter_WithInvalidDevice(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "usb")
	os.Setenv("PRINTER_DEVICE", "/dev/does/not/exist")
	defer func() {
		os.Unsetenv("PRINTER_TYPE")
		os.Unsetenv("PRINTER_DEVICE")
	}()

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("Expected error when PRINTER_DEVICE points to non-existent device")
	}
}

func TestLoadFromEnv_OptionalFields(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "mock")
	os.Setenv("RATE_LIMIT_REQUESTS_PER_HOUR", "10")
	os.Setenv("RATE_LIMIT_BURST", "3")
	os.Setenv("LOG_ENABLED", "true")
	defer func() {
		os.Unsetenv("PRINTER_TYPE")
		os.Unsetenv("RATE_LIMIT_REQUESTS_PER_HOUR")
		os.Unsetenv("RATE_LIMIT_BURST")
		os.Unsetenv("LOG_ENABLED")
	}()

	config, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if config.RateLimitRequestsPerHour != 10 {
		t.Errorf("Expected RateLimitRequestsPerHour to be 10, got %d", config.RateLimitRequestsPerHour)
	}

	if config.RateLimitBurst != 3 {
		t.Errorf("Expected RateLimitBurst to be 3, got %d", config.RateLimitBurst)
	}

	if config.LogEnabled != true {
		t.Errorf("Expected LogEnabled to be true, got %v", config.LogEnabled)
	}
}

func TestLoadFromEnv_InvalidRateLimit(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "mock")
	os.Setenv("RATE_LIMIT_REQUESTS_PER_HOUR", "invalid")
	defer func() {
		os.Unsetenv("PRINTER_TYPE")
		os.Unsetenv("RATE_LIMIT_REQUESTS_PER_HOUR")
	}()

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("Expected error for invalid RATE_LIMIT_REQUESTS_PER_HOUR")
	}
}

func TestLoadFromEnv_InvalidLogEnabled(t *testing.T) {
	os.Setenv("PRINTER_TYPE", "mock")
	os.Setenv("LOG_ENABLED", "invalid")
	defer func() {
		os.Unsetenv("PRINTER_TYPE")
		os.Unsetenv("LOG_ENABLED")
	}()

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("Expected error for invalid LOG_ENABLED")
	}
}
