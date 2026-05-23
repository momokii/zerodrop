package printer

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zerodrop/terminal/pkg/qr"
)

// Known thermal printer VID:PID pairs (58mm thermal printers)
var knownPrinters = []struct {
	vendorID  string
	productID string
	name      string
}{
	{"04b8", "0202", "Epson TM-T88 (compatible)"},
	{"04b8", "0203", "Epson TM-T88II"},
	{"1504", "0006", "POS-5890 / Generic ESC/POS"},
	{"0416", "5011", "Rongta RP58"},
	{"0456", "0808", "XPrinter XP-58III"},
	{"0493", "b002", "Citizen CT-S310"},
	{"0519", "0001", "Star Micronics TSP650"},
	{"0dd4", "01a5", "BCST Printers"},
	{"20d1", "0001", "Gainscha"},
	{"0fe6", "811e", "Zjiang"},
	{"0418", "0156", "Custom VG205"},
}

// USBPrinter writes to a USB thermal printer device
type USBPrinter struct {
	devicePath string
	file       *os.File
	available  bool
}

// NewUSBPrinter creates a new USB printer with auto-detection
// If devicePath is empty, auto-detects the first available thermal printer
func NewUSBPrinter(devicePath string) (*USBPrinter, error) {
	var actualPath string

	if devicePath == "" {
		// Auto-detect printer
		detected, err := detectPrinter()
		if err != nil {
			return nil, fmt.Errorf("auto-detection failed: %w", err)
		}
		actualPath = detected
		log.Printf("[USBPrinter] Auto-detected printer at %s", actualPath)
	} else {
		actualPath = devicePath
		log.Printf("[USBPrinter] Using configured device: %s", actualPath)
	}

	// Validate device exists and is writable
	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("printer device not found: %s", actualPath)
	}

	// Try to open device to verify accessibility
	file, err := os.OpenFile(actualPath, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot open printer device: %w", err)
	}
	file.Close()

	return &USBPrinter{
		devicePath: actualPath,
		available:  true,
	}, nil
}

// detectPrinter scans for known thermal printers in /dev/usb/
func detectPrinter() (string, error) {
	// Common USB printer device paths
	candidatePaths := []string{
		"/dev/usb/lp0",
		"/dev/usb/lp1",
		"/dev/usb/lp2",
		"/dev/usblp0",
		"/dev/usblp1",
		"/dev/usblp2",
	}

	for _, path := range candidatePaths {
		// Check if device exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Try to identify the device by reading sysfs
		deviceName, err := identifyDevice(path)
		if err != nil {
			log.Printf("[USBPrinter] Could not identify %s: %v", path, err)
			continue
		}

		log.Printf("[USBPrinter] Found thermal printer: %s at %s", deviceName, path)
		return path, nil
	}

	return "", fmt.Errorf("no supported thermal printer found (checked: %v)", candidatePaths)
}

// identifyDevice attempts to identify the printer model from sysfs
func identifyDevice(devicePath string) (string, error) {
	// Extract device number from path (e.g., /dev/usb/lp0 -> lp0)
	baseName := devicePath[strings.LastIndex(devicePath, "/")+1:]

	// Try to read vendor/product IDs from different locations
	locations := []string{
		fmt.Sprintf("/sys/class/usbmisc/%s/../../idVendor", baseName),
		fmt.Sprintf("/sys/class/usbmisc/%s/../../idProduct", baseName),
		fmt.Sprintf("/sys/class/printer/%s/../../idVendor", baseName),
		fmt.Sprintf("/sys/class/printer/%s/../../idProduct", baseName),
	}

	var vid, pid string
	for _, loc := range locations {
		if strings.Contains(loc, "idVendor") {
			data, err := os.ReadFile(loc)
			if err == nil {
				vid = strings.TrimSpace(string(data))
			}
		} else if strings.Contains(loc, "idProduct") {
			data, err := os.ReadFile(loc)
			if err == nil {
				pid = strings.TrimSpace(string(data))
			}
		}
	}

	// Check if this is a known printer
	if vid != "" && pid != "" {
		vidPid := vid + ":" + pid
		for _, printer := range knownPrinters {
			if printer.vendorID == vid && printer.productID == pid {
				return printer.name, nil
			}
		}
		return fmt.Sprintf("Unknown USB printer (VID:PID %s)", vidPid), nil
	}

	// Fallback: check if we can at least open the device
	file, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		return "", fmt.Errorf("cannot open device for identification: %w", err)
	}
	file.Close()

	return "Generic ESC/POS printer (unidentified)", nil
}

