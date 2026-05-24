package spooler

import (
	"context"
	"log"
	"runtime"
	"time"
)

// Spooler manages the print job queue and processes jobs sequentially
type Spooler struct {
	queue      chan []byte
	workerDone chan struct{}
	printer    Printer
}

// Printer interface defines the contract for printing payloads
type Printer interface {
	Print(ciphertext []byte) error
	IsAvailable() bool
}

// NewSpooler creates a new spooler with the given queue size and printer
func NewSpooler(queueSize int, printer Printer) *Spooler {
	return &Spooler{
		queue:      make(chan []byte, queueSize),
		workerDone: make(chan struct{}),
		printer:    printer,
	}
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
				s.processJob(payload)
			}
		}
	}()
}

// processJob handles a single print job with retry logic
func (s *Spooler) processJob(payload []byte) {
	log.Printf("Processing print job (payload size: %d bytes)", len(payload))

	// Retry logic: up to 3 attempts with exponential backoff
	maxRetries := 3
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := s.printer.Print(payload)
		if err == nil {
			log.Printf("Print job completed successfully")
			// Memory zeroing: clear the payload buffer
			s.zeroPayload(payload)
			return
		}

		log.Printf("Print attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			log.Printf("Retrying in %v...", backoff)
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		} else {
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
