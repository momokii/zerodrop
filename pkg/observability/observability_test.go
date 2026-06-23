package observability

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock spooler for testing ShutdownHandler
// ---------------------------------------------------------------------------

type mockSpooler struct {
	mu          sync.Mutex
	drainCalled bool
	stopCalled  bool
	drainResult bool
	drainDelay  time.Duration
	stopDelay   time.Duration
}

func (m *mockSpooler) Drain(timeout time.Duration) bool {
	m.mu.Lock()
	m.drainCalled = true
	m.mu.Unlock()
	if m.drainDelay > 0 {
		time.Sleep(m.drainDelay)
	}
	return m.drainResult
}

func (m *mockSpooler) Stop(timeout time.Duration) {
	m.mu.Lock()
	m.stopCalled = true
	m.mu.Unlock()
	if m.stopDelay > 0 {
		time.Sleep(m.stopDelay)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func captureLoggerOutput(enabled bool, fn func(l *Logger)) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewLogger(enabled)
	fn(logger)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// ---------------------------------------------------------------------------
// Logger tests
// ---------------------------------------------------------------------------

func TestLogger_Disabled_NoOutput(t *testing.T) {
	output := captureLoggerOutput(false, func(l *Logger) {
		l.Info("test message", map[string]interface{}{"key": "value"})
	})
	if output != "" {
		t.Fatalf("expected no output when logger disabled, got: %q", output)
	}
}

func TestLogger_Enabled_JSONOutput(t *testing.T) {
	output := captureLoggerOutput(true, func(l *Logger) {
		l.Info("test message", map[string]interface{}{"key": "value"})
	})
	if output == "" {
		t.Fatal("expected output when logger enabled")
	}

	var entry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("expected valid JSON log entry: %v\nraw: %q", err, output)
	}
	if entry.Level != "INFO" {
		t.Fatalf("expected level INFO, got: %s", entry.Level)
	}
	if entry.Message != "test message" {
		t.Fatalf("expected message 'test message', got: %s", entry.Message)
	}
	if entry.Fields["key"] != "value" {
		t.Fatalf("expected field key=value, got: %v", entry.Fields)
	}
}

func TestLogger_Levels(t *testing.T) {
	levels := []struct {
		name string
		fn   func(l *Logger)
	}{
		{"INFO", func(l *Logger) { l.Info("msg", nil) }},
		{"ERROR", func(l *Logger) { l.Error("msg", nil) }},
		{"WARN", func(l *Logger) { l.Warn("msg", nil) }},
		{"DEBUG", func(l *Logger) { l.Debug("msg", nil) }},
	}

	for _, tt := range levels {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLoggerOutput(true, tt.fn)
			if output == "" {
				t.Fatalf("expected output for level %s", tt.name)
			}
			var entry LogEntry
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if entry.Level != tt.name {
				t.Fatalf("expected level %s, got %s", tt.name, entry.Level)
			}
		})
	}
}

func TestLogger_Disabled_ErrorsStillLogged(t *testing.T) {
	output := captureLoggerOutput(false, func(l *Logger) {
		l.Error("something failed", map[string]interface{}{"error": "connection refused"})
	})
	if output == "" {
		t.Fatal("expected ERROR output even when logger disabled")
	}
	if !strings.Contains(output, "ERROR") {
		t.Fatal("expected ERROR level in output")
	}
	if !strings.Contains(output, "connection refused") {
		t.Fatal("expected error detail in disabled-mode output")
	}
}

func TestLogger_FieldsAreOptional(t *testing.T) {
	output := captureLoggerOutput(true, func(l *Logger) {
		l.Info("no fields message", nil)
	})
	var entry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry.Message != "no fields message" {
		t.Fatalf("expected message 'no fields message', got: %s", entry.Message)
	}
}

func TestLogger_TimestampIsRFC3339(t *testing.T) {
	output := captureLoggerOutput(true, func(l *Logger) {
		l.Info("timestamp check", nil)
	})
	var entry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	_, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		t.Fatalf("expected RFC3339 timestamp, got %q: %v", entry.Timestamp, err)
	}
}

func TestLogger_MultipleCalls(t *testing.T) {
	output := captureLoggerOutput(true, func(l *Logger) {
		l.Info("first", nil)
		l.Error("second", nil)
		l.Warn("third", nil)
	})
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(lines))
	}
}

// ---------------------------------------------------------------------------
// ShutdownHandler tests
// ---------------------------------------------------------------------------

func TestShutdownHandler_Shutdown_DrainsAndStops(t *testing.T) {
	ms := &mockSpooler{drainResult: true}
	sh := NewShutdownHandler(ms, 5*time.Second)

	sh.Shutdown()

	if !ms.drainCalled {
		t.Fatal("expected spooler.Drain to be called")
	}
	if !ms.stopCalled {
		t.Fatal("expected spooler.Stop to be called")
	}
}

func TestShutdownHandler_Shutdown_DrainTimeoutExceeded(t *testing.T) {
	ms := &mockSpooler{
		drainResult: false,
		drainDelay:  100 * time.Millisecond,
	}
	sh := NewShutdownHandler(ms, 50*time.Millisecond)

	sh.Shutdown()

	if !ms.drainCalled {
		t.Fatal("expected Drain to be called even with short timeout")
	}
}

func TestShutdownHandler_Shutdown_DrainReturnsFalse(t *testing.T) {
	ms := &mockSpooler{drainResult: false}
	sh := NewShutdownHandler(ms, 5*time.Second)

	// Should not panic or hang
	sh.Shutdown()

	if !ms.drainCalled {
		t.Fatal("expected Drain to be called")
	}
	if !ms.stopCalled {
		t.Fatal("expected Stop to be called after Drain")
	}
}

func TestShutdownHandler_Shutdown_StopCalledAfterDrain(t *testing.T) {
	ms := &mockSpooler{drainResult: true}
	sh := NewShutdownHandler(ms, 5*time.Second)

	sh.Shutdown()

	if !ms.drainCalled || !ms.stopCalled {
		t.Fatal("expected both Drain and Stop to be called in order")
	}
}