// Print writes QR code ESC/POS commands to the USB printer
func (p *USBPrinter) Print(ciphertext []byte) error {
	if !p.available {
		return fmt.Errorf("printer is not available")
	}

	// Open device (we open/close for each print to handle hotplug)
	file, err := os.OpenFile(p.devicePath, os.O_WRONLY, 0)
	if err != nil {
		p.available = false
		return fmt.Errorf("cannot open printer: %w", err)
	}
	defer file.Close()

	// Generate QR ESC/POS commands
	escpos, err := qr.GenerateQRESCPOS(ciphertext)
	if err != nil {
		return fmt.Errorf("failed to generate QR: %w", err)
	}

	// Write ESC/POS commands to printer
	if _, err := file.Write(escpos); err != nil {
		p.available = false
		return fmt.Errorf("failed to write to printer: %w", err)
	}

	log.Printf("[USBPrinter] Print job sent to %s (%d bytes, %d ciphertext bytes)",
		p.devicePath, len(escpos), len(ciphertext))
	return nil
}

// IsAvailable returns whether the USB printer is available
func (p *USBPrinter) IsAvailable() bool {
	if !p.available {
		return false
	}

	// Verify device is still accessible
	file, err := os.OpenFile(p.devicePath, os.O_WRONLY, 0)
	if err != nil {
		p.available = false
		return false
	}
	file.Close()

	return true
}

// GetDevicePath returns the device path
func (p *USBPrinter) GetDevicePath() string {
	return p.devicePath
}

// SetAvailable sets the availability status
func (p *USBPrinter) SetAvailable(available bool) {
	p.available = available
}

// HealthCheck performs a health check on the USB printer
func (p *USBPrinter) HealthCheck() map[string]interface{} {
	status := make(map[string]interface{})

	status["type"] = "usb"
	status["device_path"] = p.devicePath
	status["available"] = p.IsAvailable()

	if p.available {
		// Check if device file exists and is writable
		info, err := os.Stat(p.devicePath)
		if err != nil {
			status["error"] = err.Error()
			status["available"] = false
		} else {
			status["mode"] = info.Mode().String()
		}

		// Try to identify the device
		if name, err := identifyDevice(p.devicePath); err == nil {
			status["model"] = name
		}
	}

	return status
}

// DetectAvailablePrinters returns a list of all detected thermal printers
func DetectAvailablePrinters() []map[string]string {
	printers := make([]map[string]string, 0)

	candidatePaths := []string{
		"/dev/usb/lp0", "/dev/usb/lp1", "/dev/usb/lp2",
		"/dev/usblp0", "/dev/usblp1", "/dev/usblp2",
	}

	for _, path := range candidatePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		printerInfo := map[string]string{
			"path": path,
		}

		if name, err := identifyDevice(path); err == nil {
			printerInfo["model"] = name
		} else {
			printerInfo["model"] = "Unknown"
		}

		printers = append(printers, printerInfo)
	}

	return printers
}

// Reconnect attempts to reconnect to the printer
func (p *USBPrinter) Reconnect() error {
	file, err := os.OpenFile(p.devicePath, os.O_WRONLY, 0)
	if err != nil {
		p.available = false
		return fmt.Errorf("reconnect failed: %w", err)
	}
	file.Close()

	p.available = true
	log.Printf("[USBPrinter] Reconnected to %s", p.devicePath)
	return nil
}

// Close closes the printer connection
func (p *USBPrinter) Close() error {
	p.available = false
	return nil
}
