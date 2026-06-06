package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/zerodrop/terminal/pkg/api"
	"github.com/zerodrop/terminal/pkg/config"
	"github.com/zerodrop/terminal/pkg/crypto"
	"github.com/zerodrop/terminal/pkg/observability"
	"github.com/zerodrop/terminal/pkg/printer"
	"github.com/zerodrop/terminal/pkg/spooler"
)

// findProjectRoot walks up from the current directory to locate go.mod,
// ensuring all relative paths resolve correctly regardless of where the
// binary is invoked from (project root, cmd/zerodrop/, or elsewhere).
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func main() {
	// Route all log.* output to stdout. Go's default log writes to stderr,
	// but Docker merges stdout and stderr with no ordering guarantee —
	// this causes log lines to interleave with the QR ASCII art output.
	// Single stream = guaranteed ordering.
	log.SetOutput(os.Stdout)

	log.Println("ZeroDrop Terminal v1.0 Starting...")

	if root := findProjectRoot(); root != "." {
		if err := os.Chdir(root); err != nil {
			log.Printf("WARNING: Could not chdir to project root %s: %v", root, err)
		} else {
			log.Printf("Working directory: %s", root)
		}
	}

	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using system environment variables")
	}

	// Load configuration from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Configuration loaded:")
	log.Printf("  Printer Type: %s", cfg.PrinterType)
	if cfg.PrinterDevice != "" {
		log.Printf("  Printer Device: %s", cfg.PrinterDevice)
	}
	log.Printf("  Rate Limit: %d req/hour (burst: %d)", cfg.RateLimitRequestsPerHour, cfg.RateLimitBurst)
	log.Printf("  Logging: %v", cfg.LogEnabled)
	log.Printf("  Public Key Path: %s", cfg.PublicKeyPath)
	log.Printf("  TLS Enabled: %v", cfg.TLSEnabled)

	// Initialize logger
	logger := observability.NewLogger(cfg.LogEnabled)
	logger.Info("ZeroDrop Terminal starting", map[string]interface{}{
		"version":      "1.0",
		"printer_type": cfg.PrinterType,
	})

	// Initialize or load key pair
	log.Println("\n=== Key Provisioning ===")
	keyPair, err := crypto.InitializeOrLoadKeyPair(
		cfg.PublicKeyPath,
		cfg.PrivateKeyPath,
		cfg.KeyRotate,
		cfg.LogEnabled,
	)
	if err != nil {
		log.Fatalf("Failed to initialize key pair: %v", err)
	}

	logger.Info("Key pair generated successfully", map[string]interface{}{
		"fingerprint": func() string {
			fp, _ := crypto.GetPublicKeyFingerprint(keyPair.PublicKey)
			return fp
		}(),
	})

	log.Println("\n=== Burn Protocol ===")
	if keyPair.PrivateKey != nil {
		log.Println("Executing Burn Protocol to destroy private key from memory...")
		crypto.BurnProtocol(keyPair)
		logger.Info("Burn Protocol complete", map[string]interface{}{
			"status": "private_key_destroyed",
		})
		log.Println("Burn Protocol complete. Private key has been destroyed from server memory.")
	} else {
		log.Println("Skipped — private key not loaded (reusing existing key pair from disk).")
	}

	// Create printer based on configuration
	var printImpl printer.Printer
	var printInterface interface{} // For extended interfaces (HealthChecker, etc.)
	var pm *printer.PrinterManager

	switch cfg.PrinterType {
	case "mock":
		printImpl = printer.NewMockPrinter()
		printInterface = printImpl
		pm = printer.NewPrinterManager(printImpl, printer.PrinterInfo{
			ID: "mock", Name: "Mock Printer (stdout)", Type: "mock",
		})
		logger.Info("Mock printer initialized", nil)

	case "usb":
		log.Println("\n=== USB Printer Initialization ===")
		usbPrinter, err := printer.NewUSBPrinter(cfg.PrinterDevice)
		if err != nil {
			log.Printf("USB device node not found: %v", err)

			// Try raw USB access (bypasses usblp for chips like Zjiang 0fe6:811e)
			log.Println("Trying raw USB access (direct bulk transfer)...")
			rawPrinter, rawErr := printer.DetectRawUSBPrinter()
			if rawErr != nil {
				log.Printf("Raw USB printer not found: %v", rawErr)
				log.Println("Falling back to Mock Printer...")
				printImpl = printer.NewMockPrinter()
				printInterface = printImpl
				pm = printer.NewPrinterManager(printImpl, printer.PrinterInfo{
					ID: "mock", Name: "Mock Printer (stdout)", Type: "mock",
				})
			} else {
				printImpl = rawPrinter
				printInterface = rawPrinter
				devicePath := rawPrinter.GetDevicePath()
				pm = printer.NewPrinterManager(printImpl, printer.PrinterInfo{
					ID: devicePath, Name: "Raw USB Printer", Type: "usb", Device: devicePath,
				})
				log.Printf("Raw USB printer initialized at %s (ep %s)",
					devicePath, rawPrinter.HealthCheck()["endpoint"])
			}
		} else {
			printImpl = usbPrinter
			printInterface = usbPrinter
			devicePath := usbPrinter.GetDevicePath()
			pm = printer.NewPrinterManager(printImpl, printer.PrinterInfo{
				ID: devicePath, Name: "USB Printer", Type: "usb", Device: devicePath,
			})
			log.Printf("USB printer initialized at %s", devicePath)
		}

	default:
		log.Fatalf("Unknown printer type: %s", cfg.PrinterType)
	}

	// Detect all printers for admin panel
	pm.Detect()

	// Create spooler with queue size of 10
	splr := spooler.NewSpooler(10, printImpl)

	// Create API server with printer for health checks
	server := api.NewServer(cfg, splr.Queue(), printInterface)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start spooler worker
	splr.Start(ctx)

	// Start API server in background — HTTPS if TLS enabled, HTTP otherwise
	if cfg.TLSEnabled {
		log.Println("TLS enabled: generating self-signed certificate...")
		certPEM, keyPEM, err := crypto.GenerateSelfSignedCert()
		if err != nil {
			log.Fatalf("Failed to generate self-signed certificate: %v", err)
		}
		log.Println("Self-signed certificate generated.")
		go func() {
			if err := server.ServeTLS(":8080", certPEM, keyPEM); err != nil {
				logger.Error("HTTPS server failed", map[string]interface{}{
					"error": err.Error(),
				})
				cancel()
			}
		}()
	} else {
		go func() {
			if err := server.Start(":8080"); err != nil {
				logger.Error("API server failed", map[string]interface{}{
					"error": err.Error(),
				})
				cancel()
			}
		}()
	}

	if cfg.TLSEnabled {
		log.Println("\n=== ZeroDrop Terminal Ready (HTTPS) ===")
		log.Printf("HTTPS server listening on :8080 (self-signed certificate)")
		log.Printf("WARNING: Browsers will show a security warning for self-signed certs.")
		log.Printf("         Click 'Advanced' -> 'Proceed to site' to continue.")
	} else {
		log.Println("\n=== ZeroDrop Terminal Ready ===")
		log.Printf("API server listening on :8080")
	}
	log.Printf("Endpoints:")
	log.Printf("  GET  /key     - Retrieve public key")
	log.Printf("  POST /drop    - Submit encrypted payload")
	log.Printf("  GET  /health  - Health check (includes printer status)")
	log.Println("")
	log.Println("IMPORTANT: Ensure you have saved the private key QR code from above.")
	log.Println("The server can no longer decrypt messages.")
	log.Println("")
	log.Println("Press Ctrl+C to gracefully shutdown...")

	// Setup graceful shutdown
	shutdownHandler := observability.NewShutdownHandler(splr, 30*time.Second) // 30 second timeout

	// Wait for shutdown signal
	shutdownHandler.WaitForSignal(syscall.SIGINT, syscall.SIGTERM)

	// Initiate graceful shutdown
	shutdownHandler.Shutdown()

	log.Println("ZeroDrop Terminal stopped.")
}
