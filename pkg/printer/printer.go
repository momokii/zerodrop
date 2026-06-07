package printer

// Printer defines the interface for all printer implementations
type Printer interface {
	// Print accepts a ciphertext payload and outputs it as a QR code
	// Returns an error if printing fails
	Print(ciphertext []byte) error

	// IsAvailable returns true if the printer is connected and ready
	IsAvailable() bool
}

// HealthChecker defines optional health check functionality
type HealthChecker interface {
	// HealthCheck returns health status information
	HealthCheck() map[string]interface{}
}

// Reconnector defines optional reconnection functionality
type Reconnector interface {
	// Reconnect attempts to reconnect to the printer
	Reconnect() error
}

// PrinterInfo describes a detected printer.
type PrinterInfo struct {
	ID        string `json:"id"`        // Unique identifier (device path or "mock")
	Name      string `json:"name"`      // Human-readable model name
	Type      string `json:"type"`      // "usb", "rawusb", or "mock"
	Device    string `json:"device"`    // Device path (e.g., /dev/usb/lp0 or /dev/bus/usb/...)
	VendorID  string `json:"vendor_id,omitempty"`  // USB vendor ID (for raw USB)
	ProductID string `json:"product_id,omitempty"` // USB product ID (for raw USB)
}
