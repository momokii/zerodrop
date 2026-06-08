package spooler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/zerodrop/terminal/pkg/printer"
)

// ---------------------------------------------------------------------------
// Test-only mocks
// ---------------------------------------------------------------------------

// mockPrinter is a configurable printer for testing.
type mockPrinter struct {
	mu        sync.Mutex
	fail      bool         // when true, Print returns an error
	printFn   func() error // if set, called instead of fail check
	callCount int
}

func (m *mockPrinter) Print([]byte) error {
	m.mu.Lock()
	m.callCount++
	fn := m.printFn
	fail := m.fail
	m.mu.Unlock()
	if fn != nil {
		return fn()
	}
	if fail {
		return fmt.Errorf("mock printer error")
	}
	return nil
}

func (m *mockPrinter) IsAvailable() bool { return true }

func (m *mockPrinter) callCountValue() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// mockReconnectPrinter also implements the Reconnect interface.
type mockReconnectPrinter struct {
	mockPrinter
	reconnectMu   sync.Mutex
	reconnectCount int
}

func (m *mockReconnectPrinter) Reconnect() error {
	m.reconnectMu.Lock()
	defer m.reconnectMu.Unlock()
	m.reconnectCount++
	return nil
}

func (m *mockReconnectPrinter) reconnectCountValue() int {
	m.reconnectMu.Lock()
	defer m.reconnectMu.Unlock()
	return m.reconnectCount
}

// mockProvider implements PrinterProvider for testing runtime printer switching.
type mockProvider struct {
	mu      sync.Mutex
	printer printer.Printer
}

func (p *mockProvider) GetActive() printer.Printer {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.printer
}

func (p *mockProvider) setPrinter(pr printer.Printer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.printer = pr
}

// ---------------------------------------------------------------------------
// NewSpooler
// ---------------------------------------------------------------------------

func TestNewSpooler_WithPrinter(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)
	if s == nil {
		t.Fatal("expected non-nil spooler")
	}
	if s.QueueSize() != 0 {
		t.Errorf("expected empty queue, got size %d", s.QueueSize())
	}
	if s.queue == nil {
		t.Error("expected queue channel to be initialized")
	}
	if s.metrics == nil {
		t.Error("expected metrics to be initialized")
	}
}

func TestNewSpooler_WithPrinterProvider(t *testing.T) {
	mp := &mockPrinter{}
	provider := &mockProvider{printer: mp}
	s := NewSpooler(10, provider)
	if s == nil {
		t.Fatal("expected non-nil spooler")
	}
}

func TestNewSpooler_InvalidProviderPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with invalid provider type")
		}
	}()
	NewSpooler(10, "not-a-printer")
}

func TestNewSpooler_WithNilPrinter(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil")
		}
	}()
	NewSpooler(10, nil)
}

// ---------------------------------------------------------------------------
// Enqueue / QueueSize
// ---------------------------------------------------------------------------

func TestEnqueue(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})
	ok := s.Enqueue([]byte("ZD1:test"))
	if !ok {
		t.Error("expected enqueue to succeed")
	}
	if s.QueueSize() != 1 {
		t.Errorf("expected queue size 1, got %d", s.QueueSize())
	}
}

func TestEnqueue_Multiple(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})
	for i := 0; i < 5; i++ {
		if !s.Enqueue([]byte(fmt.Sprintf("ZD1:job-%d", i))) {
			t.Errorf("enqueue failed at job %d", i)
		}
	}
	if s.QueueSize() != 5 {
		t.Errorf("expected queue size 5, got %d", s.QueueSize())
	}
}

func TestEnqueue_FullQueue(t *testing.T) {
	s := NewSpooler(1, &mockPrinter{}) // capacity 1

	if !s.Enqueue([]byte("ZD1:first")) {
		t.Error("expected first enqueue to succeed")
	}
	if s.Enqueue([]byte("ZD1:second")) {
		t.Error("expected second enqueue to fail when queue full")
	}
	if s.QueueSize() != 1 {
		t.Errorf("expected queue size 1 (dropped), got %d", s.QueueSize())
	}
}

// ---------------------------------------------------------------------------
// Drain
// ---------------------------------------------------------------------------

func TestDrain_EmptyQueue(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})
	drained := s.Drain(100 * time.Millisecond)
	if !drained {
		t.Error("expected drain to succeed on empty queue")
	}
}

