package spooler

import (
	"sync"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m.StartTime.IsZero() {
		t.Error("expected StartTime to be set")
	}
	snap := m.Snapshot()
	if snap.TotalProcessed != 0 {
		t.Errorf("expected 0 processed initially, got %d", snap.TotalProcessed)
	}
	if snap.TotalFailed != 0 {
		t.Errorf("expected 0 failed initially, got %d", snap.TotalFailed)
	}
}

func TestRecordSuccess(t *testing.T) {
	m := NewMetrics()
	start := time.Now()
	m.recordSuccess(500 * time.Millisecond)

	if m.TotalProcessed != 1 {
		t.Errorf("expected TotalProcessed=1, got %d", m.TotalProcessed)
	}
	if m.LastPrintTime.Before(start) {
		t.Error("expected LastPrintTime to be after start")
	}
	if m.LastPrintDuration != 500*time.Millisecond {
		t.Errorf("expected LastPrintDuration=500ms, got %v", m.LastPrintDuration)
	}

	// Multiple successes
	m.recordSuccess(time.Second)
	if m.TotalProcessed != 2 {
		t.Errorf("expected TotalProcessed=2, got %d", m.TotalProcessed)
	}
}

func TestRecordFailure(t *testing.T) {
	m := NewMetrics()
	m.recordFailure()

	if m.TotalFailed != 1 {
		t.Errorf("expected TotalFailed=1, got %d", m.TotalFailed)
	}

	// Multiple failures
	m.recordFailure()
	m.recordFailure()
	if m.TotalFailed != 3 {
		t.Errorf("expected TotalFailed=3, got %d", m.TotalFailed)
	}
}

func TestUpdateDepth(t *testing.T) {
	m := NewMetrics()

	// Initial depth
	m.updateDepth(5)
	if m.CurrentDepth != 5 {
		t.Errorf("expected CurrentDepth=5, got %d", m.CurrentDepth)
	}
	if m.MaxDepth != 5 {
		t.Errorf("expected MaxDepth=5, got %d", m.MaxDepth)
	}

	// Lower depth — CurrentDepth goes down, MaxDepth stays
	m.updateDepth(3)
	if m.CurrentDepth != 3 {
		t.Errorf("expected CurrentDepth=3, got %d", m.CurrentDepth)
	}
	if m.MaxDepth != 5 {
		t.Errorf("expected MaxDepth=5 (should not decrease), got %d", m.MaxDepth)
	}

	// Higher depth — both go up
	m.updateDepth(10)
	if m.CurrentDepth != 10 {
		t.Errorf("expected CurrentDepth=10, got %d", m.CurrentDepth)
	}
	if m.MaxDepth != 10 {
		t.Errorf("expected MaxDepth=10, got %d", m.MaxDepth)
	}
}

func TestSnapshot_IsIndependentCopy(t *testing.T) {
	m := NewMetrics()
	m.recordSuccess(time.Second)
	m.recordFailure()
	m.updateDepth(3)

	snap := m.Snapshot()

	// Snapshot should reflect current state
	if snap.TotalProcessed != 1 {
		t.Errorf("snapshot TotalProcessed=1, got %d", snap.TotalProcessed)
	}
	if snap.TotalFailed != 1 {
		t.Errorf("snapshot TotalFailed=1, got %d", snap.TotalFailed)
	}
	if snap.CurrentDepth != 3 {
		t.Errorf("snapshot CurrentDepth=3, got %d", snap.CurrentDepth)
	}

	// Mutate original — snapshot must remain unchanged
	m.recordSuccess(time.Second)
	m.recordFailure()
	m.updateDepth(5)

	if snap.TotalProcessed != 1 {
		t.Error("snapshot should be an independent copy, not a reference")
	}
	if snap.TotalFailed != 1 {
		t.Error("snapshot should be an independent copy, not a reference")
	}
}

func TestSnapshot_Fields(t *testing.T) {
	m := NewMetrics()

	// Snapshot of zero state
	snap := m.Snapshot()
	if snap.TotalProcessed != 0 {
		t.Errorf("expected 0, got %d", snap.TotalProcessed)
	}
	if snap.TotalFailed != 0 {
		t.Errorf("expected 0, got %d", snap.TotalFailed)
	}
	if snap.CurrentDepth != 0 {
		t.Errorf("expected 0, got %d", snap.CurrentDepth)
	}
	if snap.MaxDepth != 0 {
		t.Errorf("expected 0, got %d", snap.MaxDepth)
	}
	if !snap.StartTime.IsZero() {
		t.Logf("StartTime is set: %v", snap.StartTime)
	}
}

func TestConcurrentAccess_NoRace(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup

	// 10 concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.recordSuccess(time.Millisecond)
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.recordFailure()
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.updateDepth(5)
		}()
	}

	// 5 concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Snapshot()
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	snap := m.Snapshot()
	totalOps := snap.TotalProcessed + snap.TotalFailed
	if totalOps == 0 {
		t.Error("expected some operations to be recorded")
	}
	t.Logf("Concurrent test: processed=%d failed=%d depth=%d maxdepth=%d",
		snap.TotalProcessed, snap.TotalFailed, snap.CurrentDepth, snap.MaxDepth)
}

func TestMixedReadWriteUnderLoad(t *testing.T) {
	m := NewMetrics()
	var wg sync.WaitGroup

	// Continuous writes + reads in parallel
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				m.recordSuccess(time.Millisecond)
				m.updateDepth(3)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = m.Snapshot()
			}
		}()
	}

	wg.Wait()

	snap := m.Snapshot()
	if snap.TotalProcessed != 1000 {
		t.Errorf("expected 1000 processed (20*50), got %d", snap.TotalProcessed)
	}
}
