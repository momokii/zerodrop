package spooler

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/zerodrop/terminal/pkg/printer"
)

// Spooler manages the print job queue and processes jobs sequentially
type Spooler struct {
	queue      chan []byte
	workerDone chan struct{}
	getPrinter func() printer.Printer
	metrics    *Metrics
}

// PrinterProvider returns the current active printer. Used by the spooler
// to resolve the printer at job time so admin printer switching takes effect.
type PrinterProvider interface {
	GetActive() printer.Printer
}

// NewSpooler creates a new spooler with the given queue size and printer provider.
// Accepts either a PrinterProvider (e.g., PrinterManager) or a static Printer.
func NewSpooler(queueSize int, provider interface{}) *Spooler {
	s := &Spooler{
		queue:      make(chan []byte, queueSize),
		workerDone: make(chan struct{}),
		metrics:    NewMetrics(),
	}
	if pp, ok := provider.(PrinterProvider); ok {
		s.getPrinter = pp.GetActive
	} else if p, ok := provider.(printer.Printer); ok {
		s.getPrinter = func() printer.Printer { return p }
	} else {
		panic("NewSpooler: provider must implement printer.Printer or PrinterProvider")
	}
	return s
}

// GetMetrics returns a snapshot of the current spooler metrics.
func (s *Spooler) GetMetrics() Metrics {
	return s.metrics.Snapshot()
}

// Start begins processing print jobs from the queue
func (s *Spooler) Start(ctx context.Context) {
	log.Println("Spooler worker started")

	go func() {
		defer close(s.workerDone)
		for {
			select {
			case <-ctx.Done():
				log.Println("Spooler worker shutting down...")
				return
			case payload, ok := <-s.queue:
				if !ok {
					log.Println("Spooler worker shutting down...")
					return
				}
				s.metrics.updateDepth(len(s.queue))
				s.processJob(payload)
			}
		}
	}()
}

// processJob handles a single print job with retry logic
func (s *Spooler) processJob(payload []byte) {
	log.Printf("Processing print job (payload size: %d bytes)", len(payload))

	start := time.Now()
	printer := s.getPrinter()

	// Retry logic: up to 3 attempts with exponential backoff
	maxRetries := 3
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := printer.Print(payload)
		if err == nil {
			log.Printf("Print job completed successfully")
			s.metrics.recordSuccess(time.Since(start))
			// Memory zeroing: clear the payload buffer
			s.zeroPayload(payload)
			return
		}

		log.Printf("Print attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			log.Printf("Retrying in %v...", backoff)

			// Re-resolve printer in case admin switched it between retries
			printer = s.getPrinter()

			// If the printer supports reconnection, attempt it before retry.
			if reconnector, ok := printer.(interface{ Reconnect() error }); ok {
				if recErr := reconnector.Reconnect(); recErr != nil {
					log.Printf("Reconnect before retry failed: %v", recErr)
				}
			}

			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		} else {
			s.metrics.recordFailure()
			log.Printf("Print job failed after %d attempts", maxRetries)
		}
	}
}

// zeroPayload securely zeros the payload buffer after printing
func (s *Spooler) zeroPayload(payload []byte) {
	// Explicitly zero each byte
	for i := range payload {
		payload[i] = 0
	}
	// Use runtime.KeepAlive to prevent compiler optimization
	runtime.KeepAlive(payload)
	log.Printf("Payload buffer zeroed from memory")
}

// Enqueue adds a payload to the print queue
func (s *Spooler) Enqueue(payload []byte) bool {
	select {
	case s.queue <- payload:
		return true
	default:
		return false
	}
}

// Queue returns the queue channel for direct pushing
func (s *Spooler) Queue() chan []byte {
	return s.queue
}

// Drain waits for the queue to empty, with a timeout
func (s *Spooler) Drain(timeout time.Duration) bool {
	log.Printf("Draining spooler (timeout: %v)...", timeout)

	deadline := time.After(timeout)
	queueEmpty := false

	for {
		select {
		case <-deadline:
			log.Printf("Spooler drain timeout")
			return queueEmpty
		default:
			if len(s.queue) == 0 {
				queueEmpty = true
				return true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Stop gracefully shuts down the spooler
func (s *Spooler) Stop(timeout time.Duration) {
	log.Printf("Stopping spooler...")

	// First drain the queue
	drained := s.Drain(timeout)
	if !drained {
		log.Printf("WARNING: Spooler did not drain completely within timeout")
	}

	// Close the queue to stop accepting new jobs
	close(s.queue)

	// Wait for worker to finish
	select {
	case <-s.workerDone:
		log.Printf("Spooler stopped cleanly")
	case <-time.After(timeout):
		log.Printf("WARNING: Spooler worker did not stop within timeout")
	}
}

// QueueSize returns the current number of items in the queue
func (s *Spooler) QueueSize() int {
	return len(s.queue)
}