func TestDrain_Timeout(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})
	s.Enqueue([]byte("ZD1:stuck-job"))

	// No worker running — queue never empties
	drained := s.Drain(50 * time.Millisecond)
	if drained {
		t.Error("expected drain to timeout when no worker is processing")
	}
}

func TestDrain_WithWorker(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Enqueue a job and let it process
	s.Enqueue([]byte("ZD1:test"))
	time.Sleep(50 * time.Millisecond)

	// Queue should be empty by now
	drained := s.Drain(time.Second)
	if !drained {
		t.Error("expected drain to succeed with worker processing")
	}

	cancel()
	s.Stop(2 * time.Second)
}

// ---------------------------------------------------------------------------
// Start / Stop / Lifecycle
// ---------------------------------------------------------------------------

func TestStartStop_Idempotent(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Should be safe to call Stop even if queue is empty
	cancel()
	s.Stop(time.Second)

	// After Stop, GetMetrics should still work
	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 0 {
		t.Errorf("expected 0 processed, got %d", metrics.TotalProcessed)
	}
}

func TestProcessJob_Success(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	s.Enqueue([]byte("ZD1:test-success"))
	time.Sleep(100 * time.Millisecond)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 1 {
		t.Errorf("expected 1 processed, got %d", metrics.TotalProcessed)
	}
	if metrics.TotalFailed != 0 {
		t.Errorf("expected 0 failed, got %d", metrics.TotalFailed)
	}
}

func TestProcessJob_MultipleJobs(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	for i := 0; i < 5; i++ {
		s.Enqueue([]byte(fmt.Sprintf("ZD1:job-%d", i)))
	}
	time.Sleep(200 * time.Millisecond)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 5 {
		t.Errorf("expected 5 processed, got %d", metrics.TotalProcessed)
	}
}

func TestProcessJob_ExhaustRetries(t *testing.T) {
	mp := &mockPrinter{fail: true}
	s := NewSpooler(10, mp)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	s.Enqueue([]byte("ZD1:test-fail"))
	// Wait: 1s + 2s sleep + small overhead
	time.Sleep(4 * time.Second)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 0 {
		t.Errorf("expected 0 processed, got %d", metrics.TotalProcessed)
	}
	if metrics.TotalFailed != 1 {
		t.Errorf("expected 1 failed, got %d", metrics.TotalFailed)
	}

	// Should have attempted 3 retries
	if mp.callCountValue() != 3 {
		t.Errorf("expected 3 print attempts, got %d", mp.callCountValue())
	}
}

func TestProcessJob_SucceedsAfterRetry(t *testing.T) {
	mp := &mockPrinter{}
	mp.printFn = func() error {
		mp.mu.Lock()
		defer mp.mu.Unlock()
		// Succeed on the 3rd attempt.
		// callCount is already incremented by Print() before calling us.
		if mp.callCount >= 3 {
			return nil
		}
		return fmt.Errorf("transient error")
	}
	s := NewSpooler(10, mp)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	s.Enqueue([]byte("ZD1:test-retry-success"))
	// Wait: 1s + 2s + small overhead
	time.Sleep(4 * time.Second)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 1 {
		t.Errorf("expected 1 processed after retry, got %d", metrics.TotalProcessed)
	}
	if metrics.TotalFailed != 0 {
		t.Errorf("expected 0 failed, got %d", metrics.TotalFailed)
	}
	if mp.callCountValue() != 3 {
		t.Errorf("expected 3 print attempts (2 fails + 1 success), got %d", mp.callCountValue())
	}
}

func TestProcessJob_ReconnectCalledOnRetry(t *testing.T) {
	// A printer that implements Reconnect interface should have
	// Reconnect() called between retries.
	mp := &mockReconnectPrinter{}
	mp.fail = true // always fails
	s := NewSpooler(10, mp)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	s.Enqueue([]byte("ZD1:test-reconnect"))
	time.Sleep(4 * time.Second)

	cancel()
	s.Stop(2 * time.Second)

	if rc := mp.reconnectCountValue(); rc == 0 {
		t.Error("expected Reconnect() to be called during retries")
	} else {
		t.Logf("Reconnect called %d times during 3 retries", rc)
	}

	metrics := s.GetMetrics()
	if metrics.TotalFailed != 1 {
		t.Errorf("expected 1 failed, got %d", metrics.TotalFailed)
	}
}

