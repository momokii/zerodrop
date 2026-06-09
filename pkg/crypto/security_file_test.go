package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zerodrop/terminal/pkg/config"
)

func TestSecurity_File_PublicKeyFile_Permissions0644(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "public_key.pem")

	err = SavePublicKeyToFile(kp.PublicKey, pubPath)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(pubPath)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("[SECURITY] Public key file permissions: expected 0644, got %04o", perm)
	}
	t.Logf("[SECURITY] Public key file permissions 0644: PASS")
}

func TestSecurity_Config_DefaultLoggingOff(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.LogEnabled {
		t.Error("[SECURITY] Default config should have logging disabled")
	}
	t.Logf("[SECURITY] Default config: logging OFF: PASS")
}

func TestSecurity_Config_DefaultRateLimitPositive(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.RateLimitRequestsPerHour < 1 {
		t.Errorf("[SECURITY] Default rate limit must be >= 1, got %d", cfg.RateLimitRequestsPerHour)
	}
	if cfg.RateLimitBurst < 1 {
		t.Errorf("[SECURITY] Default rate limit burst must be >= 1, got %d", cfg.RateLimitBurst)
	}
	t.Logf("[SECURITY] Default rate limit (%d req/hr, burst %d): PASS", cfg.RateLimitRequestsPerHour, cfg.RateLimitBurst)
}

func TestSecurity_Config_DefaultSecureValues(t *testing.T) {
	cfg := config.DefaultConfig()

	// Printer type defaults to mock (safe for dev)
	if cfg.PrinterType != "mock" {
		t.Errorf("[SECURITY] Default printer type should be 'mock', got %q", cfg.PrinterType)
	}

	// TLS off by default
	if cfg.TLSEnabled {
		t.Error("[SECURITY] TLS should be off by default")
	}

	// Key rotation off by default (preserves existing keys)
	if cfg.KeyRotate {
		t.Error("[SECURITY] Key rotation should be off by default")
	}

	// Admin token empty by default (admin disabled until explicitly configured)
	if cfg.AdminToken != "" {
		t.Error("[SECURITY] Admin token should be empty by default (disabled)")
	}

	t.Logf("[SECURITY] All default config values are secure: PASS")
}

func TestSecurity_KeyReuse_SameFingerprint_NilPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "public_key.pem")
	privPath := filepath.Join(tmpDir, "private_key.pem")

	// First run: generate keys
	kp1, err := InitializeOrLoadKeyPair(pubPath, privPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	fp1, _ := GetPublicKeyFingerprint(kp1.PublicKey)

	// Simulate Burn Protocol (as main.go does)
	BurnProtocol(kp1)

	// Second run: reload existing keys (simulating server restart)
	kp2, err := InitializeOrLoadKeyPair(pubPath, privPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	fp2, _ := GetPublicKeyFingerprint(kp2.PublicKey)

	// Fingerprints must match
	if fp1 != fp2 {
		t.Errorf("[SECURITY] Fingerprints differ after reload:\n  first:  %s\n  reload: %s", fp1, fp2)
	}

	// Private key must be nil on reload
	if kp2.PrivateKey != nil {
		t.Error("[SECURITY] Private key should be nil on reload (never loaded into RAM)")
	}

	t.Logf("[SECURITY] Key reuse: same fingerprint, private key nil on reload: PASS")
}
