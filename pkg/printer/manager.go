package printer

import (
	"fmt"
	"log"
	"sync"
)

// PrinterManager handles printer detection and active printer selection.
type PrinterManager struct {
	mu       sync.RWMutex
	active   Printer
	activeID string
	detected []PrinterInfo
}

// NewPrinterManager creates a manager with the given initial printer.
func NewPrinterManager(initial Printer, initialInfo PrinterInfo) *PrinterManager {
	return &PrinterManager{
		active:   initial,
		activeID: initialInfo.ID,
		detected: []PrinterInfo{initialInfo},
	}
}

// GetActive returns the currently active printer.
func (pm *PrinterManager) GetActive() Printer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.active
}

// GetActiveInfo returns info about the active printer.
func (pm *PrinterManager) GetActiveInfo() PrinterInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, p := range pm.detected {
		if p.ID == pm.activeID {
			return p
		}
	}
	return PrinterInfo{ID: pm.activeID, Name: "Unknown", Type: "unknown"}
}

// Detect scans for all connected printers and updates the cached list.
// Does NOT change the active printer.
func (pm *PrinterManager) Detect() []PrinterInfo {
	printers := DetectAvailablePrinters()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	result := make([]PrinterInfo, 0, len(printers)+1)

	// Always include mock as an option
	result = append(result, PrinterInfo{
		ID: "mock", Name: "Mock Printer (stdout)", Type: "mock", Device: "",
	})

	for _, p := range printers {
		result = append(result, PrinterInfo{
			ID:     p["path"],
			Name:   p["model"],
			Type:   "usb",
			Device: p["path"],
		})
	}

	pm.detected = result
	return result
}

// ListDetected returns the cached list of detected printers.
func (pm *PrinterManager) ListDetected() []PrinterInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make([]PrinterInfo, len(pm.detected))
	copy(result, pm.detected)
	return result
}

// SetActive switches the active printer. Returns error if ID not found.
func (pm *PrinterManager) SetActive(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var found *PrinterInfo
	for _, p := range pm.detected {
		if p.ID == id {
			found = &p
			break
		}
	}
	if found == nil {
		return fmt.Errorf("printer not found: %s (run detect first)", id)
	}

	var newPrinter Printer
	if found.Type == "mock" {
		newPrinter = NewMockPrinter()
	} else {
		usb, err := NewUSBPrinter(found.Device)
		if err != nil {
			return fmt.Errorf("failed to initialize USB printer %s: %w", found.Device, err)
		}
		newPrinter = usb
	}

	pm.active = newPrinter
	pm.activeID = id
	log.Printf("Active printer switched to: %s (%s)", found.Name, found.ID)
	return nil
}
