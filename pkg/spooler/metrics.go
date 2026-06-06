package spooler

import (
	"sync"
	"time"
)

// Metrics holds runtime statistics about the print spooler.
type Metrics struct {
	mu                sync.RWMutex
	TotalProcessed    int64
	TotalFailed       int64
	CurrentDepth      int
	MaxDepth          int
	LastPrintTime     time.Time
	LastPrintDuration time.Duration
	StartTime         time.Time
}

// NewMetrics creates a Metrics instance with StartTime set to now.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// Snapshot returns a copy of the current metrics for safe consumption.
func (m *Metrics) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Metrics{
		TotalProcessed:    m.TotalProcessed,
		TotalFailed:       m.TotalFailed,
		CurrentDepth:      m.CurrentDepth,
		MaxDepth:          m.MaxDepth,
		LastPrintTime:     m.LastPrintTime,
		LastPrintDuration: m.LastPrintDuration,
		StartTime:         m.StartTime,
	}
}

func (m *Metrics) recordSuccess(dur time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalProcessed++
	m.LastPrintTime = time.Now()
	m.LastPrintDuration = dur
}

func (m *Metrics) recordFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalFailed++
}

func (m *Metrics) updateDepth(depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentDepth = depth
	if depth > m.MaxDepth {
		m.MaxDepth = depth
	}
}
