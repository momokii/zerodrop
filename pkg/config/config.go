package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	// PrinterType specifies the type of printer to use (mock, usb, tcp)
	PrinterType string

	// PrinterDevice is the path to the printer device (e.g., /dev/usb/lp0)
	PrinterDevice string

	// RateLimitRequestsPerHour is the maximum number of requests per IP per hour
	RateLimitRequestsPerHour int

	// RateLimitBurst is the burst size for rate limiting
	RateLimitBurst int

	// LogEnabled enables structured JSON logging when true
	LogEnabled bool

	// PublicKeyPath is where to save/load the public key
	PublicKeyPath string
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		PrinterType:              "mock",
		PrinterDevice:            "/dev/usb/lp0",
		RateLimitRequestsPerHour: 5,
		RateLimitBurst:           1,
		LogEnabled:               false,
		PublicKeyPath:            "./data/public_key.pem",
	}
}

// LoadFromEnv loads configuration from environment variables
// Required variables must be set or an error is returned
func LoadFromEnv() (*Config, error) {
	config := DefaultConfig()

	// Required: PRINTER_TYPE
	printerType, ok := os.LookupEnv("PRINTER_TYPE")
	if !ok {
		return nil, fmt.Errorf("PRINTER_TYPE is required (must be one of: mock, usb, tcp)")
	}
	config.PrinterType = printerType

	// Validate PRINTER_TYPE
	if config.PrinterType != "mock" && config.PrinterType != "usb" && config.PrinterType != "tcp" {
		return nil, fmt.Errorf("PRINTER_TYPE must be one of: mock, usb, tcp (got: %s)", config.PrinterType)
	}

	// Optional: PRINTER_DEVICE (not required for mock, optional for usb with auto-detection)
	printerDevice, printerDeviceSet := os.LookupEnv("PRINTER_DEVICE")

	// For USB printer, if PRINTER_DEVICE is not explicitly set, use empty string for auto-detection
	// For mock printer, use default (won't be used anyway)
	if config.PrinterType == "usb" {
		if printerDeviceSet {
			config.PrinterDevice = printerDevice
		} else {
			// Empty string triggers auto-detection in printer initialization
			config.PrinterDevice = ""
		}
	} else if printerDevice != "" {
		config.PrinterDevice = printerDevice
	}

	// For USB printer with specified device, validate device exists and is writable
	// If device is empty, auto-detection will be attempted in printer initialization
	if config.PrinterType == "usb" && config.PrinterDevice != "" {
		if err := validatePrinterDevice(config.PrinterDevice); err != nil {
			return nil, fmt.Errorf("PRINTER_DEVICE validation failed: %w", err)
		}
	}

	// Optional: RATE_LIMIT_REQUESTS_PER_HOUR
	if val := os.Getenv("RATE_LIMIT_REQUESTS_PER_HOUR"); val != "" {
		hourly, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("RATE_LIMIT_REQUESTS_PER_HOUR must be an integer: %w", err)
		}
		if hourly < 1 {
			return nil, fmt.Errorf("RATE_LIMIT_REQUESTS_PER_HOUR must be at least 1 (got: %d)", hourly)
		}
		config.RateLimitRequestsPerHour = hourly
	}

	// Optional: RATE_LIMIT_BURST
	if val := os.Getenv("RATE_LIMIT_BURST"); val != "" {
		burst, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("RATE_LIMIT_BURST must be an integer: %w", err)
		}
		if burst < 1 {
			return nil, fmt.Errorf("RATE_LIMIT_BURST must be at least 1 (got: %d)", burst)
		}
		config.RateLimitBurst = burst
	}

	// Optional: LOG_ENABLED
	if val := os.Getenv("LOG_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("LOG_ENABLED must be a boolean (true/false): %w", err)
		}
		config.LogEnabled = enabled
	}

	// Optional: PUBLIC_KEY_PATH
	if val := os.Getenv("PUBLIC_KEY_PATH"); val != "" {
		config.PublicKeyPath = val
	}

	return config, nil
}

// validatePrinterDevice checks that the printer device exists and is writable
func validatePrinterDevice(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("printer device does not exist: %s", path)
		}
		return fmt.Errorf("cannot access printer device: %w", err)
	}

	// Check if it's a character device
	if info.Mode()&os.ModeCharDevice == 0 {
		return fmt.Errorf("printer device is not a character device: %s", path)
	}

	// Check if writable (try to open file)
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("printer device is not writable: %w", err)
	}
	file.Close()

	return nil
}