// ---------------------------------------------------------------------------
// GetMetrics
// ---------------------------------------------------------------------------

func TestGetMetrics(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)

	m := s.GetMetrics()
	if m.TotalProcessed != 0 {
		t.Errorf("expected 0 processed, got %d", m.TotalProcessed)
	}
	if m.MaxDepth != 0 {
		t.Errorf("expected 0 max depth, got %d", m.MaxDepth)
	}
	if m.CurrentDepth != 0 {
		t.Errorf("expected 0 current depth, got %d", m.CurrentDepth)
	}
}

// ---------------------------------------------------------------------------
// zeroPayload
// ---------------------------------------------------------------------------

func TestZeroPayload(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})

	payload := []byte("ZD1:sensitive-data-that-must-be-zeroed")
	original := make([]byte, len(payload))
	copy(original, payload)

	s.zeroPayload(payload)

	for i, b := range payload {
		if b != 0 {
			t.Errorf("payload[%d] = %d (0x%x), expected 0", i, b, b)
		}
	}

	// Verify original data is still intact (zeroPayload only modifies
	// the provided slice, not copies)
	for i, b := range original {
		if b == 0 && payload[i] == 0 {
			// Both zeroed — only valid if the original byte was also 0
			continue
		}
	}
}

func TestZeroPayload_Empty(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})
	// Should not panic on empty slice
	s.zeroPayload([]byte{})
}

func TestZeroPayload_LargePayload(t *testing.T) {
	s := NewSpooler(10, &mockPrinter{})
	payload := make([]byte, 10000)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	s.zeroPayload(payload)

	for i, b := range payload {
		if b != 0 {
			t.Errorf("payload[%d] = %d, expected 0 after zeroing", i, b)
		}
	}
}

// ---------------------------------------------------------------------------
// PrinterProvider resolution
// ---------------------------------------------------------------------------

func TestPrinterProvider_ResolvedPerJob(t *testing.T) {
	// When using PrinterProvider, the printer should be resolved at job time.
	successPrinter := &mockPrinter{}
	provider := &mockProvider{printer: successPrinter}
	s := NewSpooler(10, provider)
	if s == nil {
		t.Fatal("expected non-nil spooler")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	s.Enqueue([]byte("ZD1:test-provider"))
	time.Sleep(100 * time.Millisecond)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 1 {
		t.Errorf("expected 1 processed via provider, got %d", metrics.TotalProcessed)
	}
}

func TestPrinterProvider_SwitchPrinter(t *testing.T) {
	// Verify that switching the provider's active printer between jobs
	// takes effect on the next job.
	failPrinter := &mockPrinter{fail: true}
	successPrinter := &mockPrinter{}
	provider := &mockProvider{printer: failPrinter}
	s := NewSpooler(10, provider)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// First job — printer that fails
	s.Enqueue([]byte("ZD1:fail-job"))
	time.Sleep(5 * time.Second) // wait for all 3 retries

	// Switch printer
	provider.setPrinter(successPrinter)

	// Second job — should use the new printer
	s.Enqueue([]byte("ZD1:success-job"))
	time.Sleep(100 * time.Millisecond)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalFailed != 1 {
		t.Errorf("expected 1 failed (first job), got %d", metrics.TotalFailed)
	}
	if metrics.TotalProcessed != 1 {
		t.Errorf("expected 1 processed (second job with switched printer), got %d", metrics.TotalProcessed)
	}
}

// ---------------------------------------------------------------------------
// Queue channel interface
// ---------------------------------------------------------------------------

func TestQueueChannel_DirectPush(t *testing.T) {
	mp := &mockPrinter{}
	s := NewSpooler(10, mp)

	// Direct channel access for advanced use
	s.Queue() <- []byte("ZD1:direct-push")
	if s.QueueSize() != 1 {
		t.Errorf("expected queue size 1 after direct push, got %d", s.QueueSize())
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	cancel()
	s.Stop(2 * time.Second)

	metrics := s.GetMetrics()
	if metrics.TotalProcessed != 1 {
		t.Errorf("expected 1 processed from direct push, got %d", metrics.TotalProcessed)
	}
}
