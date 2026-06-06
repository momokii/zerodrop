package printer

import (
	"testing"
)

func TestNewPrinterManager(t *testing.T) {
	mock := NewMockPrinter()
	pm := NewPrinterManager(mock, PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})

	if pm == nil {
		t.Fatal("expected non-nil PrinterManager")
	}

	active := pm.GetActive()
	if active == nil {
		t.Fatal("expected non-nil active printer")
	}

	info := pm.GetActiveInfo()
	if info.ID != "mock" {
		t.Errorf("expected active ID 'mock', got '%s'", info.ID)
	}
}

func TestPrinterManager_ListDetected(t *testing.T) {
	mock := NewMockPrinter()
	pm := NewPrinterManager(mock, PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})

	printers := pm.ListDetected()
	if len(printers) < 1 {
		t.Fatal("expected at least one printer (initial)")
	}
	if printers[0].ID != "mock" {
		t.Errorf("expected first printer to be 'mock', got '%s'", printers[0].ID)
	}
}

func TestPrinterManager_Detect(t *testing.T) {
	mock := NewMockPrinter()
	pm := NewPrinterManager(mock, PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})

	// Detect on a system without USB printers should still return mock
	printers := pm.Detect()
	if len(printers) < 1 {
		t.Fatal("expected at least one printer (mock always included)")
	}

	found := false
	for _, p := range printers {
		if p.ID == "mock" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected mock printer in detected list")
	}
}

func TestPrinterManager_SetActive_Mock(t *testing.T) {
	mock := NewMockPrinter()
	pm := NewPrinterManager(mock, PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})
	pm.Detect()

	err := pm.SetActive("mock")
	if err != nil {
		t.Fatalf("expected no error setting mock, got: %v", err)
	}

	info := pm.GetActiveInfo()
	if info.ID != "mock" {
		t.Errorf("expected active ID 'mock', got '%s'", info.ID)
	}
}

func TestPrinterManager_SetActive_NotFound(t *testing.T) {
	mock := NewMockPrinter()
	pm := NewPrinterManager(mock, PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})

	err := pm.SetActive("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent printer ID")
	}
}

func TestPrinterManager_GetActiveInfo_Unknown(t *testing.T) {
	mock := NewMockPrinter()
	pm := NewPrinterManager(mock, PrinterInfo{
		ID: "mock", Name: "Mock Printer", Type: "mock",
	})

	// Manually set activeID to something not in detected list
	pm.mu.Lock()
	pm.activeID = "ghost"
	pm.mu.Unlock()

	info := pm.GetActiveInfo()
	if info.Name != "Unknown" {
		t.Errorf("expected 'Unknown' for missing active printer, got '%s'", info.Name)
	}
}
